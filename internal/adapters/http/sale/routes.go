package salehttp

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all sales module endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/sale-orders", func(sr chi.Router) {
		sr.Post("/", h.CreateOrder)
		sr.Get("/", h.ListOrders)
		sr.Get("/{id}", h.GetOrder)
		sr.Put("/{id}", h.UpdateOrder)
		sr.Delete("/{id}", h.DeleteOrder)
		sr.Post("/{id}/send", h.ActionSend)
		sr.Post("/{id}/confirm", h.ConfirmOrder)
		sr.Post("/{id}/cancel", h.CancelOrder)
		sr.Post("/{id}/draft", h.ResetToDraft)
		sr.Post("/{id}/invoice", h.CreateInvoice)
		sr.Get("/{id}/invoices", h.GetOrderInvoices)
	})
}
