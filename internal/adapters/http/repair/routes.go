package repairhttp

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/repairs", func(re chi.Router) {
		re.Get("/", h.List)
		re.Post("/", h.Create)
		re.Post("/{id}/confirm", h.Confirm)
		re.Post("/{id}/complete", h.Complete)
		re.Post("/{id}/create-invoice", h.CreateInvoice)
	})
}
