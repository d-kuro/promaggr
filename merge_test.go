package promaggr_test

import (
	"bytes"
	"sort"
	"testing"

	"github.com/d-kuro/promaggr"
	"github.com/d-kuro/promaggr/internal"
	"github.com/google/go-cmp/cmp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

func TestMergeMetricFamily(t *testing.T) {
	t.Parallel()

	const (
		counterMetricName = "dummy_counter_metric"
		gaugeMetricName   = "dummy_gauge_metric"
	)

	// The two are the same metric.
	mfs1 := []*dto.MetricFamily{
		internal.NewCounterMetricFamilyFixture(counterMetricName),
		internal.NewGaugeMetricFamilyFixture(gaugeMetricName),
	}
	mfs2 := []*dto.MetricFamily{
		internal.NewCounterMetricFamilyFixture(counterMetricName),
		internal.NewGaugeMetricFamilyFixture(gaugeMetricName),
	}

	mfs := promaggr.MergeMetricFamily(mfs1, mfs2, promaggr.AddIdentifierLabel(mfs1, mfs2, "cluster_name", "foo", "bar"))

	if len(mfs) != 2 {
		t.Fatalf("mismatch in the number of metrics after merging: want(%d) got(%d)", 2, len(mfs))
	}

	out := bytes.Buffer{}

	sort.Slice(mfs, func(i, j int) bool {
		return mfs[i].GetName() < mfs[j].GetName()
	})

	for _, mf := range mfs {
		if _, err := expfmt.MetricFamilyToText(&out, mf); err != nil {
			t.Fatalf("failed to convert MetricFamily to text: %v", err)
		}
	}

	got := out.String()
	want := `# HELP dummy_counter_metric Dummy text.
# TYPE dummy_counter_metric counter
dummy_counter_metric{cluster_name="foo"} 123456
dummy_counter_metric{cluster_name="bar"} 123456
# HELP dummy_gauge_metric Dummy text.
# TYPE dummy_gauge_metric gauge
dummy_gauge_metric{cluster_name="foo"} 123.456
dummy_gauge_metric{cluster_name="bar"} 123.456
`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("MergeMetricFamily() mismatch (-want +got):\n%s", diff)
	}
}
