package bankstatementhttp

import (
	"cashflow_backend/internal/platform/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all Bank Statement endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}
	access := func(r chi.Router, model string, action auth.Action) chi.Router {
		if authorizer == nil {
			return r
		}
		return r.With(auth.RequireAccess(authorizer, model, action))
	}

	r.Route("/bank-statements", func(br chi.Router) {
		access(br, "account.bank.statement", auth.ActionCreate).Post("/", h.CreateStatement)
		access(br, "account.bank.statement", auth.ActionRead).Get("/", h.ListStatements)

		// Static routes before parameterized routes
		access(br, "account.bank.statement", auth.ActionCreate).Post("/import", h.ImportCSV)

		access(br, "account.bank.statement", auth.ActionRead).Get("/{id}", h.GetStatement)
		access(br, "account.bank.statement", auth.ActionWrite).Put("/{id}", h.UpdateStatement)
		access(br, "account.bank.statement", auth.ActionUnlink).Delete("/{id}", h.DeleteStatement)
		access(br, "account.bank.statement", auth.ActionWrite).Post("/{id}/confirm", h.ConfirmStatement)
		access(br, "account.bank.statement", auth.ActionWrite).Post("/{id}/auto-reconcile", h.AutoReconcile)
		access(br, "account.bank.statement", auth.ActionCreate).Post("/{id}/lines", h.AddLines)
	})

	r.Route("/bank-statement-lines", func(lr chi.Router) {
		access(lr, "account.bank.statement", auth.ActionRead).Get("/{id}/candidates", h.FindCandidates)
		access(lr, "account.bank.statement", auth.ActionWrite).Post("/{id}/reconcile", h.ReconcileLine)
		access(lr, "account.bank.statement", auth.ActionWrite).Post("/{id}/undo-reconcile", h.UndoReconcile)
	})

	r.Route("/reconcile-models", func(mr chi.Router) {
		access(mr, "account.reconcile.model", auth.ActionCreate).Post("/", h.CreateReconcileModel)
		access(mr, "account.reconcile.model", auth.ActionRead).Get("/", h.ListReconcileModels)
		access(mr, "account.reconcile.model", auth.ActionRead).Get("/{id}", h.GetReconcileModel)
		access(mr, "account.reconcile.model", auth.ActionWrite).Put("/{id}", h.UpdateReconcileModel)
		access(mr, "account.reconcile.model", auth.ActionUnlink).Delete("/{id}", h.DeleteReconcileModel)
	})

	r.Route("/cash-roundings", func(cr chi.Router) {
		access(cr, "account.cash.rounding", auth.ActionCreate).Post("/", h.CreateCashRounding)
		access(cr, "account.cash.rounding", auth.ActionRead).Get("/", h.ListCashRoundings)
		access(cr, "account.cash.rounding", auth.ActionRead).Get("/{id}", h.GetCashRounding)
		access(cr, "account.cash.rounding", auth.ActionWrite).Put("/{id}", h.UpdateCashRounding)
		access(cr, "account.cash.rounding", auth.ActionUnlink).Delete("/{id}", h.DeleteCashRounding)
	})
}
