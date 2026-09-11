package qualityhttp

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/quality", func(q chi.Router) {
		q.Get("/control-points", h.ListPoints)
		q.Post("/control-points", h.CreatePoint)
		q.Post("/checks/execute", h.ExecuteCheck)
		q.Get("/alerts", h.ListAlerts)
		q.Post("/alerts", h.CreateAlert)
	})
}
