package lifecycle

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Pingable is implemented by external dependencies (PostgreSQL pools, caches, etc.)
// that must be healthy before the service claims readiness.
type Pingable interface {
	PingContext(ctx context.Context) error
}

// HealthChecker manages process liveness and readiness states.
type HealthChecker struct {
	alive atomic.Bool
	ready atomic.Bool
	mu    sync.RWMutex
	deps  map[string]Pingable
}

// NewHealthChecker creates a health checker initialized with alive=true and ready=false.
func NewHealthChecker() *HealthChecker {
	hc := &HealthChecker{
		deps: make(map[string]Pingable),
	}
	hc.alive.Store(true)
	hc.ready.Store(false)
	return hc
}

// RegisterDependency adds a critical pingable dependency (e.g. "postgres", "redis").
func (hc *HealthChecker) RegisterDependency(name string, dep Pingable) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.deps[name] = dep
}

// MarkReady flips readiness to true once listeners and workers are serving.
func (hc *HealthChecker) MarkReady() {
	hc.ready.Store(true)
}

// MarkNotReady flips readiness to false (e.g., at the start of the drain phase).
func (hc *HealthChecker) MarkNotReady() {
	hc.ready.Store(false)
}

// MarkDead marks liveness as false if the application encountered an unrecoverable state.
func (hc *HealthChecker) MarkDead() {
	hc.alive.Store(false)
}

// IsReady reports current readiness state.
func (hc *HealthChecker) IsReady() bool {
	return hc.ready.Load()
}

// IsAlive reports current liveness state.
func (hc *HealthChecker) IsAlive() bool {
	return hc.alive.Load()
}

// RegisterRoutes registers the standard /livez and /readyz endpoints.
func (hc *HealthChecker) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /livez", hc.HandleLiveness)
	mux.HandleFunc("GET /readyz", hc.HandleReadiness)
}

// HandleLiveness handles the /livez endpoint.
// It checks only if the process is responsive. It intentionally does NOT check external dependencies.
func (hc *HealthChecker) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !hc.alive.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "dead",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "alive",
	})
}

// HandleReadiness handles the /readyz endpoint.
// It verifies the atomic ready flag and pings all registered critical dependencies.
func (hc *HealthChecker) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !hc.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "not_ready",
		})
		return
	}

	// Verify dependencies with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	hc.mu.RLock()
	depsCopy := make(map[string]Pingable, len(hc.deps))
	for k, v := range hc.deps {
		depsCopy[k] = v
	}
	hc.mu.RUnlock()

	for name, dep := range depsCopy {
		if dep != nil {
			if err := dep.PingContext(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status":     "degraded",
					"dependency": name,
					"error":      err.Error(),
				})
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
	})
}
