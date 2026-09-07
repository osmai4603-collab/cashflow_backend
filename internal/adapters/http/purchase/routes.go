package purchasehttp

import (
	"cashflow_backend/internal/platform/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all purchase module endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}
	r.Route("/purchase-orders", func(pr chi.Router) {
		access := func(action auth.Action) chi.Router { return withAccess(pr, authorizer, "purchase.order", action) }
		access(auth.ActionCreate).Post("/", h.CreateOrder)
		access(auth.ActionRead).Get("/", h.ListOrders)
		access(auth.ActionRead).Get("/{id}", h.GetOrder)
		access(auth.ActionWrite).Put("/{id}", h.UpdateOrder)
		access(auth.ActionUnlink).Delete("/{id}", h.DeleteOrder)
		access(auth.ActionWrite).Post("/{id}/send", h.ActionSend)
		access(auth.ActionWrite).Post("/{id}/confirm", h.ConfirmOrder)
		access(auth.ActionWrite).Post("/{id}/cancel", h.CancelOrder)
		access(auth.ActionWrite).Post("/{id}/draft", h.ResetToDraft)
		access(auth.ActionWrite).Post("/{id}/bill", h.CreateBill)
		access(auth.ActionRead).Get("/{id}/bills", h.GetOrderBills)
	})

	r.Route("/purchase-requisitions", func(rr chi.Router) {
		access := func(action auth.Action) chi.Router { return withAccess(rr, authorizer, "purchase.requisition", action) }
		access(auth.ActionCreate).Post("/", h.CreateRequisition)
		access(auth.ActionRead).Get("/", h.ListRequisitions)
		access(auth.ActionRead).Get("/{id}", h.GetRequisition)
		access(auth.ActionWrite).Put("/{id}", h.UpdateRequisition)
		access(auth.ActionUnlink).Delete("/{id}", h.DeleteRequisition)
		access(auth.ActionWrite).Post("/{id}/confirm", h.ConfirmRequisition)
		access(auth.ActionWrite).Post("/{id}/close", h.CloseRequisition)
		access(auth.ActionWrite).Post("/{id}/cancel", h.CancelRequisition)
		access(auth.ActionWrite).Post("/{id}/create-po", h.CreatePOFromRequisition)
	})
}

func withAccess(r chi.Router, authorizer auth.Authorizer, model string, action auth.Action) chi.Router {
	if authorizer == nil {
		return r
	}
	return r.With(auth.RequireAccess(authorizer, model, action))
}
