package surveyhttp

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/surveys", func(s chi.Router) {
		s.Get("/", h.List)
		s.Post("/", h.Create)
	})
}

func RegisterPublicRoutes(r chi.Router, h *Handler) {
	r.Route("/public/surveys", func(p chi.Router) {
		p.Get("/{id}", h.Get)
		p.Post("/{id}/submit", h.Submit)
	})
}
