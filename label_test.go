package promaggr_test

import (
	"testing"

	"github.com/d-kuro/promaggr"
	"github.com/d-kuro/promaggr/internal"
	"github.com/google/go-cmp/cmp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"
)

func TestAddLabels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mfs       []*dto.MetricFamily
		labelSet  model.LabelSet
		wantLabel []*dto.LabelPair
	}{
		{
			name: "add labels",
			mfs: []*dto.MetricFamily{
				internal.NewCounterMetricFamilyFixture("dummy",
					internal.Label([]*dto.LabelPair{
						{
							Name:  internal.StringToPointer("foo"),
							Value: internal.StringToPointer("foo"),
						},
					}),
				),
			},
			labelSet: map[model.LabelName]model.LabelValue{
				"bar": "bar",
			},
			wantLabel: []*dto.LabelPair{ // labels must be sorted in dictionary order
				{
					Name:  internal.StringToPointer("bar"),
					Value: internal.StringToPointer("bar"),
				},
				{
					Name:  internal.StringToPointer("foo"),
					Value: internal.StringToPointer("foo"),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			promaggr.AddLabels(tt.mfs, tt.labelSet)

			for _, mf := range tt.mfs {
				for _, metric := range mf.Metric {
					if diff := cmp.Diff(tt.wantLabel, metric.Label); diff != "" {
						t.Errorf("labels mismatch (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}
