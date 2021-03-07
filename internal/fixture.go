package internal

import dto "github.com/prometheus/client_model/go"

type FixtureOption func(mf *dto.MetricFamily)

func Label(labels []*dto.LabelPair) func(mf *dto.MetricFamily) {
	return func(mf *dto.MetricFamily) {
		for _, metric := range mf.Metric {
			metric.Label = labels
		}
	}
}

func NewCounterMetricFamilyFixture(name string, opts ...FixtureOption) *dto.MetricFamily {
	mf := &dto.MetricFamily{
		Name: &name,
		Help: StringToPointer("Dummy text."),
		Type: dto.MetricType_COUNTER.Enum(),
		Metric: []*dto.Metric{
			{
				Counter: &dto.Counter{
					Value: Float64ToPointer(123456),
				},
			},
		},
	}

	for _, o := range opts {
		o(mf)
	}

	return mf
}

func NewGaugeMetricFamilyFixture(name string, opts ...FixtureOption) *dto.MetricFamily {
	mf := &dto.MetricFamily{
		Name: &name,
		Help: StringToPointer("Dummy text."),
		Type: dto.MetricType_GAUGE.Enum(),
		Metric: []*dto.Metric{
			{
				Gauge: &dto.Gauge{
					Value: Float64ToPointer(123.456),
				},
			},
		},
	}

	for _, o := range opts {
		o(mf)
	}

	return mf
}

func StringToPointer(s string) *string {
	return &s
}

func Float64ToPointer(f float64) *float64 {
	return &f
}
