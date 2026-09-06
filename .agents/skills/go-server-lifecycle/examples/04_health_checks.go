package lifecycle

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

// =============================================================================
// Phase 4: Health Check Endpoints
// =============================================================================
// Separate endpoints for each probe type.
// CRITICAL: Liveness must NOT check external dependencies.
//
// Official best practice for Kubernetes probes:
// - Liveness  → /livez   → Is the process alive?
// - Readiness → /readyz  → Can it serve traffic?
// - Startup   → /startupz → Has initialization finished?
// =============================================================================

// HealthChecker manages the health state of the server.
// It uses atomic values for thread-safe state changes during shutdown.
type HealthChecker struct {
	// ready indicates whether the server is ready to accept traffic.
	// Set to false during drain phase to remove from load balancer.
	ready atomic.Bool

	// alive indicates whether the process is healthy.
	alive atomic.Bool

	// dependencies holds references to check in readiness probes.
	deps *Dependencies
}

// NewHealthChecker creates a new HealthChecker.
// The server starts as alive but NOT ready (until initialization completes).
func NewHealthChecker(deps *Dependencies) *HealthChecker {
	hc := &HealthChecker{
		deps: deps,
	}
	hc.alive.Store(true)
	hc.ready.Store(false) // Not ready until initialization completes
	return hc
}

// MarkReady signals that the server has completed initialization
// and is ready to serve traffic.
func (hc *HealthChecker) MarkReady() {
	hc.ready.Store(true)
}

// MarkNotReady signals that the server should stop receiving traffic.
// Used during the drain phase before shutdown.
func (hc *HealthChecker) MarkNotReady() {
	hc.ready.Store(false)
}

// RegisterRoutes registers health check endpoints on the given mux.
func (hc *HealthChecker) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /livez", hc.handleLiveness)
	mux.HandleFunc("GET /readyz", hc.handleReadiness)
}

// ─────────────────────────────────────────────────────────────────────────────
// Liveness Probe: /livez
// ─────────────────────────────────────────────────────────────────────────────
// PURPOSE: Determine if the process is alive and not deadlocked.
//
// RULE: Check ONLY the process health.
//       DO NOT check external dependencies (DB, cache, etc.)
//
// WHY: If liveness checks the DB and the DB has a temporary flicker,
//       Kubernetes restarts ALL pods → cascading failure.
// ─────────────────────────────────────────────────────────────────────────────

func (hc *HealthChecker) handleLiveness(w http.ResponseWriter, r *http.Request) {
	if !hc.alive.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "dead",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "alive",
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Readiness Probe: /readyz
// ─────────────────────────────────────────────────────────────────────────────
// PURPOSE: Determine if the server is ready to serve traffic.
//
// RULE: Check CRITICAL dependencies (DB connection, etc.)
//       If this fails, the pod is removed from the load balancer
//       but NOT restarted.
//
// USED DURING DRAIN: Set ready=false to stop receiving traffic
//                     before graceful shutdown.
// ─────────────────────────────────────────────────────────────────────────────

func (hc *HealthChecker) handleReadiness(w http.ResponseWriter, r *http.Request) {
	// First check: is the server marked as ready?
	if !hc.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not_ready",
			"reason": "server is draining or not initialized",
		})
		return
	}

	// Second check: are critical dependencies healthy?
	if hc.deps != nil && hc.deps.DB != nil {
		if err := hc.deps.DB.PingContext(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "not_ready",
				"reason": "database connection failed",
			})
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
	})
}
