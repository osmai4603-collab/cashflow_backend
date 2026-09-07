package sequencehttp

import "github.com/go-chi/chi/v5"

// RegisterRoutes mounts all Sequence REST endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/sequences", func(s chi.Router) {
		s.Post("/", h.Create)
		s.Get("/", h.List)
		s.Get("/{id}", h.GetByID)
		s.Put("/{id}", h.Update)
		s.Delete("/{id}", h.Delete)
		s.Post("/{id}/next", h.NextNumber)
	})
}
