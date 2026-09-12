package metrics

import (
	"fmt"
	"math"

	"github.com/prometheus/client_golang/prometheus"
)

// RUM (real-user monitoring) families, ingested via POST /rum on the
// management listener (P7). Time-based families are seconds; cashflow_rum_cls
// is a dimensionless layout-shift score. Every family is a histogram
// partitioned by the closed client_type label set.
const (
	MetricRUMTTFB           = "cashflow_rum_ttfb"
	MetricRUMLCP            = "cashflow_rum_lcp"
	MetricRUMINP            = "cashflow_rum_inp"
	MetricRUMCLS            = "cashflow_rum_cls"
	MetricRUMDOMInteractive = "cashflow_rum_dom_interactive"
)

// ValidRUMFamily reports whether s is one of the registered RUM families.
func ValidRUMFamily(s string) bool {
	switch s {
	case MetricRUMTTFB, MetricRUMLCP, MetricRUMINP, MetricRUMCLS, MetricRUMDOMInteractive:
		return true
	default:
		return false
	}
}

// newRUMHistograms builds the five RUM families keyed by family name, with the
// closed client_type dimension. Called once from NewRegistry.
func newRUMHistograms() map[string]*prometheus.HistogramVec {
	kind := "real-user-monitoring sample in seconds, partitioned by client type."
	cls := "real-user-monitoring cumulative layout shift score, partitioned by client type."
	return map[string]*prometheus.HistogramVec{
		MetricRUMTTFB:           prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: MetricRUMTTFB, Help: "Time to first byte " + kind, Buckets: LatencyBuckets}, []string{LabelClientType}),
		MetricRUMLCP:            prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: MetricRUMLCP, Help: "Largest contentful paint " + kind, Buckets: LatencyBuckets}, []string{LabelClientType}),
		MetricRUMINP:            prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: MetricRUMINP, Help: "Interaction to next paint " + kind, Buckets: LatencyBuckets}, []string{LabelClientType}),
		MetricRUMDOMInteractive: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: MetricRUMDOMInteractive, Help: "DOM interactive " + kind, Buckets: LatencyBuckets}, []string{LabelClientType}),
		MetricRUMCLS:            prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: MetricRUMCLS, Help: cls, Buckets: CLSBuckets}, []string{LabelClientType}),
	}
}

// observeRUM records one real-user sample. family and clientType must be from
// their closed sets; value must be finite and non-negative. An error is
// returned for any violation so callers never introduce unbounded labels.
func (r *Registry) observeRUM(family, clientType string, value float64) error {
	if !ValidRUMFamily(family) {
		return fmt.Errorf("unknown rum family %q", family)
	}
	if !ValidClientType(clientType) {
		return fmt.Errorf("unknown client_type %q", clientType)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return fmt.Errorf("rum value must be finite and >= 0, got %v", value)
	}
	vec, ok := r.rum[family]
	if !ok {
		return fmt.Errorf("rum family %q not registered", family)
	}
	vec.WithLabelValues(clientType).Observe(value)
	return nil
}
