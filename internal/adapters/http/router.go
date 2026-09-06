package httpadapter

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type HealthRoutes interface {
	HandleLiveness(w http.ResponseWriter, r *http.Request)
	HandleReadiness(w http.ResponseWriter, r *http.Request)
}

func NewRouter(handler *TransactionHandler, health HealthRoutes, logger *slog.Logger) chi.Router {
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(structuredLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health Endpoints (Phase 4: /livez and /readyz)
	if health != nil {
		r.Get("/livez", health.HandleLiveness)
		r.Get("/readyz", health.HandleReadiness)
	}

	// Root Endpoint
	r.Get("/", handler.Root)

	// API Endpoints
	r.Route("/api/v1/transactions", func(cr chi.Router) {
		cr.Post("/", handler.Create)
		cr.Get("/", handler.List)
		cr.Get("/summary", handler.Summary)
		cr.Get("/{id}", handler.GetByID)
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
