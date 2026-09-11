package subscriptionhttp

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/subscriptions", func(s chi.Router) {
		s.Get("/", h.List)
		s.Post("/", h.Create)
		s.Get("/{id}", h.Get)
		s.Post("/{id}/activate", h.Activate)
		s.Post("/{id}/cancel", h.Cancel)
		s.Post("/cron/process-billing", h.ProcessBilling)
		s.Get("/metrics/mrr", h.GetMRR)
	})
}
