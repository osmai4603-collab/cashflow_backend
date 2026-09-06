package product

import (
	"context"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines the persistent storage contract for product catalog operations.
type Repository interface {
	// Product Templates
	CreateTemplate(ctx context.Context, pt *ProductTemplate) error
	GetTemplateByID(ctx context.Context, id int64) (*ProductTemplate, error)
	UpdateTemplate(ctx context.Context, pt *ProductTemplate) error
	DeleteTemplate(ctx context.Context, id int64) error
	ListTemplates(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[ProductTemplate], error)

	// Product Variants
	CreateVariant(ctx context.Context, pv *ProductVariant) error
	GetVariantByID(ctx context.Context, id int64) (*ProductVariant, error)
	GetVariantsByTemplateID(ctx context.Context, templateID int64) ([]ProductVariant, error)
	UpdateVariant(ctx context.Context, pv *ProductVariant) error
	DeleteVariant(ctx context.Context, id int64) error

	// Product Categories
	CreateCategory(ctx context.Context, c *ProductCategory) error
	GetCategoryByID(ctx context.Context, id int64) (*ProductCategory, error)
	UpdateCategory(ctx context.Context, c *ProductCategory) error
	DeleteCategory(ctx context.Context, id int64) error
	ListCategories(ctx context.Context) ([]ProductCategory, error)

	// Units of Measure (UoM)
	CreateUoM(ctx context.Context, u *UnitOfMeasure) error
	GetUoMByID(ctx context.Context, id int64) (*UnitOfMeasure, error)
	UpdateUoM(ctx context.Context, u *UnitOfMeasure) error
	DeleteUoM(ctx context.Context, id int64) error
	ListUoMs(ctx context.Context) ([]UnitOfMeasure, error)

	// Pricelists
	CreatePricelist(ctx context.Context, pl *Pricelist) error
	GetPricelistByID(ctx context.Context, id int64) (*Pricelist, error)
	UpdatePricelist(ctx context.Context, pl *Pricelist) error
	DeletePricelist(ctx context.Context, id int64) error
	ListPricelists(ctx context.Context) ([]Pricelist, error)

	// Pricelist Items
	AddPricelistItem(ctx context.Context, item *PricelistItem) error
	DeletePricelistItem(ctx context.Context, id int64) error
	GetPricelistItems(ctx context.Context, pricelistID int64) ([]PricelistItem, error)
}
