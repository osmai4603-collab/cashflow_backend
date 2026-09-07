package analytichttp

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the Analytic Accounting endpoints on the given router.
func RegisterRoutes(r chi.Router, h *Handler) {
	// Plans (account.analytic.plan)
	r.Route("/analytic-plans", func(plans chi.Router) {
		plans.Get("/", h.ListPlans)
		plans.Post("/", h.CreatePlan)
		plans.Get("/{id}", h.GetPlan)
		plans.Put("/{id}", h.UpdatePlan)
		plans.Delete("/{id}", h.DeletePlan)
		plans.Get("/{id}/structure", h.GetPlanStructure)
		plans.Put("/{id}/applicability", h.SetApplicability)
	})

	// Accounts (account.analytic.account)
	r.Route("/analytic-accounts", func(accounts chi.Router) {
		accounts.Get("/", h.ListAccounts)
		accounts.Post("/", h.CreateAccount)
		accounts.Get("/{id}", h.GetAccount)
		accounts.Put("/{id}", h.UpdateAccount)
		accounts.Delete("/{id}", h.DeleteAccount)
		accounts.Get("/{id}/balance", h.GetAccountBalance)
		accounts.Get("/{id}/lines", h.ListAccountLines)
	})

	// Lines (account.analytic.line)
	r.Route("/analytic-lines", func(lines chi.Router) {
		lines.Get("/", h.ListLines)
		lines.Post("/", h.RegisterManualLine)
		lines.Get("/{id}", h.GetLine)
		lines.Put("/{id}", h.UpdateLine)
	})

	// Distribution models (account.analytic.distribution.model - G7)
	r.Route("/analytic-distribution-models", func(models chi.Router) {
		models.Get("/", h.ListDistributionModels)
		models.Post("/", h.CreateDistributionModel)
		models.Post("/match", h.MatchDistributionModel)
		models.Get("/{id}", h.GetDistributionModel)
		models.Put("/{id}", h.UpdateDistributionModel)
		models.Delete("/{id}", h.DeleteDistributionModel)
	})

	// Integrations
	r.Route("/analytic", func(analytic chi.Router) {
		analytic.Get("/relevant-plans", h.GetRelevantPlans)
		analytic.Get("/project-plan", h.GetProjectPlan)
		analytic.Put("/project-plan", h.SetProjectPlan)
	})

	// Move-line analytic integration (Phase 3)
	r.Route("/move-lines/{id}/analytic", func(moveLines chi.Router) {
		moveLines.Get("/", h.ListMoveLineAnalytic)
		moveLines.Post("/", h.DistributeMoveLine)
	})
}
