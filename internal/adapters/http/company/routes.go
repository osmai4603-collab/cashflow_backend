package companyhttp

import (
	"github.com/go-chi/chi/v5"

	"cashflow_backend/internal/platform/auth"
)

// RegisterRoutes mounts all Company REST endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}
	r.Route("/companies", func(c chi.Router) {
		withAccess(c, authorizer, auth.ActionCreate).Post("/", h.Create)
		withAccess(c, authorizer, auth.ActionRead).Get("/", h.List)
		withAccess(c, authorizer, auth.ActionRead).Get("/default", h.GetDefault)
		withAccess(c, authorizer, auth.ActionRead).Get("/{id}", h.GetByID)
		withAccess(c, authorizer, auth.ActionWrite).Put("/{id}", h.Update)
		withAccess(c, authorizer, auth.ActionUnlink).Delete("/{id}", h.Delete)
	})
}

func withAccess(r chi.Router, authorizer auth.Authorizer, action auth.Action) chi.Router {
	if authorizer == nil {
		return r
	}
	return r.With(auth.RequireAccess(authorizer, "res.company", action))
}
