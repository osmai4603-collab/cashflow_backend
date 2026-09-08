package deliveryhttp

import (
	"cashflow_backend/internal/platform/auth"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all delivery and carrier endpoints.
func RegisterRoutes(r chi.Router, h *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	r.Route("/delivery", func(dr chi.Router) {
		access := func(action auth.Action) chi.Router { return withAccess(dr, authorizer, action) }

		// Carrier management
		access(auth.ActionRead).Get("/carriers", h.ListCarriers)
		access(auth.ActionWrite).Post("/rate", h.CalculateRate)

		// Sale Order delivery actions
		access(auth.ActionWrite).Post("/sale-orders/{orderID}/rate", h.CalculateOrderRate)
		access(auth.ActionWrite).Post("/sale-orders/{orderID}/set", h.SetOrderDelivery)
	})
}

func withAccess(r chi.Router, authorizer auth.Authorizer, action auth.Action) chi.Router {
	if authorizer == nil {
		return r
	}
	return r.With(auth.RequireAccess(authorizer, "delivery.carrier", action))
}
