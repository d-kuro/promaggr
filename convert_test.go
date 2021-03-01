package promaggr_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/d-kuro/promaggr"
	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

const metricsText = `# HELP dummy_counter_metric Dummy text.
# TYPE dummy_counter_metric counter
dummy_counter_metric{name="foo"} 123456
# HELP dummy_gauge_metric Dummy text.
# TYPE dummy_gauge_metric gauge
dummy_gauge_metric{name="foo"} 123.456
# HELP dummy_histogram_metric Dummy text.
# TYPE dummy_histogram_metric histogram
dummy_histogram_metric_bucket{name="foo",le="0.001"} 0
dummy_histogram_metric_bucket{name="foo",le="0.01"} 0
dummy_histogram_metric_bucket{name="foo",le="0.1"} 180
dummy_histogram_metric_bucket{name="foo",le="1"} 184
dummy_histogram_metric_bucket{name="foo",le="10"} 184
dummy_histogram_metric_bucket{name="foo",le="+Inf"} 184
dummy_histogram_metric_sum{name="foo"} 10.544979995999995
dummy_histogram_metric_count{name="foo"} 184
# HELP dummy_summary_metric Dummy text.
# TYPE dummy_summary_metric summary
dummy_summary_metric{quantile="0"} 4.4276e-05
dummy_summary_metric{quantile="0.25"} 5.2031e-05
dummy_summary_metric{quantile="0.5"} 7.3375e-05
dummy_summary_metric{quantile="0.75"} 8.3761e-05
dummy_summary_metric{quantile="1"} 0.002849601
dummy_summary_metric_sum 0.042124416
dummy_summary_metric_count 461
`

func TestMetricFamilyToDesc(t *testing.T) {
	t.Parallel()

	const (
		metricName  = "dummy_counter_metric"
		metricHelp  = "Dummy text."
		metricLabel = "name"
		metricText  = `# HELP dummy_counter_metric Dummy text.
# TYPE dummy_counter_metric counter
dummy_counter_metric{name="foo"} 123456
dummy_counter_metric{name="bar"} 123456
`
	)

	var parser expfmt.TextParser

	parsed, err := parser.TextToMetricFamilies(strings.NewReader(metricText))
	if err != nil {
		t.Fatalf("failed to parse prometheus metrics: %v", err)
	}

	got := promaggr.MetricFamilyToDesc(parsed[metricName])
	want := prometheus.NewDesc(metricName, metricHelp, []string{metricLabel}, nil)

	if diff := cmp.Diff(want.String(), got.String()); diff != "" {
		t.Errorf("prometheus.Desc mismatch (-want +got):\n%s", diff)
	}
}

var _ prometheus.Collector = &testCollector{}

type testCollector struct {
	mfs []*dto.MetricFamily
}

func (c *testCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, mf := range c.mfs {
		ch <- promaggr.MetricFamilyToDesc(mf)
	}
}

func (c *testCollector) Collect(ch chan<- prometheus.Metric) {
	for _, mf := range c.mfs {
		for _, metric := range promaggr.MetricFamilyToMetrics(mf) {
			ch <- metric
		}
	}
}

func TestMetricFamilyToMetrics(t *testing.T) {
	t.Parallel()

	var parser expfmt.TextParser

	parsed, err := parser.TextToMetricFamilies(strings.NewReader(metricsText))
	if err != nil {
		t.Fatalf("failed to parse prometheus metrics: %v", err)
	}

	mfs := make([]*dto.MetricFamily, 0, len(parsed))
	for _, mf := range parsed {
		mfs = append(mfs, mf)
	}

	collector := &testCollector{mfs: mfs}
	registry := prometheus.NewRegistry()

	registry.MustRegister(collector)
	defer registry.Unregister(collector)

	promServer := httptest.NewServer(promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	defer promServer.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, promServer.URL, nil)
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

	want := metricsText
	got := string(body)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("prometheus metrics mismatch (-want +got):\n%s", diff)
	}
}
