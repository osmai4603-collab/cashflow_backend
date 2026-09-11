package ecommerce

import (
	"context"
	"cashflow_backend/internal/platform/pagination"
)

type Repository interface {
	// Catalog
	ListCategories(ctx context.Context, websiteID int64) ([]CategoryView, error)
	ListProducts(ctx context.Context, websiteID int64, filter CatalogFilter, page pagination.PageRequest) (pagination.PageResult[ProductWebView], error)
	GetProductBySlug(ctx context.Context, websiteID int64, slug string) (*ProductWebView, error)

	// Cart
	GetCartBySession(ctx context.Context, websiteID int64, sessionUUID string) (*Cart, error)
	GetCartByID(ctx context.Context, id int64) (*Cart, error)
	SaveCart(ctx context.Context, cart *Cart) error
	DeleteCart(ctx context.Context, id int64) error

	// Checkout integration
	CreateSaleOrderFromCart(ctx context.Context, cartID int64) (int64, string, error)
}

type Service interface {
	AddToCart(ctx context.Context, websiteID int64, sessionUUID string, productID int64, qty float64) (*Cart, error)
	UpdateCartLine(ctx context.Context, cartID int64, lineID int64, qty float64) (*Cart, error)
	ApplyCoupon(ctx context.Context, cartID int64, code string) error
	ProcessCheckout(ctx context.Context, req CheckoutRequest) (*CheckoutResult, error)
}
