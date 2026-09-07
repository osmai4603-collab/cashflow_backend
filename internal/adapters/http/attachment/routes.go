package attachmenthttp

import "github.com/go-chi/chi/v5"

// RegisterRoutes mounts all Attachment REST endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/attachments", func(a chi.Router) {
		a.Post("/", h.Upload)
		a.Get("/", h.ListByModel)
		a.Get("/{id}", h.GetByID)
		a.Get("/{id}/download", h.Download)
		a.Delete("/{id}", h.Archive)
	})
}
