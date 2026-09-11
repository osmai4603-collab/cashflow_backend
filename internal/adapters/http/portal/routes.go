package portalhttp

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/portal", func(p chi.Router) {
		p.Post("/auth/login", h.Login)
		p.Post("/auth/register-invite", h.RegisterInvite)

		p.Group(func(a chi.Router) {
			// In real app, auth middleware would be here
			a.Get("/dashboard/summary", h.GetDashboard)
			a.Get("/orders", h.ListOrders)
			a.Get("/orders/{id}", h.GetOrder)
			a.Get("/invoices", h.ListInvoices)
			a.Get("/invoices/{id}/pdf", h.GetInvoicePDF)
			a.Get("/deliveries", h.ListDeliveries)
		})
	})
}
