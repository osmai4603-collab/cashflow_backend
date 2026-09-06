package accountinghttp

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all accounting module endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler) {
	// Chart of Accounts
	r.Route("/accounts", func(ar chi.Router) {
		ar.Post("/", h.CreateAccount)
		ar.Get("/", h.ListAccounts)
		ar.Get("/{id}", h.GetAccount)
		ar.Put("/{id}", h.UpdateAccount)
		ar.Delete("/{id}", h.DeleteAccount)
	})

	// Journals
	r.Route("/journals", func(jr chi.Router) {
		jr.Post("/", h.CreateJournal)
		jr.Get("/", h.ListJournals)
		jr.Get("/{id}", h.GetJournal)
		jr.Put("/{id}", h.UpdateJournal)
		jr.Delete("/{id}", h.DeleteJournal)
	})

	// Taxes
	r.Route("/taxes", func(tr chi.Router) {
		tr.Post("/", h.CreateTax)
		tr.Get("/", h.ListTaxes)
		tr.Post("/compute", h.ComputeTax)
		tr.Get("/{id}", h.GetTax)
		tr.Put("/{id}", h.UpdateTax)
		tr.Delete("/{id}", h.DeleteTax)
	})

	// Payment Terms
	r.Route("/payment-terms", func(pr chi.Router) {
		pr.Post("/", h.CreatePaymentTerm)
		pr.Get("/", h.ListPaymentTerms)
		pr.Get("/{id}", h.GetPaymentTerm)
		pr.Put("/{id}", h.UpdatePaymentTerm)
		pr.Delete("/{id}", h.DeletePaymentTerm)
	})

	// Journal Entries & Invoices
	r.Route("/moves", func(mr chi.Router) {
		mr.Post("/", h.CreateJournalEntry)
		mr.Get("/", h.ListMoves)
		mr.Get("/{id}", h.GetMove)
		mr.Put("/{id}", h.UpdateMove)
		mr.Delete("/{id}", h.DeleteMove)
		mr.Post("/{id}/post", h.PostMove)
		mr.Post("/{id}/cancel", h.CancelMove)
		mr.Post("/{id}/reverse", h.ReverseMove)
	})

	// Convenient Invoice Endpoint
	r.Post("/invoices", h.CreateInvoice)

	// Financial Reports
	r.Route("/accounting/reports", func(rr chi.Router) {
		rr.Get("/trial-balance", h.GetTrialBalance)
		rr.Get("/profit-loss", h.GetProfitAndLoss)
		rr.Get("/balance-sheet", h.GetBalanceSheet)
		rr.Get("/general-ledger", h.GetGeneralLedger)
	})
}
