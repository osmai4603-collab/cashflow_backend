package app

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"cashflow_backend/internal/infrastructure/health"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(logger *slog.Logger, healthChecker *health.HealthChecker) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// Health probes
	if healthChecker != nil {
		r.Get("/livez", healthChecker.HandleLiveness)
		r.Get("/readyz", healthChecker.HandleReadiness)
	}

	// Root API Info
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"app":     "cashflow_backend",
			"status":  "running",
			"router":  "chi/v5",
			"version": "1.0.0",
		})
	})

	return r
}
