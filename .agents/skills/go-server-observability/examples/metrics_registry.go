package observability

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// =============================================================================
// Unified In-Memory Metrics Registry & Dual-Exposition Handlers
// =============================================================================

type MetricsRegistry struct {
	startTime time.Time
	window    *WindowTracker
	errors    *ErrorRingBuffer
	dbProvider DBPoolStatsProvider

	// Cumulative counters for Prometheus
	totalRequests  atomic.Int64
	total2xx       atomic.Int64
	total4xx       atomic.Int64
	total5xx       atomic.Int64
	totalLatencyUs atomic.Int64
}

func NewMetricsRegistry(db DBPoolStatsProvider) *MetricsRegistry {
	return &MetricsRegistry{
		startTime:  time.Now(),
		window:     NewWindowTracker(),
		errors:     NewErrorRingBuffer(10),
		dbProvider: db,
	}
}

// RecordRequest updates window percentiles, cumulative counters, and error buffers.
func (r *MetricsRegistry) RecordRequest(method, path string, status int, dur time.Duration) {
	micros := dur.Microseconds()
	r.window.Observe(dur)

	r.totalRequests.Add(1)
	r.totalLatencyUs.Add(micros)

	switch {
	case status >= 500:
		r.total5xx.Add(1)
		r.errors.Push(HTTPErrorEvent{
			Timestamp:  time.Now(),
			Method:     method,
			Path:       path,
			Status:     status,
			DurationMs: micros / 1000,
		})
	case status >= 400:
		r.total4xx.Add(1)
		r.errors.Push(HTTPErrorEvent{
			Timestamp:  time.Now(),
			Method:     method,
			Path:       path,
			Status:     status,
			DurationMs: micros / 1000,
		})
	default:
		r.total2xx.Add(1)
	}
}

// JSONSnapshotPayload defines the compact single-pass JSON schema.
type JSONSnapshotPayload struct {
	UptimeSeconds float64          `json:"uptime_seconds"`
	Timestamp     time.Time        `json:"timestamp"`
	Traffic       TrafficSnapshot  `json:"traffic"`
	Database      DBPoolMetrics    `json:"db"`
	Runtime       RuntimeSnapshot  `json:"runtime"`
	RecentErrors  []HTTPErrorEvent `json:"recent_errors"`
}

type TrafficSnapshot struct {
	TotalRequests int64   `json:"total_requests"`
	Total2xx      int64   `json:"total_2xx"`
	Total4xx      int64   `json:"total_4xx"`
	Total5xx      int64   `json:"total_5xx"`
	WindowRPS     float64 `json:"window_rps"`
	WindowAvgMs   float64 `json:"window_avg_ms"`
	P50Ms         float64 `json:"p50_ms"`
	P95Ms         float64 `json:"p95_ms"`
	P99Ms         float64 `json:"p99_ms"`
	IsIdle        bool    `json:"is_idle"`
}

type RuntimeSnapshot struct {
	AllocMB        float64 `json:"alloc_mb"`
	HeapSysMB      float64 `json:"heap_sys_mb"`
	NumGoroutines  int     `json:"num_goroutines"`
	NumGCCycles    uint32  `json:"num_gc_cycles"`
}

// HandleMetricsJSON exposes the single-pass compact JSON snapshot for CLI monitors.
func (r *MetricsRegistry) HandleMetricsJSON(w http.ResponseWriter, req *http.Request) {
	windowStats := r.window.Snapshot()
	dbStats := CollectPoolMetrics(r.dbProvider)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	payload := JSONSnapshotPayload{
		UptimeSeconds: time.Since(r.startTime).Seconds(),
		Timestamp:     time.Now(),
		Traffic: TrafficSnapshot{
			TotalRequests: r.totalRequests.Load(),
			Total2xx:      r.total2xx.Load(),
			Total4xx:      r.total4xx.Load(),
			Total5xx:      r.total5xx.Load(),
			WindowRPS:     windowStats.RPS,
			WindowAvgMs:   windowStats.AverageMs,
			P50Ms:         windowStats.P50Ms,
			P95Ms:         windowStats.P95Ms,
			P99Ms:         windowStats.P99Ms,
			IsIdle:        windowStats.IsIdle,
		},
		Database: dbStats,
		Runtime: RuntimeSnapshot{
			AllocMB:       float64(m.Alloc) / 1048576.0,
			HeapSysMB:     float64(m.HeapSys) / 1048576.0,
			NumGoroutines: runtime.NumGoroutine(),
			NumGCCycles:   m.NumGC,
		},
		RecentErrors: r.errors.Recent(),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(payload)
}

// HandleOpenMetrics exposes Prometheus 0.0.4 text format for centralized scrapers.
func (r *MetricsRegistry) HandleOpenMetrics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	fmt.Fprintf(w, "# HELP http_requests_total Total number of HTTP requests\n")
	fmt.Fprintf(w, "# TYPE http_requests_total counter\n")
	fmt.Fprintf(w, "http_requests_total{status=\"2xx\"} %d\n", r.total2xx.Load())
	fmt.Fprintf(w, "http_requests_total{status=\"4xx\"} %d\n", r.total4xx.Load())
	fmt.Fprintf(w, "http_requests_total{status=\"5xx\"} %d\n", r.total5xx.Load())

	if r.dbProvider != nil {
		stats := CollectPoolMetrics(r.dbProvider)
		fmt.Fprintf(w, "# HELP db_connections Active and idle database connections\n")
		fmt.Fprintf(w, "# TYPE db_connections gauge\n")
		fmt.Fprintf(w, "db_connections{state=\"active\"} %d\n", stats.ActiveConns)
		fmt.Fprintf(w, "db_connections{state=\"idle\"} %d\n", stats.IdleConns)
		fmt.Fprintf(w, "db_connections{state=\"max\"} %d\n", stats.MaxConns)
		fmt.Fprintf(w, "# HELP db_empty_acquire_total Count of times callers blocked waiting for DB pool slot\n")
		fmt.Fprintf(w, "# TYPE db_empty_acquire_total counter\n")
		fmt.Fprintf(w, "db_empty_acquire_total %d\n", stats.EmptyAcquireCount)
	}
}
