package promaggr

import (
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"
)

// labelSet is a type for adding your own methods to prometheus model.LabelSet:
// toLabelNameSlice(), toLabelValueSlice().
type labelSet model.LabelSet

// newLabelSet converts a slice of LabelPair to a LabelSet.
func newLabelSet(labels []*dto.LabelPair) labelSet {
	labelSet := make(labelSet, len(labels))
	for _, l := range labels {
		labelSet[model.LabelName(l.GetName())] = model.LabelValue(l.GetValue())
	}

	return labelSet
}

// toLabelNameSlice returns a slice of label name from labelSet.
// The slice will be sorted lexicographically by label name.
func (s labelSet) toLabelNameSlice() []string {
	labelNames := make([]string, 0, len(s))
	for labelName := range s {
		labelNames = append(labelNames, string(labelName))
	}

	sort.Strings(labelNames)

	return labelNames
}

// toLabelNameSlice returns a slice of label value from labelSet.
// The slice will be sorted lexicographically by label name.
func (s labelSet) toLabelValueSlice() []string {
	// To return results sorted by label name,
	// we first extract and sort only the label name.
	labelNames := make([]string, 0, len(s))
	for labelName := range s {
		labelNames = append(labelNames, string(labelName))
	}

	sort.Strings(labelNames)

	// Create a slice of the label value based on the sorted label name.
	labelValues := make([]string, 0, len(s))
	for _, labelName := range labelNames {
		labelValues = append(labelValues, string(s[model.LabelName(labelName)]))
	}

	return labelValues
}

// MetricFamilyToDesc generates prometheus.Desc from MetricFamily.
func MetricFamilyToDesc(f *dto.MetricFamily) *prometheus.Desc {
	labelSet := newLabelSet(f.Metric[0].GetLabel())

	return prometheus.NewDesc(f.GetName(), f.GetHelp(), labelSet.toLabelNameSlice(), nil)
}

// MetricFamilyToMetrics generates slice of prometheus.Metric slice from MetricFamily.
func MetricFamilyToMetrics(f *dto.MetricFamily) []prometheus.Metric {
	metrics := make([]prometheus.Metric, 0, len(f.Metric))

	for _, metric := range f.GetMetric() {
		desc := MetricFamilyToDesc(f)
		metrics = append(metrics, convertMetric(f.GetType(), desc, metric))
	}

	return metrics
}

// convertMetric converts the metric to a type that satisfies the prometheus.Metric interface.
// This allows you to export metrics by implementing the prometheus.Collector interface with the transformed metrics.
// Using an unsupported metric type will cause panic.
func convertMetric(metricType dto.MetricType, desc *prometheus.Desc, metric *dto.Metric) prometheus.Metric {
	labelSet := newLabelSet(metric.GetLabel())

	switch metricType {
	case dto.MetricType_COUNTER:
		return prometheus.MustNewConstMetric(desc, prometheus.CounterValue, metric.Counter.GetValue(), labelSet.toLabelValueSlice()...)
	case dto.MetricType_GAUGE:
		return prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, metric.Gauge.GetValue(), labelSet.toLabelValueSlice()...)
	case dto.MetricType_UNTYPED:
		return prometheus.MustNewConstMetric(desc, prometheus.UntypedValue, metric.Untyped.GetValue(), labelSet.toLabelValueSlice()...)
	case dto.MetricType_SUMMARY:
		quantiles := make(map[float64]float64, len(metric.Summary.GetQuantile()))
		for _, q := range metric.Summary.GetQuantile() {
			quantiles[q.GetQuantile()] = q.GetValue()
		}

		return prometheus.MustNewConstSummary(desc, metric.Summary.GetSampleCount(), metric.Summary.GetSampleSum(), quantiles, labelSet.toLabelValueSlice()...)
	case dto.MetricType_HISTOGRAM:
		buckets := make(map[float64]uint64, len(metric.Histogram.GetBucket()))
		for _, b := range metric.Histogram.GetBucket() {
			buckets[b.GetUpperBound()] = b.GetCumulativeCount()
		}

		return prometheus.MustNewConstHistogram(desc, metric.Histogram.GetSampleCount(), metric.Histogram.GetSampleSum(), buckets, labelSet.toLabelValueSlice()...)
	default:
		panic("unsupported metric type")
	}
}
