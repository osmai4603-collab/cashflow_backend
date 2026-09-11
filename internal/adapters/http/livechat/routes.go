package livechathttp

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/livechat", func(l chi.Router) {
		l.Get("/sessions/active", h.ListActiveSessions)
		l.Post("/sessions/{id}/close", h.CloseSession)
		l.Post("/sessions/{id}/convert-ticket", h.ConvertToTicket)
	})
}

func RegisterPublicRoutes(r chi.Router, h *Handler) {
	r.Route("/public/livechat", func(p chi.Router) {
		p.Get("/channels/{id}", h.GetChannel)
		p.Post("/sessions/init", h.InitSession)
		p.Get("/ws/{session_id}", h.HandleWS)
		p.Post("/sessions/{id}/rate", h.RateSession)
	})
}
