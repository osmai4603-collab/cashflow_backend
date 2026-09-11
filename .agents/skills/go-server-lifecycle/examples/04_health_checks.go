package lifecycle

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// =============================================================================
// Phase 4: Health Check & Metrics Endpoints
// =============================================================================
// Probes:
// - Liveness  → /livez   → Is the process healthy? (No external deps check)
// - Readiness → /readyz  → Can it serve traffic? (Check critical deps)
// - Metrics   → /metrics → Provide runtime telemetry (CPU, Mem, DB, Traffic)
// =============================================================================

// DBStatsProvider allows storage backends to report health.
type DBStatsProvider interface {
	Stats() DBStats
}

type DBStats struct {
	ActiveConns int32 `json:"active_conns"`
	MaxConns    int32 `json:"max_conns"`
}

type HealthChecker struct {
	ready atomic.Bool
	alive atomic.Bool
	deps  *Dependencies
	db    DBStatsProvider
}

func NewHealthChecker(deps *Dependencies, db DBStatsProvider) *HealthChecker {
	hc := &HealthChecker{deps: deps, db: db}
	hc.alive.Store(true)
	hc.ready.Store(false)
	return hc
}

func (hc *HealthChecker) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /livez", hc.handleLiveness)
	mux.HandleFunc("GET /readyz", hc.handleReadiness)
	mux.HandleFunc("GET /metrics", hc.handleMetrics)
}

func (hc *HealthChecker) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

func (hc *HealthChecker) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if !hc.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Example: Check DB dependency
	if hc.deps != nil && hc.deps.DB != nil {
		if err := hc.deps.DB.PingContext(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (hc *HealthChecker) handleMetrics(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	data := map[string]interface{}{
		"memory_alloc_mb": float64(m.Alloc) / 1048576,
		"num_goroutines":  runtime.NumGoroutine(),
		"uptime_seconds":  time.Since(startTime).Seconds(),
	}

	if hc.db != nil {
		data["db"] = hc.db.Stats()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

var startTime = time.Now()
