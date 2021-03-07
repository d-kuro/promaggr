package promaggr

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// ScraperOption is a functional option used by the NewScraper.
type ScraperOption func(*Scraper)

// Scraper will scrape metrics from prometheus exporter.
type Scraper struct {
	// URL is the URL of the scraping target.
	URL string

	// Labels is a set of labels to be added to the scraped metrics.
	// If not specified, nothing will be added.
	Labels model.LabelSet

	// HTTPClient is the http.Client to be used for the request. If not specified, the http.DefaultClient will be used.
	HTTPClient *http.Client
}

// NewScraper creates and returns a new Scraper.
func NewScraper(url string, opts ...ScraperOption) *Scraper {
	scraper := &Scraper{
		URL: url,
	}

	for _, o := range opts {
		o(scraper)
	}

	return scraper
}

// Labels is an option available for NewScraper.
// This will be used to add labels to the scraped metrics.
func Labels(labelSets model.LabelSet) ScraperOption {
	return func(s *Scraper) {
		s.Labels = labelSets
	}
}

// HTTPClient is an option available for NewScraper.
// Override the http.DefaultClient.
func HTTPClient(client *http.Client) ScraperOption {
	return func(s *Scraper) {
		s.HTTPClient = client
	}
}

// Scrape.
func (s *Scraper) Scrape(ctx context.Context) ([]*dto.MetricFamily, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var client *http.Client

	if s.HTTPClient != nil {
		client = s.HTTPClient
	} else {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request to %s: %w", s.URL, err)
	}

	defer resp.Body.Close()

	var parser expfmt.TextParser

	parsed, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metric: %w", err)
	}

	mfs := make([]*dto.MetricFamily, 0, len(parsed))

	for _, mf := range parsed {
		mfs = append(mfs, mf)
	}

	if s.Labels != nil {
		AddLabels(mfs, s.Labels)
	}

	return mfs, nil
}

var _ prometheus.Collector = &Collector{}

// Collector implements the prometheus.Collector interface.
type Collector struct {
	//
	Scrapers []*Scraper

	// Logger is a logger that implements the logr.Logger interface.
	// If it is not specified, nothing will be logged.
	Logger logr.Logger

	once  sync.Once
	mutex sync.RWMutex
	cache []*dto.MetricFamily
}

// CollectorOption is a functional option used by the NewCollector.
type CollectorOption func(*Collector)

// NewCollector will create and return a new Collector.
func NewCollector(scrapers []*Scraper, opts ...CollectorOption) *Collector {
	collector := &Collector{
		Scrapers: scrapers,
	}

	for _, o := range opts {
		o(collector)
	}

	return collector
}

// Logger is an option available for NewCollector.
// You can set a logger that implements the logr.Logger interface.
// If a logger is set, an error will be output to the log.
// If the logger is not set, nothing will be output.
func Logger(logger logr.Logger) CollectorOption {
	return func(c *Collector) {
		c.Logger = logger
	}
}

// Describe implements the prometheus.Collector interface.
// Register prometheus.Desc.
// It is called at registration time and is used to avoid duplicate registration of metrics.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	c.once.Do(func() {
		c.rsyncCache(context.Background())
	})

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, mf := range c.cache {
		ch <- MetricFamilyToDesc(mf)
	}
}

// Collect implements the prometheus.Collector interface.
// Collect and register metrics.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.rsyncCache(context.Background())

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, mf := range c.cache {
		for _, metric := range MetricFamilyToMetrics(mf) {
			ch <- metric
		}
	}
}

// rsyncCache will update the scrape results of the metrics kept by the Collector.
// Use goroutine to scrape from multiple prometheus exporter and merge the results.
func (c *Collector) rsyncCache(ctx context.Context) {
	var wg sync.WaitGroup

	mfsCh := make(chan []*dto.MetricFamily)
	errCh := make(chan error)
	done := make(chan struct{})

	newMfs := make([]*dto.MetricFamily, 0)

	go func() {
		for mfs := range mfsCh {
			newMfs = MergeMetricFamily(newMfs, mfs)
		}

		close(done)
	}()

	go func() {
		for err := range errCh {
			if c.Logger != nil {
				c.Logger.Error(err, "failed to scrape prometheus exporter")
			}
		}
	}()

	for _, scraper := range c.Scrapers {
		scraper := scraper

		wg.Add(1)

		go func() {
			defer wg.Done()

			mfs, err := scraper.Scrape(ctx)
			if err != nil {
				errCh <- err

				return
			}

			mfsCh <- mfs
		}()
	}

	wg.Wait()
	close(mfsCh)
	close(errCh)

	<-done

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache = newMfs
}
