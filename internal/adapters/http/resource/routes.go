package resourcehttp

import "github.com/go-chi/chi/v5"

func RegisterRoutes(router chi.Router, handler *Handler) {
	router.Route("/resources", func(route chi.Router) {
		route.Post("/calendars", handler.CreateCalendar)
		route.Post("/work-entries", handler.CreateWorkEntry)
		route.Get("/work-entries", handler.ListWorkEntries)
	})
}
