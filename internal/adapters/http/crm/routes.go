package crmhttp

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all CRM and sales pipeline endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler) {
	// Leads & Opportunities
	r.Route("/leads", func(lr chi.Router) {
		lr.Post("/", h.CreateLead)
		lr.Get("/", h.ListLeads)
		lr.Get("/{id}", h.GetLead)
		lr.Put("/{id}", h.UpdateLead)
		lr.Delete("/{id}", h.DeleteLead)
		lr.Post("/{id}/convert", h.ConvertLead)
		lr.Post("/{id}/won", h.MarkWon)
		lr.Post("/{id}/lost", h.MarkLost)
	})

	// CRM Pipeline, Analytics & Configurations
	r.Route("/crm", func(cr chi.Router) {
		cr.Get("/pipeline", h.GetPipeline)
		cr.Get("/stats", h.GetStats)

		cr.Route("/stages", func(sr chi.Router) {
			sr.Post("/", h.CreateStage)
			sr.Get("/", h.ListStages)
			sr.Get("/{id}", h.GetStage)
			sr.Put("/{id}", h.UpdateStage)
			sr.Delete("/{id}", h.DeleteStage)
		})

		cr.Route("/lost-reasons", func(lrr chi.Router) {
			lrr.Post("/", h.CreateLostReason)
			lrr.Get("/", h.ListLostReasons)
			lrr.Get("/{id}", h.GetLostReason)
			lrr.Put("/{id}", h.UpdateLostReason)
			lrr.Delete("/{id}", h.DeleteLostReason)
		})

		cr.Route("/tags", func(tr chi.Router) {
			tr.Post("/", h.CreateTag)
			tr.Get("/", h.ListTags)
		})
	})
}
