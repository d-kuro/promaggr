package promaggr

import (
	"sort"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"
)

// AddLabels adds the given label set to all metrics in the given MetricFamily's.
func AddLabels(mfs []*dto.MetricFamily, labels model.LabelSet) {
	for _, mf := range mfs {
		for _, m := range mf.Metric {
			sourceSet := make(model.LabelSet, len(m.Label))

			for _, l := range m.GetLabel() {
				if l.Name != nil {
					sourceSet[model.LabelName(l.GetName())] = model.LabelValue(l.GetValue())
				}
			}

			outputSet := sourceSet.Merge(labels)
			outputPairs := make([]*dto.LabelPair, 0, len(outputSet))

			for name, value := range outputSet {
				nameStr := string(name)
				valueStr := string(value)

				outputPairs = append(outputPairs, &dto.LabelPair{
					Name:  &nameStr,
					Value: &valueStr,
				})
			}

			// prometheus.Metric interface recommends sorting labels in lexicographic order.
			// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Metric
			sort.Slice(outputPairs, func(i, j int) bool {
				return outputPairs[i].GetName() < outputPairs[j].GetName()
			})

			m.Label = outputPairs
		}
	}
}
