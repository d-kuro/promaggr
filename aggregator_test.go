package promaggr_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/d-kuro/promaggr"
	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
)

func TestCollector(t *testing.T) {
	t.Parallel()

	scrapeTargetCounter1 := newHTTPRequestCounter()
	scrapeTargetRegistry1 := prometheus.NewRegistry()
	scrapeTargetRegistry1.MustRegister(scrapeTargetCounter1)
	scrapeTargetCounter1.WithLabelValues("200", http.MethodGet).Inc()

	scrapeTarget1 := httptest.NewServer(promhttp.HandlerFor(scrapeTargetRegistry1, promhttp.HandlerOpts{}))
	defer scrapeTarget1.Close()

	scrapeTargetCounter2 := newHTTPRequestCounter()
	scrapeTargetRegistry2 := prometheus.NewRegistry()
	scrapeTargetRegistry2.MustRegister(scrapeTargetCounter2)
	scrapeTargetCounter2.WithLabelValues("200", http.MethodGet).Inc()

	scrapeTarget2 := httptest.NewServer(promhttp.HandlerFor(scrapeTargetRegistry2, promhttp.HandlerOpts{}))
	defer scrapeTarget2.Close()

	scrapers := []*promaggr.Scraper{
		promaggr.NewScraper(scrapeTarget1.URL, promaggr.Labels(map[model.LabelName]model.LabelValue{"cluster": "foo"})),
		promaggr.NewScraper(scrapeTarget2.URL, promaggr.Labels(map[model.LabelName]model.LabelValue{"cluster": "bar"})),
	}

	collector := promaggr.NewCollector(scrapers)
	aggregatorRegistry := prometheus.NewRegistry()
	aggregatorRegistry.MustRegister(collector)

	aggregator := httptest.NewServer(promhttp.HandlerFor(aggregatorRegistry, promhttp.HandlerOpts{}))
	defer aggregator.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, aggregator.URL, nil)
	if err != nil {
		t.Fatalf("failed to create new request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP request to prometheus expoter failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read the response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code from prometheus exporter: %v", resp.StatusCode)
	}

	got := string(body)
	want := `# HELP http_requests_total Dummy text.
# TYPE http_requests_total counter
http_requests_total{cluster="bar",code="200",method="GET"} 1
http_requests_total{cluster="foo",code="200",method="GET"} 1
`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("prometheus metrics mismatch (-want +got):\n%s", diff)
	}
}

func newHTTPRequestCounter() *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Dummy text.",
		},
		[]string{"code", "method"},
	)
}
