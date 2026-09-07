package currencyhttp

import "github.com/go-chi/chi/v5"

// RegisterRoutes mounts all Currency REST endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/currencies", func(c chi.Router) {
		c.Post("/convert", h.Convert)
		c.Post("/", h.Create)
		c.Get("/", h.List)
		c.Get("/{id}", h.GetByID)
		c.Put("/{id}", h.Update)
		c.Delete("/{id}", h.Delete)
		c.Get("/{id}/rates", h.ListRates)
		c.Post("/{id}/rates", h.CreateRate)
	})
}
