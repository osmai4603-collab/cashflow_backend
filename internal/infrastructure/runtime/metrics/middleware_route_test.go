package metrics

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
)

type seriesInfo struct {
	labels    map[string]string
	counter   float64
	histCount uint64
	histSum   float64
}

func gatherSeries(t *testing.T, reg *Registry, family string) []seriesInfo {
	t.Helper()
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	var out []seriesInfo
	for _, f := range families {
		if f.GetName() != family {
			continue
		}
		for _, m := range f.GetMetric() {
			si := seriesInfo{labels: map[string]string{}}
			for _, lp := range m.GetLabel() {
				si.labels[lp.GetName()] = lp.GetValue()
			}
			if m.Counter != nil {
				si.counter = m.GetCounter().GetValue()
			}
			if m.Histogram != nil {
				si.histCount = m.GetHistogram().GetSampleCount()
				si.histSum = m.GetHistogram().GetSampleSum()
			}
			out = append(out, si)
		}
	}
	return out
}

func routeValues(t *testing.T, reg *Registry, family string) []string {
	t.Helper()
	var routes []string
	for _, s := range gatherSeries(t, reg, family) {
		routes = append(routes, s.labels[LabelRoute])
	}
	sort.Strings(routes)
	return routes
}

func chiTestRouter(reg *Registry) *chi.Mux {
	r := chi.NewRouter()
	r.Use(reg.httpMiddleware)
	r.Get("/api/v1/accounting", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	r.Get("/api/v1/sale", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	r.Get("/api/v1/partners/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("partner"))
	})
	return r
}

// TestMiddleware_DynamicRouteProducesSingleSeries asserts that a dynamic route
// yields ONE label, not one label per concrete ID value (gap 4.1).
func TestMiddleware_DynamicRouteProducesSingleSeries(t *testing.T) {
	reg := NewRegistry()
	router := chiTestRouter(reg)

	for _, p := range []string{"/api/v1/partners/123", "/api/v1/partners/999", "/api/v1/partners/xyz"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	}

	routes := routeValues(t, reg, MetricHTTPRequests)
	if len(routes) != 1 {
		t.Fatalf("expected exactly 1 route label from 3 dynamic IDs, got %v", routes)
	}
	if routes[0] != "/api/v1/partners/{id}" {
		t.Errorf("expected route template %q, got %q", "/api/v1/partners/{id}", routes[0])
	}

	series := gatherSeries(t, reg, MetricHTTPRequests)
	if len(series) != 1 {
		t.Fatalf("expected 1 series, got %d", len(series))
	}
	if series[0].labels[LabelMethod] != http.MethodGet {
		t.Errorf("unexpected method label %q", series[0].labels[LabelMethod])
	}
	if series[0].labels[LabelStatusClass] != Status2xx {
		t.Errorf("unexpected status_class label %q", series[0].labels[LabelStatusClass])
	}
	if series[0].counter != 3 {
		t.Errorf("expected 3 requests on the single series, got %v", series[0].counter)
	}
}

// TestMiddleware_RouteDistinction is P2's completion criterion: accounting,
// sale and partners resolve to three distinct templates with a bounded set.
func TestMiddleware_RouteDistinction(t *testing.T) {
	reg := NewRegistry()
	router := chiTestRouter(reg)

	for _, p := range []string{"/api/v1/accounting", "/api/v1/sale", "/api/v1/partners/42"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	}

	routes := routeValues(t, reg, MetricHTTPRequests)
	want := []string{"/api/v1/accounting", "/api/v1/partners/{id}", "/api/v1/sale"}
	if len(routes) != len(want) {
		t.Fatalf("expected %d route templates, got %v", len(want), routes)
	}
	for i := range want {
		if routes[i] != want[i] {
			t.Errorf("unexpected template set %v, want %v", routes, want)
		}
	}
}

// TestMiddleware_ProbeNotInBusinessTraffic asserts /readyz and /livez stay out
// of the business counter while their latency is tracked separately.
func TestMiddleware_ProbeNotInBusinessTraffic(t *testing.T) {
	reg := NewRegistry()
	router := chi.NewRouter()
	router.Use(reg.httpMiddleware)
	router.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	router.Get("/livez", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	router.Get("/api/v1/sale", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	for _, p := range []string{"/readyz", "/livez", "/api/v1/sale"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		router.ServeHTTP(httptest.NewRecorder(), req)
	}

	business := routeValues(t, reg, MetricHTTPRequests)
	if len(business) != 1 || business[0] != "/api/v1/sale" {
		t.Fatalf("probe traffic leaked into business counters: %v", business)
	}

	probes := gatherSeries(t, reg, MetricProbeDuration)
	if len(probes) != 2 {
		t.Fatalf("expected 2 probe series (livez, readyz), got %d", len(probes))
	}
	seen := map[string]bool{}
	for _, s := range probes {
		seen[s.labels[LabelProbe]] = true
		if s.histCount != 1 {
			t.Errorf("expected 1 probe sample for %q, got %d", s.labels[LabelProbe], s.histCount)
		}
	}
	if !seen[ProbeLivez] || !seen[ProbeReadyz] {
		t.Errorf("missing probe series: %v", seen)
	}
}

// TestMiddleware_UnknownRouteForUnmatched asserts that non-routed requests
// collapse onto the "unknown" label.
func TestMiddleware_UnknownRouteForUnmatched(t *testing.T) {
	reg := NewRegistry()
	router := chi.NewRouter()
	router.Use(reg.httpMiddleware)
	router.Get("/api/v1/known", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/no/such/route", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	routes := routeValues(t, reg, MetricHTTPRequests)
	if len(routes) != 1 || routes[0] != UnknownRoute {
		t.Fatalf("expected single %q series, got %v", UnknownRoute, routes)
	}
}

// TestMiddleware_ResponseSizeRecorded asserts the byte histogram captures the
// written body size per route.
func TestMiddleware_ResponseSizeRecorded(t *testing.T) {
	reg := NewRegistry()
	router := chi.NewRouter()
	router.Use(reg.httpMiddleware)
	router.Get("/api/v1/sale", func(w http.ResponseWriter, _ *http.Request) {
		w.Write(make([]byte, 1000))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sale", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	series := gatherSeries(t, reg, MetricHTTPResponseSize)
	if len(series) != 1 {
		t.Fatalf("expected 1 response-size series, got %d", len(series))
	}
	s := series[0]
	if s.histCount != 1 {
		t.Errorf("expected 1 sample, got %d", s.histCount)
	}
	if s.histSum != 1000 {
		t.Errorf("expected sample sum 1000, got %v", s.histSum)
	}
	if s.labels[LabelRoute] != "/api/v1/sale" {
		t.Errorf("unexpected route label %q", s.labels[LabelRoute])
	}
}

// TestMiddleware_ConcurrentSafety locks the "concurrent" middleware checklist
// item (run under -race): every request through the middleware must be recorded
// exactly once, with no dropped or duplicated series.
func TestMiddleware_ConcurrentSafety(t *testing.T) {
	reg := NewRegistry()
	router := chiTestRouter(reg)

	const goroutines = 24
	const perGoroutine = 40
	want := goroutines * perGoroutine

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/accounting", nil))
				if rr.Code != http.StatusOK {
					t.Errorf("unexpected non-200 under concurrency: %d", rr.Code)
				}
			}
		}()
	}
	wg.Wait()

	var total float64
	for _, s := range gatherSeries(t, reg, MetricHTTPRequests) {
		total += s.counter
	}
	if total != float64(want) {
		t.Errorf("expected %d recorded requests, got %.0f", want, total)
	}

	var histCount uint64
	for _, s := range gatherSeries(t, reg, MetricHTTPRequestDuration) {
		histCount += s.histCount
	}
	if histCount != uint64(want) {
		t.Errorf("expected %d histogram samples, got %d", want, histCount)
	}
}
