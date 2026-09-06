package stockhttp

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all stock and inventory management endpoints onto the router.
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/warehouses", func(wr chi.Router) {
		wr.Post("/", h.CreateWarehouse)
		wr.Get("/", h.ListWarehouses)
		wr.Get("/{id}", h.GetWarehouse)
		wr.Put("/{id}", h.UpdateWarehouse)
		wr.Delete("/{id}", h.DeleteWarehouse)
	})

	r.Route("/stock-locations", func(lr chi.Router) {
		lr.Post("/", h.CreateLocation)
		lr.Get("/", h.ListLocations)
		lr.Get("/{id}", h.GetLocation)
		lr.Put("/{id}", h.UpdateLocation)
		lr.Delete("/{id}", h.DeleteLocation)
	})

	r.Route("/stock-pickings", func(pr chi.Router) {
		pr.Post("/", h.CreatePicking)
		pr.Get("/", h.ListPickings)
		pr.Get("/{id}", h.GetPicking)
		pr.Put("/{id}", h.UpdatePicking)
		pr.Delete("/{id}", h.DeletePicking)
		pr.Post("/{id}/confirm", h.ConfirmPicking)
		pr.Post("/{id}/validate", h.ValidatePicking)
		pr.Post("/{id}/cancel", h.CancelPicking)
	})

	r.Route("/stock", func(sr chi.Router) {
		sr.Get("/on-hand", h.GetOnHandStock)
		sr.Get("/moves", h.ListMoves)
		sr.Post("/adjust", h.AdjustStock)
	})
}
