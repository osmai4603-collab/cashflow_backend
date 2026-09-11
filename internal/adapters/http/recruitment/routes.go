package recruitmenthttp

import "github.com/go-chi/chi/v5"

func RegisterRoutes(router chi.Router, handler *Handler) {
	router.Route("/hr/recruitment", func(route chi.Router) {
		route.Get("/stages", handler.ListStages)
		route.Get("/applicants", handler.ListApplicants)
		route.Post("/applicants", handler.CreateApplicant)
		route.Put("/applicants/{id}/stage", handler.MoveApplicant)
		route.Post("/applicants/{id}/hire", handler.HireApplicant)
		route.Post("/interviews", handler.CreateInterview)
	})
}