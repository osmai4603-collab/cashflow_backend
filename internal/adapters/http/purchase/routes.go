package purchasehttp

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all purchase module endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/purchase-orders", func(pr chi.Router) {
		pr.Post("/", h.CreateOrder)
		pr.Get("/", h.ListOrders)
		pr.Get("/{id}", h.GetOrder)
		pr.Put("/{id}", h.UpdateOrder)
		pr.Delete("/{id}", h.DeleteOrder)
		pr.Post("/{id}/send", h.ActionSend)
		pr.Post("/{id}/confirm", h.ConfirmOrder)
		pr.Post("/{id}/cancel", h.CancelOrder)
		pr.Post("/{id}/draft", h.ResetToDraft)
		pr.Post("/{id}/bill", h.CreateBill)
		pr.Get("/{id}/bills", h.GetOrderBills)
	})
}
