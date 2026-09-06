package httpadapter

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"cashflow_backend/internal/adapters/http/partner"
	producthttp "cashflow_backend/internal/adapters/http/product"
	"cashflow_backend/internal/platform/response"
)

// HealthRoutes defines the liveness and readiness probe endpoints.
type HealthRoutes interface {
	HandleLiveness(w http.ResponseWriter, r *http.Request)
	HandleReadiness(w http.ResponseWriter, r *http.Request)
}

// NewRouter initializes and configures the HTTP router with standard middlewares.
func NewRouter(
	handler *BaseHandler,
	health HealthRoutes,
	partnerHandler *partnerhttp.Handler,
	productHandler *producthttp.Handler,
	logger *slog.Logger,
) chi.Router {
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(structuredLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health Endpoints (/livez and /readyz)
	if health != nil {
		r.Get("/livez", health.HandleLiveness)
		r.Get("/readyz", health.HandleReadiness)
	}

	// Root Endpoint
	r.Get("/", handler.Root)

	// API v1 Mount Point
	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Get("/", func(w http.ResponseWriter, r *http.Request) {
			response.JSON(w, http.StatusOK, map[string]string{
				"message": "ERP API v1 is online and ready for business modules",
			})
		})

		// Business Modules
		if partnerHandler != nil {
			partnerhttp.RegisterRoutes(v1, partnerHandler)
		}
		if productHandler != nil {
			producthttp.RegisterRoutes(v1, productHandler)
		}
	})

	return r
}

func structuredLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
