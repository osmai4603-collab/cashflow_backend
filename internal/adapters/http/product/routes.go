package producthttp

import (
	"github.com/go-chi/chi/v5"

	"cashflow_backend/internal/platform/auth"
)

// RegisterRoutes mounts all product catalog endpoints on the parent Chi router.
func RegisterRoutes(r chi.Router, h *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}
	variant := func(vr chi.Router, action auth.Action) chi.Router { return withAccess(vr, authorizer, "product.product", action) }
	// Product Templates
	r.Route("/products", func(pr chi.Router) {
		withAccess(pr, authorizer, "product.template", auth.ActionCreate).Post("/", h.CreateProduct)
		withAccess(pr, authorizer, "product.template", auth.ActionRead).Get("/", h.ListProducts)
		withAccess(pr, authorizer, "product.template", auth.ActionRead).Get("/{id}", h.GetProduct)
		withAccess(pr, authorizer, "product.template", auth.ActionWrite).Put("/{id}", h.UpdateProduct)
		withAccess(pr, authorizer, "product.template", auth.ActionUnlink).Delete("/{id}", h.DeleteProduct)
		withAccess(pr, authorizer, "product.product", auth.ActionCreate).Post("/{id}/variants", h.CreateProductVariant)
		withAccess(pr, authorizer, "product.product", auth.ActionRead).Get("/{id}/variants", h.GetProductVariants)
	})

	// Standalone Variants
	r.Route("/variants", func(vr chi.Router) {
		variant(vr, auth.ActionRead).Get("/{id}", h.GetVariant)
		variant(vr, auth.ActionUnlink).Delete("/{id}", h.DeleteVariant)
	})

	// Product Categories
	r.Route("/product-categories", func(cr chi.Router) {
		withAccess(cr, authorizer, "product.category", auth.ActionCreate).Post("/", h.CreateCategory)
		withAccess(cr, authorizer, "product.category", auth.ActionRead).Get("/", h.ListCategories)
		withAccess(cr, authorizer, "product.category", auth.ActionRead).Get("/{id}", h.GetCategory)
		withAccess(cr, authorizer, "product.category", auth.ActionWrite).Put("/{id}", h.UpdateCategory)
		withAccess(cr, authorizer, "product.category", auth.ActionUnlink).Delete("/{id}", h.DeleteCategory)
	})

	// Units of Measure (UoM)
	r.Route("/uom", func(ur chi.Router) {
		withAccess(ur, authorizer, "uom.uom", auth.ActionCreate).Post("/", h.CreateUoM)
		withAccess(ur, authorizer, "uom.uom", auth.ActionRead).Get("/", h.ListUoMs)
		withAccess(ur, authorizer, "uom.uom", auth.ActionRead).Get("/{id}", h.GetUoM)
		withAccess(ur, authorizer, "uom.uom", auth.ActionWrite).Put("/{id}", h.UpdateUoM)
		withAccess(ur, authorizer, "uom.uom", auth.ActionUnlink).Delete("/{id}", h.DeleteUoM)
	})

	// Pricelists
	r.Route("/pricelists", func(plr chi.Router) {
		withAccess(plr, authorizer, "product.pricelist", auth.ActionCreate).Post("/", h.CreatePricelist)
		withAccess(plr, authorizer, "product.pricelist", auth.ActionRead).Get("/", h.ListPricelists)
		withAccess(plr, authorizer, "product.pricelist", auth.ActionRead).Get("/{id}", h.GetPricelist)
		withAccess(plr, authorizer, "product.pricelist", auth.ActionWrite).Put("/{id}", h.UpdatePricelist)
		withAccess(plr, authorizer, "product.pricelist", auth.ActionUnlink).Delete("/{id}", h.DeletePricelist)
		withAccess(plr, authorizer, "product.pricelist", auth.ActionCreate).Post("/{id}/items", h.AddPricelistItem)
		withAccess(plr, authorizer, "product.pricelist", auth.ActionUnlink).Delete("/{id}/items/{itemId}", h.DeletePricelistItem)
		withAccess(plr, authorizer, "product.pricelist", auth.ActionRead).Post("/{id}/compute-price", h.ComputePrice)
	})
}

func withAccess(r chi.Router, authorizer auth.Authorizer, model string, action auth.Action) chi.Router {
	if authorizer == nil {
		return r
	}
	return r.With(auth.RequireAccess(authorizer, model, action))
}
