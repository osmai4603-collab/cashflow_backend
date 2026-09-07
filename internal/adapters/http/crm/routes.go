package crmhttp

import (
	"cashflow_backend/internal/platform/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all CRM and sales pipeline endpoints onto the provided Chi router.
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
	// Leads & Opportunities
	r.Route("/leads", func(lr chi.Router) {
		access(lr, "crm.lead", auth.ActionCreate).Post("/", h.CreateLead)
		access(lr, "crm.lead", auth.ActionRead).Get("/", h.ListLeads)
		access(lr, "crm.lead", auth.ActionRead).Get("/{id}", h.GetLead)
		access(lr, "crm.lead", auth.ActionWrite).Put("/{id}", h.UpdateLead)
		access(lr, "crm.lead", auth.ActionUnlink).Delete("/{id}", h.DeleteLead)
		access(lr, "crm.lead", auth.ActionWrite).Post("/{id}/convert", h.ConvertLead)
		access(lr, "crm.lead", auth.ActionWrite).Post("/{id}/won", h.MarkWon)
		access(lr, "crm.lead", auth.ActionWrite).Post("/{id}/lost", h.MarkLost)
	})

	// CRM Pipeline, Analytics & Configurations
	r.Route("/crm", func(cr chi.Router) {
		access(cr, "crm.lead", auth.ActionRead).Get("/pipeline", h.GetPipeline)
		access(cr, "crm.lead", auth.ActionRead).Get("/stats", h.GetStats)

		cr.Route("/stages", func(sr chi.Router) {
			access(sr, "crm.stage", auth.ActionCreate).Post("/", h.CreateStage)
			access(sr, "crm.stage", auth.ActionRead).Get("/", h.ListStages)
			access(sr, "crm.stage", auth.ActionRead).Get("/{id}", h.GetStage)
			access(sr, "crm.stage", auth.ActionWrite).Put("/{id}", h.UpdateStage)
			access(sr, "crm.stage", auth.ActionUnlink).Delete("/{id}", h.DeleteStage)
		})

		cr.Route("/lost-reasons", func(lrr chi.Router) {
			access(lrr, "crm.lost_reason", auth.ActionCreate).Post("/", h.CreateLostReason)
			access(lrr, "crm.lost_reason", auth.ActionRead).Get("/", h.ListLostReasons)
			access(lrr, "crm.lost_reason", auth.ActionRead).Get("/{id}", h.GetLostReason)
			access(lrr, "crm.lost_reason", auth.ActionWrite).Put("/{id}", h.UpdateLostReason)
			access(lrr, "crm.lost_reason", auth.ActionUnlink).Delete("/{id}", h.DeleteLostReason)
		})

		cr.Route("/tags", func(tr chi.Router) {
			access(tr, "crm.tag", auth.ActionCreate).Post("/", h.CreateTag)
			access(tr, "crm.tag", auth.ActionRead).Get("/", h.ListTags)
		})
	})
}
