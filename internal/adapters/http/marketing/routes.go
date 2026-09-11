package marketinghttp

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/marketing", func(m chi.Router) {
		m.Get("/campaigns", h.ListCampaigns)
		m.Post("/campaigns", h.CreateCampaign)
		m.Get("/lists", h.ListMailingLists)
		m.Post("/lists", h.CreateMailingList)
		m.Post("/lists/{id}/contacts/import", h.ImportContacts)
		m.Get("/contacts/blacklist", h.ListBlacklist)
		m.Post("/contacts/blacklist", h.AddToBlacklist)

		m.Post("/mailings", h.CreateMassMailing)
		m.Post("/mailings/{id}/send-test", h.SendTestMailing)
		m.Post("/mailings/{id}/schedule", h.ScheduleMailing)
		m.Post("/mailings/{id}/cancel", h.CancelMailing)
		m.Get("/mailings/{id}/stats", h.GetMailingStats)

		m.Get("/automations", h.ListAutomations)
		m.Post("/automations", h.CreateAutomation)
		m.Put("/automations/{id}", h.UpdateAutomation)
		m.Post("/automations/{id}/trigger-test", h.TriggerTestAutomation)
	})
}

func RegisterPublicRoutes(r chi.Router, h *Handler) {
	r.Route("/public/marketing", func(p chi.Router) {
		p.Get("/track/open/{code}.gif", h.TrackOpen)
		p.Get("/track/click/{code}", h.TrackClick)
		p.Get("/unsubscribe/{token}", h.Unsubscribe)
	})
}
