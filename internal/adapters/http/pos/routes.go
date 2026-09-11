package poshttp

import "github.com/go-chi/chi/v5"

func RegisterRoutes(router chi.Router, handler *Handler) {
	router.Route("/pos", func(posRouter chi.Router) {
		posRouter.Post("/configs", handler.CreateConfig)
		posRouter.Post("/sessions", handler.OpenSession)
		posRouter.Post("/sessions/{id}/close", handler.CloseSession)
		posRouter.Post("/orders", handler.CreateOrder)
		posRouter.Post("/orders/sync", handler.SyncOrders)
	})
}
