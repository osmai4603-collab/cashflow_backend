package metrics

import (
	"encoding/json"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type Metrics struct {
	// System Metrics
	ServerURL           string  `json:"server_url,omitempty"`
	MemoryAlloc         uint64  `json:"memory_alloc_bytes"`
	MemoryTotal         uint64  `json:"memory_total_bytes"`
	MemorySys           uint64  `json:"memory_sys_bytes"`
	HeapAlloc           uint64  `json:"heap_alloc_bytes"`
	HeapIdle            uint64  `json:"heap_idle_bytes"`
	HeapInuse           uint64  `json:"heap_inuse_bytes"`
	NumGoroutines       int     `json:"num_goroutines"`
	NumGC               uint32  `json:"num_gc"`
	GCPauseSecondsTotal float64 `json:"gc_pause_seconds_total"`
	UptimeSeconds       float64 `json:"uptime_seconds"`

	// HTTP Traffic Metrics
	HTTPRequests     uint64  `json:"http_requests_total"`
	HTTP2xx          uint64  `json:"http_2xx_total"`
	HTTP3xx          uint64  `json:"http_3xx_total"`
	HTTP4xx          uint64  `json:"http_4xx_total"`
	HTTP5xx          uint64  `json:"http_5xx_total"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	P50LatencyMs     float64 `json:"p50_latency_ms"`
	P95LatencyMs     float64 `json:"p95_latency_ms"`
	P99LatencyMs     float64 `json:"p99_latency_ms"`
	AvgResponseBytes float64 `json:"avg_response_bytes"`
	WindowSeconds    int     `json:"window_seconds,omitempty"`
	LifetimeAvgMs    float64 `json:"lifetime_avg_latency_ms,omitempty"`
	LifetimeP95Ms    float64 `json:"lifetime_p95_latency_ms,omitempty"`

	// Health Probes Latency
	ReadyzLatencyMs float64 `json:"readyz_latency_ms"`
	LivezLatencyMs  float64 `json:"livez_latency_ms"`

	// Database Metrics
	DB *DBStats `json:"db,omitempty"`

	// Real User Monitoring (optional, omitted when empty)
	RUM *RUMSummary `json:"rum,omitempty"`

	// Recent Errors (optional, omitted when empty)
	RecentErrors []HTTPErrorEvent `json:"recent_errors,omitempty"`
}

// defaultRegistry is the process-wide single source of truth. It is safe to
// register, gather and update concurrently once built.
var defaultRegistry = NewRegistry()

// RegisterDB registers a database stats provider (e.g., a pgxpool) on the
// default registry. Safe to call after startup.
func RegisterDB(provider DBStatsProvider) {
	defaultRegistry.RegisterDB(provider)
}

// RegisterServerURL registers the advertised base URL for the server.
// Safe to call after startup.
func RegisterServerURL(url string) {
	defaultRegistry.RegisterServerURL(url)
}

// RecordHTTPError records an HTTP error on the default registry.
func RecordHTTPError(method, path string, status int, duration time.Duration) {
	defaultRegistry.RecordHTTPError(method, path, status, duration)
}

// ObserveRUM records one real-user-monitoring sample into the default
// registry. family and clientType must be from their closed sets; value must
// be finite and non-negative. An error is returned for any violation.
func ObserveRUM(family, clientType string, value float64) error {
	return defaultRegistry.observeRUM(family, clientType, value)
}

type statusWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.bytesWritten += int64(n)
	return n, err
}

// Unwrap keeps http.ResponseController and friends working through the wrapper.
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// routePatternFor resolves the matched route template for a request. Outside a
// chi router has no route context; unmatched requests (e.g. 404) produce an
// empty pattern. Both collapse onto the "unknown" label so cardinality stays
// bounded.
func routePatternFor(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if p := rc.RoutePattern(); p != "" {
			return p
		}
	}
	return UnknownRoute
}

// Middleware records HTTP traffic into the default registry. Requests to
// observability endpoints themselves are suppressed from business counters;
// probes keep a dedicated latency metric so their cost does not pollute the
// traffic histogram.
func Middleware(next http.Handler) http.Handler {
	return defaultRegistry.httpMiddleware(next)
}

func (r *Registry) httpMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/metrics":
			next.ServeHTTP(w, req)
			return
		case "/livez", "/readyz":
			start := time.Now()
			next.ServeHTTP(w, req)
			r.ObserveProbe(strings.TrimPrefix(req.URL.Path, "/"), time.Since(start))
			return
		}

		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, req)

		dur := time.Since(start)
		r.ObserveHTTP(
			req.Method,
			routePatternFor(req),
			statusClassFor(sw.status),
			dur,
			sw.bytesWritten,
		)
		if sw.status >= 400 {
			r.RecordHTTPError(req.Method, req.URL.Path, sw.status, dur)
		}
	})
}

// Handler returns current system and application metrics as JSON, read from
// the default registry.
func Handler(w http.ResponseWriter, r *http.Request) {
	defaultRegistry.jsonHandler(w)
}

// PrometheusHandler returns the text (exposition) handler backed by the
// default registry, for the isolated management listener.
func PrometheusHandler() http.Handler {
	return defaultRegistry.PrometheusHandler()
}

func (r *Registry) jsonHandler(w http.ResponseWriter) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := r.ExtendedStats()

	metrics := Metrics{
		ServerURL:           r.serverURLValue(),
		MemoryAlloc:         m.Alloc,
		MemoryTotal:         m.TotalAlloc,
		MemorySys:           m.Sys,
		HeapAlloc:           m.HeapAlloc,
		HeapIdle:            m.HeapIdle,
		HeapInuse:           m.HeapInuse,
		NumGoroutines:       runtime.NumGoroutine(),
		NumGC:               m.NumGC,
		GCPauseSecondsTotal: float64(m.PauseTotalNs) / float64(time.Second),
		UptimeSeconds:       r.uptimeSeconds(),

		// Baseline contract: 3xx folds into the 2xx bucket.
		HTTPRequests:     stats.Total,
		HTTP2xx:          stats.ByClass[Status2xx] + stats.ByClass[Status3xx],
		HTTP3xx:          stats.ByClass[Status3xx],
		HTTP4xx:          stats.ByClass[Status4xx],
		HTTP5xx:          stats.ByClass[Status5xx],
		AvgLatencyMs:     stats.AvgMs,
		P50LatencyMs:     stats.P50Ms,
		P95LatencyMs:     stats.P95Ms,
		P99LatencyMs:     stats.P99Ms,
		AvgResponseBytes: stats.AvgResponseBytes,
		WindowSeconds:    stats.WindowSeconds,
		LifetimeAvgMs:    stats.LifetimeAvgMs,
		LifetimeP95Ms:    stats.LifetimeP95Ms,

		ReadyzLatencyMs: stats.ReadyzMs,
		LivezLatencyMs:  stats.LivezMs,
		RUM:             stats.RUM,
		RecentErrors:    stats.RecentErrors,
	}

	if p, ok := r.dbProvider.Load().(DBStatsProvider); ok && p != nil {
		dbStats := p.Stats()
		metrics.DB = &dbStats
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(metrics)
}
