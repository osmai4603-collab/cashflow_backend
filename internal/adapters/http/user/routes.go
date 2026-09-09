package userhttp

import "github.com/go-chi/chi/v5"

// RegisterRoutes mounts all User REST endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/users", func(u chi.Router) {
		u.Post("/", h.Create)
		u.Get("/", h.List)
		u.Get("/{id}", h.GetByID)
		u.Put("/{id}", h.Update)
		u.Delete("/{id}", h.Delete)
		u.Post("/logout", h.Logout)
	})
}
