package ecommerce

import (
	"cashflow_backend/internal/platform/i18n"
)

// ProductWebView represents product details optimized for the storefront.
type ProductWebView struct {
	ID              int64                  `json:"id"`
	Name            i18n.TranslationString `json:"name"`
	Slug            string                 `json:"slug"`
	SKU             string                 `json:"sku"`
	Description     i18n.TranslationString `json:"description"`
	WebsiteDesc     i18n.TranslationString `json:"website_description,omitempty"`
	ListPrice       float64                `json:"list_price"`
	SalesPrice      float64                `json:"sales_price"` // Price after pricelist application
	Currency        string                 `json:"currency"`
	ImageURLs       []string               `json:"image_urls"`
	IsAvailable     bool                   `json:"is_available"`
	InventoryQty    float64                `json:"inventory_qty"`
	Attributes      []ProductAttribute     `json:"attributes,omitempty"`
	MetaTitle       string                 `json:"meta_title,omitempty"`
	MetaDescription string                 `json:"meta_description,omitempty"`
}

type ProductAttribute struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}
