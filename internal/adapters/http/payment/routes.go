package paymenthttp

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all Payment endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/payments", func(pr chi.Router) {
		pr.Post("/", h.CreatePayment)
		pr.Get("/", h.ListPayments)

		// Static routes before parameterized routes
		pr.Get("/receivable", h.GetReceivableAging)
		pr.Get("/payable", h.GetPayableAging)

		// Parameterized routes
		pr.Get("/{id}", h.GetPayment)
		pr.Put("/{id}", h.UpdatePayment)
		pr.Delete("/{id}", h.DeletePayment)
		pr.Post("/{id}/post", h.PostPayment)
		pr.Post("/{id}/cancel", h.CancelPayment)
		pr.Post("/{id}/reconcile", h.ReconcilePayment)
	})
}
