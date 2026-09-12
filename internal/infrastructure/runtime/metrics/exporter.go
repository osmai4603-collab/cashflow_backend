package metrics

import (
	"net/http"
	"sort"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// PrometheusContentType is the exposition format used by the exporter. Only
// version 0.0.4 (Prometheus text) is offered; OpenMetrics (0.0.1) is left to
// P6 where the exact content negotiation can be decided with the scrape target.
const PrometheusContentType = "text/plain; version=0.0.4; charset=utf-8"

// familyOrder drives the exposition order: process first, then Go runtime,
// then HTTP traffic, then probes, then the database.
func familyOrder(name string) int {
	switch {
	case strings.HasPrefix(name, "cashflow_process_"):
		return 0
	case strings.HasPrefix(name, "cashflow_go_"):
		return 1
	case strings.HasPrefix(name, "cashflow_http_"):
		return 2
	case strings.HasPrefix(name, "cashflow_probe_"):
		return 3
	case strings.HasPrefix(name, "cashflow_db_"):
		return 4
	default:
		return 5
	}
}

// sortFamilies reorders gathered families by category so the text output is
// stable and human-aligned. Prometheus ignores ordering, so this is purely a
// presentation decorator; it never renames or drops series.
func sortFamilies(families []*dto.MetricFamily) {
	sort.SliceStable(families, func(i, j int) bool {
		oi, oj := familyOrder(families[i].GetName()), familyOrder(families[j].GetName())
		if oi != oj {
			return oi < oj
		}
		return families[i].GetName() < families[j].GetName()
	})
}

// PrometheusHandler returns an http.Handler serving the registry content as
// Prometheus text exposition. The output is cache-proof, reordered by category
// and prefixed with a decoration banner. It is intended to be mounted on the
// dedicated management listener (P4).
func (r *Registry) PrometheusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		families, err := r.Gather()
		if err != nil {
			http.Error(w, "metrics gather error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		sortFamilies(families)

		w.Header().Set("Content-Type", PrometheusContentType)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if _, err := w.Write([]byte("# cashflow backend observability, scrape target: " + r.serverURLValue() + "\n")); err != nil {
			return
		}
		for _, mf := range families {
			if _, err := expfmt.MetricFamilyToText(w, mf); err != nil {
				return
			}
		}
	})
}

// GatherCollectors exposes the underlying DTO families for decoration without
// triggering the JSON aggregation path.
func (r *Registry) GatherCollectors() ([]*dto.MetricFamily, error) {
	return r.Gather()
}

var _ prometheus.Gatherer = (*GathererAdapter)(nil)

// GathererAdapter adapts a Registry to prometheus.Gatherer so it composes with
// promhttp and any collector that expects the standard interface.
type GathererAdapter struct {
	Registry *Registry
}

func (g *GathererAdapter) Gather() ([]*dto.MetricFamily, error) {
	return g.Registry.Gather()
}
