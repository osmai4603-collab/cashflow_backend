package mrphttp

import (
	"github.com/go-chi/chi/v5"
)

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/workcenters", func(r chi.Router) {
		r.Post("/", h.CreateWorkcenter)
		r.Get("/", h.ListWorkcenters)
		r.Get("/{id}", h.GetWorkcenter)
	})

	r.Route("/boms", func(r chi.Router) {
		r.Post("/", h.CreateBoM)
		r.Get("/", h.ListBoMs)
		r.Get("/{id}", h.GetBoM)
	})

	r.Route("/productions", func(r chi.Router) {
		r.Post("/", h.CreateProduction)
		r.Get("/", h.ListProductions)
		r.Get("/{id}", h.GetProduction)
		r.Post("/{id}/confirm", h.ConfirmProduction)
		r.Post("/{id}/produce", h.ProduceProduction)
	})

	r.Route("/workorders", func(r chi.Router) {
		r.Get("/", h.ListWorkorders)
		r.Post("/{id}/start", h.StartWorkorder)
		r.Post("/{id}/done", h.DoneWorkorder)
	})

	r.Route("/unbuild", func(r chi.Router) {
		r.Post("/", h.CreateUnbuild)
		r.Get("/{id}", h.GetUnbuild)
	})

	r.Get("/reports/oee", h.GetOEEReport)

	return r
}
