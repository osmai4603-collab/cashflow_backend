package timesheethttp

import "github.com/go-chi/chi/v5"

func RegisterRoutes(router chi.Router, handler *Handler) {
	router.Route("/timesheets", func(route chi.Router) {
		route.Get("/", handler.ListEntries)
		route.Post("/", handler.CreateEntry)
		route.Post("/timer/start", handler.StartTimer)
		route.Post("/timer/stop", handler.StopTimer)
		route.Post("/submit", handler.Submit)
		route.Post("/approve", handler.Approve)
	})
}
