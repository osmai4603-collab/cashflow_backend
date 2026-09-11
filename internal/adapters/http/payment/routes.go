package paymenthttp

import (
	"cashflow_backend/internal/platform/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all Payment endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}
	access := func(r chi.Router, action auth.Action) chi.Router {
		if authorizer == nil {
			return r
		}
		return r.With(auth.RequireAccess(authorizer, "account.payment", action))
	}
	r.Route("/payments", func(pr chi.Router) {
		access(pr, auth.ActionCreate).Post("/", h.CreatePayment)
		access(pr, auth.ActionRead).Get("/", h.ListPayments)
		access(pr, auth.ActionCreate).Post("/initiate", h.InitiateTransaction)
		access(pr, auth.ActionWrite).Post("/webhook/{code}", h.ProcessTransactionWebhook)

		// Static routes before parameterized routes
		access(pr, auth.ActionRead).Get("/receivable", h.GetReceivableAging)
		access(pr, auth.ActionRead).Get("/payable", h.GetPayableAging)

		// Parameterized routes
		access(pr, auth.ActionRead).Get("/{id}", h.GetPayment)
		access(pr, auth.ActionWrite).Put("/{id}", h.UpdatePayment)
		access(pr, auth.ActionUnlink).Delete("/{id}", h.DeletePayment)
		access(pr, auth.ActionWrite).Post("/{id}/post", h.PostPayment)
		access(pr, auth.ActionWrite).Post("/{id}/cancel", h.CancelPayment)
		access(pr, auth.ActionWrite).Post("/{id}/reconcile", h.ReconcilePayment)
		access(pr, auth.ActionWrite).Post("/{id}/capture", h.CaptureTransaction)
		access(pr, auth.ActionWrite).Post("/{id}/void", h.VoidTransaction)
	})
}
