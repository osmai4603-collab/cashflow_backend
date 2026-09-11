package websitehttp

import (
	"github.com/go-chi/chi/v5"
)

func RegisterPublicRoutes(r chi.Router, h *Handler) {
	r.Route("/website", func(w chi.Router) {
		w.Get("/sites/{id}", h.GetSite)
		w.Get("/pages/{slug}", h.GetPage)
		w.Get("/menus", h.GetMenus)
	})
}
