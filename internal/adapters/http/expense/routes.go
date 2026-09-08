package expensehttp

import (
	"cashflow_backend/internal/platform/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, handler *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}
	access := func(router chi.Router, model string, action auth.Action) chi.Router {
		if authorizer == nil {
			return router
		}
		return router.With(auth.RequireAccess(authorizer, model, action))
	}
	r.Route("/expenses", func(expenses chi.Router) {
		access(expenses, "hr.expense", auth.ActionCreate).Post("/", handler.Create)
		access(expenses, "hr.expense", auth.ActionRead).Get("/", handler.List)
		access(expenses, "hr.expense", auth.ActionRead).Get("/{id}", handler.Get)
		access(expenses, "hr.expense", auth.ActionWrite).Put("/{id}", handler.Update)
		access(expenses, "hr.expense", auth.ActionUnlink).Delete("/{id}", handler.Delete)
		access(expenses, "hr.expense", auth.ActionWrite).Post("/{id}/submit", handler.Submit)
		access(expenses, "hr.expense", auth.ActionWrite).Post("/{id}/approve", handler.Approve)
		access(expenses, "hr.expense", auth.ActionWrite).Post("/{id}/refuse", handler.Refuse)
		access(expenses, "hr.expense", auth.ActionWrite).Post("/{id}/split", handler.Split)
		access(expenses, "hr.expense", auth.ActionWrite).Post("/{id}/post", handler.Post)
	})
}
