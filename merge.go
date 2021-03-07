package promaggr

import (
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"
)

// MergeOption is a functional option used by the MergeMetricFamily.
type MergeOption func(mfs1, mfs2 []*dto.MetricFamily)

// MergeMetricFamily returns the result of merging the slices of two MetricFamily's.
// Be careful not to have metric with the same name and the same label when merging.
// To avoid metric conflicts, you can use the built-in AddIdentifierLabel option.
func MergeMetricFamily(mfs1, mfs2 []*dto.MetricFamily, opts ...MergeOption) []*dto.MetricFamily {
	for _, o := range opts {
		o(mfs1, mfs2)
	}

	mfSet := make(map[string]*dto.MetricFamily)

	for _, mfs := range [...][]*dto.MetricFamily{mfs1, mfs2} {
		for _, mf := range mfs {
			if _, ok := mfSet[mf.GetName()]; ok {
				mfSet[mf.GetName()].Metric = append(mfSet[mf.GetName()].Metric, mf.GetMetric()...)
			} else {
				mfSet[mf.GetName()] = mf
			}
		}
	}

	mergedMfs := make([]*dto.MetricFamily, 0, len(mfSet))
	for _, mf := range mfSet {
		mergedMfs = append(mergedMfs, mf)
	}

	return mergedMfs
}

// AddIdentifierLabel is an option available for MergeMetricFamily.
// Add labels for identifiers to avoid metric conflicts when merging.
func AddIdentifierLabel(mfs1, mfs2 []*dto.MetricFamily, label model.LabelName, mfs1Identifier, mfs2Identifier model.LabelValue) MergeOption {
	return func(mfs1, mfs2 []*dto.MetricFamily) {
		AddLabels(mfs1, model.LabelSet{label: mfs1Identifier})
		AddLabels(mfs2, model.LabelSet{label: mfs2Identifier})
	}
}
