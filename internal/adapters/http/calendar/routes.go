package calendarhttp

import (
	"cashflow_backend/internal/platform/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}
	access := func(route chi.Router, model string, action auth.Action) chi.Router {
		if authorizer == nil {
			return route
		}
		return route.With(auth.RequireAccess(authorizer, model, action))
	}
	router.Route("/calendar", func(calendarRouter chi.Router) {
		access(calendarRouter, "calendar.event", auth.ActionCreate).Post("/events", handler.CreateEvent)
		access(calendarRouter, "calendar.event", auth.ActionRead).Get("/events", handler.ListEvents)
		access(calendarRouter, "calendar.event", auth.ActionRead).Get("/events/{id}", handler.GetEvent)
		access(calendarRouter, "calendar.event", auth.ActionWrite).Put("/events/{id}", handler.UpdateEvent)
		access(calendarRouter, "calendar.event", auth.ActionUnlink).Delete("/events/{id}", handler.DeleteEvent)
		access(calendarRouter, "calendar.event", auth.ActionRead).Get("/export/{id}.ics", handler.ExportICS)
		access(calendarRouter, "calendar.event", auth.ActionWrite).Post("/attendee/respond", handler.RespondAttendee)
		access(calendarRouter, "calendar.appointment_type", auth.ActionCreate).Post("/appointments/types", handler.CreateAppointmentType)
		access(calendarRouter, "calendar.appointment_type", auth.ActionRead).Get("/appointments/types", handler.ListAppointmentTypes)
	})
	router.Route("/public/appointments/{slug}", func(publicRouter chi.Router) {
		publicRouter.Get("/slots", handler.ListSlots)
		publicRouter.Post("/book", handler.BookAppointment)
	})
}
