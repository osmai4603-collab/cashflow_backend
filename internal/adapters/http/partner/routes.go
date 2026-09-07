package partnerhttp

import (
	"github.com/go-chi/chi/v5"

	"cashflow_backend/internal/platform/auth"
)

// RegisterRoutes mounts all Partner REST endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}
	r.Route("/partners", func(p chi.Router) {
		withAccess(p, authorizer, auth.ActionCreate).Post("/", h.Create)
		withAccess(p, authorizer, auth.ActionRead).Get("/", h.List)
		withAccess(p, authorizer, auth.ActionRead).Get("/customers", h.ListCustomers)
		withAccess(p, authorizer, auth.ActionRead).Get("/suppliers", h.ListSuppliers)
		withAccess(p, authorizer, auth.ActionRead).Get("/{id}", h.GetByID)
		withAccess(p, authorizer, auth.ActionWrite).Put("/{id}", h.Update)
		withAccess(p, authorizer, auth.ActionUnlink).Delete("/{id}", h.Delete)
	})
}

func withAccess(r chi.Router, authorizer auth.Authorizer, action auth.Action) chi.Router {
	if authorizer == nil {
		return r
	}
	return r.With(auth.RequireAccess(authorizer, "res.partner", action))
}
