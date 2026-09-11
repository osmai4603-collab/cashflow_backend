package helpdeskhttp

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/helpdesk", func(he chi.Router) {
		he.Get("/tickets", h.ListTickets)
		he.Post("/tickets", h.CreateTicket)
		he.Get("/tickets/{id}", h.GetTicket)
		he.Put("/tickets/{id}/stage", h.MoveToStage)
		he.Post("/tickets/{id}/assign", h.AssignTicket)
		he.Post("/tickets/{id}/reply", h.Reply)

		he.Post("/knowledge/articles", h.CreateArticle)
		he.Put("/knowledge/articles/{id}", h.UpdateArticle)
	})
}

func RegisterPublicRoutes(r chi.Router, h *Handler) {
	r.Route("/public/knowledge", func(p chi.Router) {
		p.Get("/articles", h.ListArticles)
		p.Get("/articles/{slug}", h.GetArticle)
	})
}
