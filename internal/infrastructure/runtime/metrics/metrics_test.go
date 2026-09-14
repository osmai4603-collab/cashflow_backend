package metrics_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cashflow_backend/internal/infrastructure/runtime/metrics"
)

// fakeDBProvider reports a fixed pool state for JSON shape tests.
type fakeDBProvider struct {
	stats metrics.DBStats
}

func (f fakeDBProvider) Stats() metrics.DBStats { return f.stats }

// snapshot invokes the JSON handler directly and decodes its output.
func snapshot(t *testing.T) map[string]interface{} {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	metrics.Handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from /metrics handler, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("expected valid JSON from /metrics handler: %v", err)
	}
	return m
}

func readRequestsTotal(t *testing.T) float64 {
	t.Helper()
	v, ok := snapshot(t)["http_requests_total"].(float64)
	if !ok {
		t.Fatal("http_requests_total missing or not a number")
	}
	return v
}

func TestMetricsHandler_JSONShapeAndContentType(t *testing.T) {
	metrics.RegisterServerURL("http://localhost:8070")
	metrics.RegisterDB(fakeDBProvider{stats: metrics.DBStats{
		MaxConns:          25,
		ActiveConns:       3,
		IdleConns:         2,
		WaitCount:         7,
		EmptyAcquireCount: 1,
		WaitDuration:      500 * time.Millisecond,
	}})

	m := snapshot(t)

	// Every field name in the current JSON contract must be present.
	// (Baseline: these names must NOT change in the first observability release.)
	for _, name := range []string{
		"server_url",
		"memory_alloc_bytes",
		"memory_total_bytes",
		"memory_sys_bytes",
		"heap_alloc_bytes",
		"heap_idle_bytes",
		"heap_inuse_bytes",
		"num_goroutines",
		"num_gc",
		"uptime_seconds",
		"http_requests_total",
		"http_2xx_total",
		"http_4xx_total",
		"http_5xx_total",
		"avg_latency_ms",
		"db",
	} {
		if _, ok := m[name]; !ok {
			t.Errorf("missing JSON metric field %q in /metrics snapshot", name)
		}
	}

	if got := m["server_url"]; got != "http://localhost:8070" {
		t.Errorf("expected server_url http://localhost:8070, got %v", got)
	}

	db, ok := m["db"].(map[string]interface{})
	if !ok {
		t.Fatal("expected db object in /metrics snapshot")
	}
	if v := db["max_conns"].(float64); v != 25 {
		t.Errorf("expected db.max_conns 25, got %v", v)
	}
	if v := db["active_conns"].(float64); v != 3 {
		t.Errorf("expected db.active_conns 3, got %v", v)
	}
	if v := db["idle_conns"].(float64); v != 2 {
		t.Errorf("expected db.idle_conns 2, got %v", v)
	}
	if v := db["wait_count"].(float64); v != 7 {
		t.Errorf("expected db.wait_count 7, got %v", v)
	}
	if v := db["empty_acquire_count"].(float64); v != 1 {
		t.Errorf("expected db.empty_acquire_count 1, got %v", v)
	}
	if v := db["wait_duration_ns"].(float64); v != float64((500 * time.Millisecond).Nanoseconds()) {
		t.Errorf("expected db.wait_duration_ns 500000000, got %v", v)
	}
}

func TestMiddleware_ExcludesObservabilityPaths(t *testing.T) {
	before := readRequestsTotal(t)

	handler := metrics.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Current baseline: only these three paths are suppressed from counting.
	for _, path := range []string{"/metrics", "/livez", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 on %s, got %d", path, rr.Code)
		}
	}

	if after := readRequestsTotal(t); after != before {
		t.Errorf("observability paths leaked into request counters: before=%v after=%v", before, after)
	}
}

func TestMiddleware_CountsBusinessRequestsByStatusClass(t *testing.T) {
	before := snapshot(t)
	f := func(key string) float64 {
		v, ok := before[key].(float64)
		if !ok {
			t.Fatalf("%s missing or not a number", key)
		}
		return v
	}
	totalBefore := f("http_requests_total")
	before2xx := f("http_2xx_total")
	before4xx := f("http_4xx_total")
	before5xx := f("http_5xx_total")

	handler := metrics.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
		case "/notfound":
			w.WriteHeader(http.StatusNotFound)
		case "/boom":
			w.WriteHeader(http.StatusInternalServerError)
		case "/redirect":
			w.WriteHeader(http.StatusMovedPermanently)
		}
	}))

	for _, p := range []string{"/ok", "/ok", "/notfound", "/boom", "/redirect"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	after := snapshot(t)
	g := func(key string) float64 {
		v, ok := after[key].(float64)
		if !ok {
			t.Fatalf("%s missing or not a number", key)
		}
		return v
	}

	if got := g("http_requests_total") - totalBefore; got != 5 {
		t.Errorf("expected +5 total requests, got %v", got)
	}
	// Baseline nuance: the current classifier counts any status >= 200 (including
	// 3xx redirects) in the 2xx bucket. P2 will introduce a real status_class.
	if got := g("http_2xx_total") - before2xx; got != 3 {
		t.Errorf("expected +3 to 2xx counters (2x 200 + 1 redirect), got %v", got)
	}
	if got := g("http_4xx_total") - before4xx; got != 1 {
		t.Errorf("expected +1 to 4xx counters, got %v", got)
	}
	if got := g("http_5xx_total") - before5xx; got != 1 {
		t.Errorf("expected +1 to 5xx counters, got %v", got)
	}
	if got := g("http_3xx_total"); got < 1 {
		t.Errorf("expected at least 1 redirect in http_3xx_total, got %v", got)
	}
}

func TestMetricsHandler_ExtendedStatsAndPercentiles(t *testing.T) {
	m := snapshot(t)

	// Ensure all extended fields exist in the JSON output
	extendedFields := []string{
		"p50_latency_ms",
		"p95_latency_ms",
		"p99_latency_ms",
		"avg_response_bytes",
		"readyz_latency_ms",
		"livez_latency_ms",
		"gc_pause_seconds_total",
		"http_3xx_total",
		"window_seconds",
	}

	for _, field := range extendedFields {
		if _, ok := m[field]; !ok {
			t.Errorf("missing extended metric field %q in /metrics snapshot", field)
		}
	}
}

func TestMetricsHandler_RUMExposition(t *testing.T) {
	if err := metrics.ObserveRUM("cashflow_rum_lcp", "web", 0.85); err != nil {
		t.Fatalf("ObserveRUM: %v", err)
	}

	m := snapshot(t)
	rum, ok := m["rum"].(map[string]interface{})
	if !ok || rum == nil {
		t.Fatal("expected non-nil rum object after ObserveRUM")
	}

	if count, ok := rum["samples_count"].(float64); !ok || count < 1 {
		t.Errorf("expected samples_count >= 1, got %v", rum["samples_count"])
	}
	if lcp, ok := rum["avg_lcp_ms"].(float64); !ok || lcp < 800 {
		t.Errorf("expected avg_lcp_ms around 850ms, got %v", rum["avg_lcp_ms"])
	}
}

