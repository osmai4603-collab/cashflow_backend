package httpadapter

import (
	"crypto/subtle"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"cashflow_backend/internal/infrastructure/runtime/metrics"
	platconfig "cashflow_backend/internal/platform/config"
)

// NewManagementRouter builds the isolated observability listener. It only
// serves monitoring and management routes — never business APIs — and is
// intended to be bound to 127.0.0.1 on its own port.
//
// Security posture:
//   - metrics/pprof live ONLY here, never on the public router;
//   - pprof is served only when management.pprof_enabled=true (static config,
//     never via query params or headers);
//   - when require_auth is set, every request demands a Bearer token compared
//     in constant time; rejections are logged without any header/body content.
func NewManagementRouter(
	cfg *platconfig.Configuration,
	health HealthRoutes,
	logger *slog.Logger,
) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	if cfg.Management.RequireAuth {
		r.Use(managementAuthMiddleware(cfg.Management.AuthToken, logger))
	}

	r.Get("/", metrics.IndexPage().ServeHTTP)

	if cfg.Management.MetricsEnabled {
		r.Get("/metrics", metrics.PrometheusHandler().ServeHTTP)
		r.Get("/metrics/json", metrics.Handler)
		r.Post("/rum", NewRUMHandler(logger))
	}

	if health != nil {
		r.Get("/livez", health.HandleLiveness)
		r.Get("/readyz", health.HandleReadiness)
	}

	if cfg.Management.PprofEnabled {
		r.Mount("/debug", middleware.Profiler())
	}

	return r
}

// managementAuthMiddleware enforces a constant-time Bearer token check. The
// remote address and path are logged on rejection; the Authorization header
// and request body are never logged.
func managementAuthMiddleware(token string, logger *slog.Logger) func(http.Handler) http.Handler {
	expected := []byte("Bearer " + token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := []byte(r.Header.Get("Authorization"))
			if token == "" || subtle.ConstantTimeCompare(got, expected) != 1 {
				if logger != nil {
					logger.Warn("management auth rejected",
						"remote_addr", r.RemoteAddr,
						"path", r.URL.Path,
					)
				}
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
