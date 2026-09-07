package activityhttp

import (
	"cashflow_backend/internal/platform/auth"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all Activity and Notification endpoints.
func RegisterRoutes(r chi.Router, h *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}

	r.Route("/activity-types", func(r chi.Router) {
		withAccess(r, authorizer, "mail.activity.type", auth.ActionRead).Get("/", h.ListTypes)
	})

	r.Route("/activities", func(r chi.Router) {
		withAccess(r, authorizer, "mail.activity", auth.ActionCreate).Post("/", h.Create)
		withAccess(r, authorizer, "mail.activity", auth.ActionRead).Get("/my", h.ListMy)
		r.Route("/{id}", func(r chi.Router) {
			withAccess(r, authorizer, "mail.activity", auth.ActionRead).Get("/", h.Get)
			withAccess(r, authorizer, "mail.activity", auth.ActionWrite).Post("/done", h.Complete)
		})
	})

	r.Route("/notifications", func(r chi.Router) {
		withAccess(r, authorizer, "mail.notification", auth.ActionRead).Get("/", h.ListNotifications)
		withAccess(r, authorizer, "mail.notification", auth.ActionRead).Get("/stream", h.StreamNotifications)
		withAccess(r, authorizer, "mail.notification", auth.ActionWrite).Post("/mark-all-read", h.MarkAllNotifsRead)
		withAccess(r, authorizer, "mail.notification", auth.ActionWrite).Post("/{id}/mark-read", h.MarkNotifRead)
	})
}

func withAccess(r chi.Router, authorizer auth.Authorizer, model string, action auth.Action) chi.Router {
	if authorizer == nil {
		return r
	}
	return r.With(auth.RequireAccess(authorizer, model, action))
}
