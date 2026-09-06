package producthttp

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all product catalog endpoints on the parent Chi router.
func RegisterRoutes(r chi.Router, h *Handler) {
	// Product Templates
	r.Route("/products", func(pr chi.Router) {
		pr.Post("/", h.CreateProduct)
		pr.Get("/", h.ListProducts)
		pr.Get("/{id}", h.GetProduct)
		pr.Put("/{id}", h.UpdateProduct)
		pr.Delete("/{id}", h.DeleteProduct)
		pr.Post("/{id}/variants", h.CreateProductVariant)
		pr.Get("/{id}/variants", h.GetProductVariants)
	})

	// Standalone Variants
	r.Route("/variants", func(vr chi.Router) {
		vr.Get("/{id}", h.GetVariant)
		vr.Delete("/{id}", h.DeleteVariant)
	})

	// Product Categories
	r.Route("/product-categories", func(cr chi.Router) {
		cr.Post("/", h.CreateCategory)
		cr.Get("/", h.ListCategories)
		cr.Get("/{id}", h.GetCategory)
		cr.Put("/{id}", h.UpdateCategory)
		cr.Delete("/{id}", h.DeleteCategory)
	})

	// Units of Measure (UoM)
	r.Route("/uom", func(ur chi.Router) {
		ur.Post("/", h.CreateUoM)
		ur.Get("/", h.ListUoMs)
		ur.Get("/{id}", h.GetUoM)
		ur.Put("/{id}", h.UpdateUoM)
		ur.Delete("/{id}", h.DeleteUoM)
	})

	// Pricelists
	r.Route("/pricelists", func(plr chi.Router) {
		plr.Post("/", h.CreatePricelist)
		plr.Get("/", h.ListPricelists)
		plr.Get("/{id}", h.GetPricelist)
		plr.Put("/{id}", h.UpdatePricelist)
		plr.Delete("/{id}", h.DeletePricelist)
		plr.Post("/{id}/items", h.AddPricelistItem)
		plr.Delete("/{id}/items/{itemId}", h.DeletePricelistItem)
		plr.Post("/{id}/compute-price", h.ComputePrice)
	})
}
