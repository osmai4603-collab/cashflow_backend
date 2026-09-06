package partnerhttp

import "github.com/go-chi/chi/v5"

// RegisterRoutes mounts all Partner REST endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/partners", func(p chi.Router) {
		p.Post("/", h.Create)
		p.Get("/", h.List)
		p.Get("/customers", h.ListCustomers)
		p.Get("/suppliers", h.ListSuppliers)
		p.Get("/{id}", h.GetByID)
		p.Put("/{id}", h.Update)
		p.Delete("/{id}", h.Delete)
	})
}
