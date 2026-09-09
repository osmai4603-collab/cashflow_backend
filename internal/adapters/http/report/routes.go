package reporthttp

import (
	"cashflow_backend/internal/platform/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all report module endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}
	r.Route("/reports", func(rr chi.Router) {
		access := func(action auth.Action) chi.Router {
			if authorizer == nil {
				return rr
			}
			return rr.With(auth.RequireAccess(authorizer, "account.report", action))
		}
		access(auth.ActionRead).Get("/dashboard", h.GetDashboard)
		access(auth.ActionRead).Get("/{code}", h.GetReport)
	})
}