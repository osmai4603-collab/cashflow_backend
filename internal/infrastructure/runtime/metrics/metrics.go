package metrics

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// DBStatsProvider allows different storage backends to report their pool health.
type DBStatsProvider interface {
	Stats() DBStats
}

type DBStats struct {
	MaxConns    int32 `json:"max_conns"`
	ActiveConns int32 `json:"active_conns"`
	IdleConns   int32 `json:"idle_conns"`
	WaitCount   int64 `json:"wait_count"`
}

type Metrics struct {
	// System Metrics
	ServerURL     string  `json:"server_url,omitempty"`
	MemoryAlloc   uint64  `json:"memory_alloc_bytes"`
	MemoryTotal   uint64  `json:"memory_total_bytes"`
	MemorySys     uint64  `json:"memory_sys_bytes"`
	HeapAlloc     uint64  `json:"heap_alloc_bytes"`
	HeapIdle      uint64  `json:"heap_idle_bytes"`
	HeapInuse     uint64  `json:"heap_inuse_bytes"`
	NumGoroutines int     `json:"num_goroutines"`
	NumGC         uint32  `json:"num_gc"`
	UptimeSeconds float64 `json:"uptime_seconds"`

	// HTTP Traffic Metrics
	HTTPRequests uint64  `json:"http_requests_total"`
	HTTP2xx      uint64  `json:"http_2xx_total"`
	HTTP4xx      uint64  `json:"http_4xx_total"`
	HTTP5xx      uint64  `json:"http_5xx_total"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`

	// Database Metrics
	DB *DBStats `json:"db,omitempty"`
}

var (
	startTime      = time.Now()
	requestCount   uint64
	status2xx      uint64
	status4xx      uint64
	status5xx      uint64
	totalLatencyNs int64
	dbProvider     DBStatsProvider
	serverURL      string
)

// RegisterDB registers a database stats provider (e.g., a pgxpool).
func RegisterDB(provider DBStatsProvider) {
	dbProvider = provider
}

// RegisterServerURL registers the advertised base URL for the server.
func RegisterServerURL(url string) {
	serverURL = url
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Middleware increments the global request counter and tracks latency/status codes.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Suppress internal observability traffic from metrics calculation
		if r.URL.Path == "/metrics" || r.URL.Path == "/livez" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		atomic.AddUint64(&requestCount, 1)

		next.ServeHTTP(sw, r)

		duration := time.Since(start)
		atomic.AddInt64(&totalLatencyNs, int64(duration))

		switch {
		case sw.status >= 500:
			atomic.AddUint64(&status5xx, 1)
		case sw.status >= 400:
			atomic.AddUint64(&status4xx, 1)
		case sw.status >= 200:
			atomic.AddUint64(&status2xx, 1)
		}
	})
}

// Handler returns current system and application metrics as JSON.
func Handler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	reqCount := atomic.LoadUint64(&requestCount)
	latNs := atomic.LoadInt64(&totalLatencyNs)

	var avgLatMs float64
	if reqCount > 0 {
		avgLatMs = float64(latNs) / float64(reqCount) / float64(time.Millisecond)
	}

	metrics := Metrics{
		ServerURL:     serverURL,
		MemoryAlloc:   m.Alloc,
		MemoryTotal:   m.TotalAlloc,
		MemorySys:     m.Sys,
		HeapAlloc:     m.HeapAlloc,
		HeapIdle:      m.HeapIdle,
		HeapInuse:     m.HeapInuse,
		NumGoroutines: runtime.NumGoroutine(),
		NumGC:         m.NumGC,
		UptimeSeconds: time.Since(startTime).Seconds(),

		HTTPRequests: reqCount,
		HTTP2xx:      atomic.LoadUint64(&status2xx),
		HTTP4xx:      atomic.LoadUint64(&status4xx),
		HTTP5xx:      atomic.LoadUint64(&status5xx),
		AvgLatencyMs: avgLatMs,
	}

	if dbProvider != nil {
		stats := dbProvider.Stats()
		metrics.DB = &stats
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(metrics)
}
