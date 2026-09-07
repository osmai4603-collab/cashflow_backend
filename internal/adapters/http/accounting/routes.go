package accountinghttp

import (
	"cashflow_backend/internal/platform/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all accounting module endpoints onto the provided Chi router.
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
	// Chart of Accounts
	r.Route("/accounts", func(ar chi.Router) {
		access(ar, "account.account", auth.ActionCreate).Post("/", h.CreateAccount)
		access(ar, "account.account", auth.ActionRead).Get("/", h.ListAccounts)
		access(ar, "account.account", auth.ActionRead).Get("/{id}", h.GetAccount)
		access(ar, "account.account", auth.ActionWrite).Put("/{id}", h.UpdateAccount)
		access(ar, "account.account", auth.ActionUnlink).Delete("/{id}", h.DeleteAccount)
	})

	// Journals
	r.Route("/journals", func(jr chi.Router) {
		access(jr, "account.journal", auth.ActionCreate).Post("/", h.CreateJournal)
		access(jr, "account.journal", auth.ActionRead).Get("/", h.ListJournals)
		access(jr, "account.journal", auth.ActionRead).Get("/{id}", h.GetJournal)
		access(jr, "account.journal", auth.ActionWrite).Put("/{id}", h.UpdateJournal)
		access(jr, "account.journal", auth.ActionUnlink).Delete("/{id}", h.DeleteJournal)
	})

	// Taxes
	r.Route("/taxes", func(tr chi.Router) {
		access(tr, "account.tax", auth.ActionCreate).Post("/", h.CreateTax)
		access(tr, "account.tax", auth.ActionRead).Get("/", h.ListTaxes)
		access(tr, "account.tax", auth.ActionRead).Post("/compute", h.ComputeTax)
		access(tr, "account.tax", auth.ActionRead).Get("/{id}", h.GetTax)
		access(tr, "account.tax", auth.ActionWrite).Put("/{id}", h.UpdateTax)
		access(tr, "account.tax", auth.ActionUnlink).Delete("/{id}", h.DeleteTax)
	})

	// Payment Terms
	r.Route("/payment-terms", func(pr chi.Router) {
		access(pr, "account.payment_term", auth.ActionCreate).Post("/", h.CreatePaymentTerm)
		access(pr, "account.payment_term", auth.ActionRead).Get("/", h.ListPaymentTerms)
		access(pr, "account.payment_term", auth.ActionRead).Get("/{id}", h.GetPaymentTerm)
		access(pr, "account.payment_term", auth.ActionWrite).Put("/{id}", h.UpdatePaymentTerm)
		access(pr, "account.payment_term", auth.ActionUnlink).Delete("/{id}", h.DeletePaymentTerm)
	})

	// Journal Entries & Invoices
	r.Route("/moves", func(mr chi.Router) {
		access(mr, "account.move", auth.ActionCreate).Post("/", h.CreateJournalEntry)
		access(mr, "account.move", auth.ActionRead).Get("/", h.ListMoves)
		access(mr, "account.move", auth.ActionRead).Get("/{id}", h.GetMove)
		access(mr, "account.move", auth.ActionWrite).Put("/{id}", h.UpdateMove)
		access(mr, "account.move", auth.ActionUnlink).Delete("/{id}", h.DeleteMove)
		access(mr, "account.move", auth.ActionWrite).Post("/{id}/post", h.PostMove)
		access(mr, "account.move", auth.ActionWrite).Post("/{id}/cancel", h.CancelMove)
		access(mr, "account.move", auth.ActionWrite).Post("/{id}/reverse", h.ReverseMove)

		// EDI (Electronic Data Interchange)
		mr.Route("/{id}/edi", func(er chi.Router) {
			access(er, "account.edi.document", auth.ActionWrite).Post("/process", h.ProcessMoveEDI)
			access(er, "account.edi.document", auth.ActionRead).Get("/", h.GetMoveEDIDocuments)
		})
	})

	// Convenient Invoice Endpoint
	access(r, "account.move", auth.ActionCreate).Post("/invoices", h.CreateInvoice)

	// Financial Reports
	r.Route("/accounting/reports", func(rr chi.Router) {
		access(rr, "account.report", auth.ActionRead).Get("/trial-balance", h.GetTrialBalance)
		access(rr, "account.report", auth.ActionRead).Get("/profit-loss", h.GetProfitAndLoss)
		access(rr, "account.report", auth.ActionRead).Get("/balance-sheet", h.GetBalanceSheet)
		access(rr, "account.report", auth.ActionRead).Get("/general-ledger", h.GetGeneralLedger)
	})
}
