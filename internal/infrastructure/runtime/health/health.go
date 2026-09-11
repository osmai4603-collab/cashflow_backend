package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
)

// Pinger defines an interface for critical dependencies that must be pinged in readiness checks.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthChecker manages the liveness and readiness state of the server
// using atomic booleans for thread-safe state changes.
type HealthChecker struct {
	alive atomic.Bool
	ready atomic.Bool
	deps  Pinger
}

// NewHealthChecker creates a health checker instance.
// The server starts as alive, but NOT ready (until initialization and startup complete).
func NewHealthChecker(deps Pinger) *HealthChecker {
	hc := &HealthChecker{
		deps: deps,
	}
	hc.alive.Store(true)
	hc.ready.Store(false) // Not ready until explicitly marked ready
	return hc
}

// MarkReady signals that startup is complete and the server is ready to receive traffic.
func (hc *HealthChecker) MarkReady() {
	hc.ready.Store(true)
}

// MarkNotReady signals that the server is draining and should stop receiving traffic.
func (hc *HealthChecker) MarkNotReady() {
	hc.ready.Store(false)
}

// IsReady returns the current readiness status.
func (hc *HealthChecker) IsReady() bool {
	return hc.ready.Load()
}

// IsAlive returns the current liveness status.
func (hc *HealthChecker) IsAlive() bool {
	return hc.alive.Load()
}

// HandleLiveness probe (/livez):
// RULE: ONLY checks process health. NEVER checks external dependencies.
// Checking DB here causes cascading pod restarts in Kubernetes during temporary DB flickers.
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

// HandleReadiness probe (/readyz):
// RULE: Checks if server is draining AND checks critical dependencies.
// If not ready, load balancer removes pod from routing pool without restarting it.
func (hc *HealthChecker) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !hc.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "not_ready",
			"reason": "server is draining or initializing",
		})
		return
	}

	if hc.deps != nil {
		if err := hc.deps.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "not_ready",
				"reason": "critical dependency check failed: " + err.Error(),
			})
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
	})
}
