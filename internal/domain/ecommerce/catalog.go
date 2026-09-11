package ecommerce

import (
	"cashflow_backend/internal/platform/i18n"
)

// CategoryView is a read-optimized view of product categories for the web.
type CategoryView struct {
	ID          int64                  `json:"id"`
	ParentID    *int64                 `json:"parent_id,omitempty"`
	Name        i18n.TranslationString `json:"name"`
	Slug        string                 `json:"slug"`
	Description i18n.TranslationString `json:"description,omitempty"`
	ImageURL    *string                `json:"image_url,omitempty"`
	Sequence    int                    `json:"sequence"`
}

// CatalogFilter represents search and filter criteria for the web catalog.
type CatalogFilter struct {
	CategoryID *int64   `json:"category_id,omitempty"`
	Search     string   `json:"search,omitempty"`
	MinPrice   *float64 `json:"min_price,omitempty"`
	MaxPrice   *float64 `json:"max_price,omitempty"`
	Attributes map[string][]string `json:"attributes,omitempty"`
	SortBy     string   `json:"sort_by,omitempty"`
}
