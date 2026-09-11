package ecommercehttp

import (
	"github.com/go-chi/chi/v5"
)

func RegisterPublicRoutes(r chi.Router, h *Handler) {
	r.Route("/ecommerce", func(e chi.Router) {
		e.Route("/catalog", func(c chi.Router) {
			c.Get("/categories", h.GetCategories)
			c.Get("/products", h.GetProducts)
			c.Get("/products/{id}", h.GetProduct)
		})
		e.Route("/cart", func(c chi.Router) {
			c.Get("/", h.GetCart)
			c.Post("/item", h.AddToCart)
			c.Put("/item/{line_id}", h.UpdateCartLine)
			c.Delete("/item/{line_id}", h.RemoveFromCart)
			c.Post("/apply-coupon", h.ApplyCoupon)
		})
		e.Route("/checkout", func(c chi.Router) {
			c.Post("/addresses", h.CheckoutAddresses)
			c.Post("/shipping", h.CheckoutShipping)
			c.Post("/pay", h.Pay)
		})
	})
}
