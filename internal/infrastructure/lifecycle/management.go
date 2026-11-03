package lifecycle

import (
	"encoding/json"
	"expvar"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// ManagementRouter creates the internal management HTTP handler.
// It exposes health probes, system metrics, and operational endpoints,
// secured by optional Bearer token authentication.
func NewManagementRouter(cfg *Config, health *HealthChecker, appStartTime time.Time) http.Handler {
	mux := http.NewServeMux()

	// 1. Probes on management port
	health.RegisterRoutes(mux)

	// 2. System info endpoint
	mux.HandleFunc("GET /info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		_ = json.NewEncoder(w).Encode(map[string]any{
			"app":         cfg.AppName,
			"uptime":      time.Since(appStartTime).String(),
			"go_version":  runtime.Version(),
			"num_cpu":     runtime.NumCPU(),
			"goroutines":  runtime.NumGoroutine(),
			"alloc_bytes": m.Alloc,
			"sys_bytes":   m.Sys,
		})
	})

	// 3. Expvar metrics endpoint
	mux.Handle("GET /debug/vars", expvar.Handler())

	// Apply Bearer token authentication middleware if token is configured or binding to 0.0.0.0
	var handler http.Handler = mux
	if cfg.ManagementToken != "" || cfg.ManagementHost == "0.0.0.0" {
		handler = ManagementAuthMiddleware(cfg.ManagementToken)(handler)
	}

	return handler
}

// ManagementAuthMiddleware protects management routes with Bearer token authentication.
func ManagementAuthMiddleware(requiredToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no token is required, pass through
			if requiredToken == "" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "unauthorized: missing or invalid bearer token",
				})
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != requiredToken {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "unauthorized: invalid token",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
