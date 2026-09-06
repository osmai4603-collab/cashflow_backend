package productusecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// ─────────────────────────────────────────────────────────────────────────────
// Input DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateProductInput struct {
	Name        string              `json:"name"`
	Type        product.ProductType `json:"type"`
	CategoryID  *int64              `json:"category_id"`
	InternalRef string              `json:"internal_ref"`
	Barcode     string              `json:"barcode"`
	SalePrice   float64             `json:"sale_price"`
	CostPrice   float64             `json:"cost_price"`
	UoMID       *int64              `json:"uom_id"`
	SaleOK      *bool               `json:"sale_ok"`
	PurchaseOK  *bool               `json:"purchase_ok"`
	Weight      float64             `json:"weight"`
	Volume      float64             `json:"volume"`
	Description string              `json:"description"`
	CompanyID   *int64              `json:"company_id"`
}

type UpdateProductInput struct {
	Name        *string              `json:"name"`
	Type        *product.ProductType `json:"type"`
	CategoryID  *int64               `json:"category_id"`
	InternalRef *string              `json:"internal_ref"`
	Barcode     *string              `json:"barcode"`
	SalePrice   *float64             `json:"sale_price"`
	CostPrice   *float64             `json:"cost_price"`
	UoMID       *int64               `json:"uom_id"`
	SaleOK      *bool                `json:"sale_ok"`
	PurchaseOK  *bool                `json:"purchase_ok"`
	Weight      *float64             `json:"weight"`
	Volume      *float64             `json:"volume"`
	Description *string              `json:"description"`
	CompanyID   *int64               `json:"company_id"`
}

type CreateVariantInput struct {
	SKU        string  `json:"sku"`
	Barcode    string  `json:"barcode"`
	ExtraPrice float64 `json:"extra_price"`
}

type CreateCategoryInput struct {
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id"`
}

type UpdateCategoryInput struct {
	Name     *string `json:"name"`
	ParentID *int64  `json:"parent_id"`
}

type CreateUoMInput struct {
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Ratio    float64 `json:"ratio"`
	Rounding float64 `json:"rounding"`
}

type UpdateUoMInput struct {
	Name     *string  `json:"name"`
	Category *string  `json:"category"`
	Ratio    *float64 `json:"ratio"`
	Rounding *float64 `json:"rounding"`
}

type CreatePricelistInput struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

type UpdatePricelistInput struct {
	Name     *string `json:"name"`
	Currency *string `json:"currency"`
}

type CreatePricelistItemInput struct {
	AppliedOn    product.PricelistAppliedOn   `json:"applied_on"`
	CategoryID   *int64                       `json:"category_id"`
	TemplateID   *int64                       `json:"template_id"`
	VariantID    *int64                       `json:"variant_id"`
	MinQuantity  float64                      `json:"min_quantity"`
	ComputePrice product.PricelistComputeType `json:"compute_price"`
	FixedPrice   float64                      `json:"fixed_price"`
	PercentPrice float64                      `json:"percent_price"`
	DateStart    *time.Time                   `json:"date_start"`
	DateEnd      *time.Time                   `json:"date_end"`
}

// ─────────────────────────────────────────────────────────────────────────────
// UseCase Interface
// ─────────────────────────────────────────────────────────────────────────────

type UseCase interface {
	// Product Templates
	CreateProduct(ctx context.Context, in CreateProductInput) (*product.ProductTemplate, error)
	GetProduct(ctx context.Context, id int64) (*product.ProductTemplate, error)
	UpdateProduct(ctx context.Context, id int64, in UpdateProductInput) (*product.ProductTemplate, error)
	DeleteProduct(ctx context.Context, id int64) error
	ListProducts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[product.ProductTemplate], error)

	// Variants
	CreateProductVariant(ctx context.Context, templateID int64, in CreateVariantInput) (*product.ProductVariant, error)
	GetVariant(ctx context.Context, id int64) (*product.ProductVariant, error)
	GetProductVariants(ctx context.Context, templateID int64) ([]product.ProductVariant, error)
	DeleteVariant(ctx context.Context, id int64) error

	// Categories
	CreateCategory(ctx context.Context, in CreateCategoryInput) (*product.ProductCategory, error)
	GetCategory(ctx context.Context, id int64) (*product.ProductCategory, error)
	UpdateCategory(ctx context.Context, id int64, in UpdateCategoryInput) (*product.ProductCategory, error)
	DeleteCategory(ctx context.Context, id int64) error
	ListCategories(ctx context.Context) ([]product.ProductCategory, error)

	// Units of Measure (UoM)
	CreateUoM(ctx context.Context, in CreateUoMInput) (*product.UnitOfMeasure, error)
	GetUoM(ctx context.Context, id int64) (*product.UnitOfMeasure, error)
	UpdateUoM(ctx context.Context, id int64, in UpdateUoMInput) (*product.UnitOfMeasure, error)
	DeleteUoM(ctx context.Context, id int64) error
	ListUoMs(ctx context.Context) ([]product.UnitOfMeasure, error)

	// Pricelists
	CreatePricelist(ctx context.Context, in CreatePricelistInput) (*product.Pricelist, error)
	GetPricelist(ctx context.Context, id int64) (*product.Pricelist, error)
	UpdatePricelist(ctx context.Context, id int64, in UpdatePricelistInput) (*product.Pricelist, error)
	DeletePricelist(ctx context.Context, id int64) error
	ListPricelists(ctx context.Context) ([]product.Pricelist, error)
	AddPricelistItem(ctx context.Context, pricelistID int64, in CreatePricelistItemInput) (*product.PricelistItem, error)
	DeletePricelistItem(ctx context.Context, id int64) error
	ComputePrice(ctx context.Context, pricelistID int64, productID int64, variantID *int64, quantity float64) (float64, error)
}

// ProductUseCase implements UseCase.
type ProductUseCase struct {
	repo   product.Repository
	logger *slog.Logger
}

// New constructs a ProductUseCase.
func New(repo product.Repository, logger *slog.Logger) *ProductUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &ProductUseCase{
		repo:   repo,
		logger: logger,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Products & Templates
// ─────────────────────────────────────────────────────────────────────────────

func (uc *ProductUseCase) CreateProduct(ctx context.Context, in CreateProductInput) (*product.ProductTemplate, error) {
	// Verify Category exists if specified
	if in.CategoryID != nil && *in.CategoryID > 0 {
		if _, err := uc.repo.GetCategoryByID(ctx, *in.CategoryID); err != nil {
			return nil, platformerrors.Validation("category does not exist", map[string]string{
				"category_id": fmt.Sprintf("category %d not found", *in.CategoryID),
			})
		}
	}

	// Verify UoM exists if specified
	if in.UoMID != nil && *in.UoMID > 0 {
		if _, err := uc.repo.GetUoMByID(ctx, *in.UoMID); err != nil {
			return nil, platformerrors.Validation("unit of measure does not exist", map[string]string{
				"uom_id": fmt.Sprintf("unit of measure %d not found", *in.UoMID),
			})
		}
	}

	saleOK := true
	if in.SaleOK != nil {
		saleOK = *in.SaleOK
	}

	purchaseOK := true
	if in.PurchaseOK != nil {
		purchaseOK = *in.PurchaseOK
	}

	pt := &product.ProductTemplate{
		Name:        in.Name,
		Type:        in.Type,
		CategoryID:  in.CategoryID,
		InternalRef: in.InternalRef,
		Barcode:     in.Barcode,
		SalePrice:   in.SalePrice,
		CostPrice:   in.CostPrice,
		UoMID:       in.UoMID,
		SaleOK:      saleOK,
		PurchaseOK:  purchaseOK,
		Weight:      in.Weight,
		Volume:      in.Volume,
		Description: in.Description,
		CompanyID:   in.CompanyID,
		Active:      true,
		Audit:       audit.NewFields(ctx),
	}

	if err := pt.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreateTemplate(ctx, pt); err != nil {
		uc.logger.Error("failed to create product template", "error", err, "name", pt.Name)
		return nil, err
	}

	// Automatically create default single variant (matches Odoo product.product)
	defaultVariant := &product.ProductVariant{
		TemplateID: pt.ID,
		SKU:        pt.InternalRef,
		Barcode:    pt.Barcode,
		ExtraPrice: 0.0,
		Active:     true,
		Audit:      audit.NewFields(ctx),
	}
	if err := uc.repo.CreateVariant(ctx, defaultVariant); err != nil {
		uc.logger.Error("failed to create default product variant", "error", err, "template_id", pt.ID)
	}

	uc.logger.Info("product template created successfully", "id", pt.ID, "name", pt.Name)
	return pt, nil
}

func (uc *ProductUseCase) GetProduct(ctx context.Context, id int64) (*product.ProductTemplate, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid product id")
	}
	return uc.repo.GetTemplateByID(ctx, id)
}

func (uc *ProductUseCase) UpdateProduct(ctx context.Context, id int64, in UpdateProductInput) (*product.ProductTemplate, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid product id")
	}

	pt, err := uc.repo.GetTemplateByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		pt.Name = *in.Name
	}
	if in.Type != nil {
		pt.Type = *in.Type
	}
	if in.CategoryID != nil {
		if *in.CategoryID > 0 {
			if _, err := uc.repo.GetCategoryByID(ctx, *in.CategoryID); err != nil {
				return nil, platformerrors.Validation("category does not exist", map[string]string{
					"category_id": fmt.Sprintf("category %d not found", *in.CategoryID),
				})
			}
		}
		pt.CategoryID = in.CategoryID
	}
	if in.InternalRef != nil {
		pt.InternalRef = *in.InternalRef
	}
	if in.Barcode != nil {
		pt.Barcode = *in.Barcode
	}
	if in.SalePrice != nil {
		pt.SalePrice = *in.SalePrice
	}
	if in.CostPrice != nil {
		pt.CostPrice = *in.CostPrice
	}
	if in.UoMID != nil {
		if *in.UoMID > 0 {
			if _, err := uc.repo.GetUoMByID(ctx, *in.UoMID); err != nil {
				return nil, platformerrors.Validation("unit of measure does not exist", map[string]string{
					"uom_id": fmt.Sprintf("unit of measure %d not found", *in.UoMID),
				})
			}
		}
		pt.UoMID = in.UoMID
	}
	if in.SaleOK != nil {
		pt.SaleOK = *in.SaleOK
	}
	if in.PurchaseOK != nil {
		pt.PurchaseOK = *in.PurchaseOK
	}
	if in.Weight != nil {
		pt.Weight = *in.Weight
	}
	if in.Volume != nil {
		pt.Volume = *in.Volume
	}
	if in.Description != nil {
		pt.Description = *in.Description
	}
	if in.CompanyID != nil {
		pt.CompanyID = in.CompanyID
	}

	pt.Audit.Touch(ctx)

	if err := pt.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateTemplate(ctx, pt); err != nil {
		uc.logger.Error("failed to update product template", "error", err, "id", id)
		return nil, err
	}

	uc.logger.Info("product template updated successfully", "id", pt.ID)
	return pt, nil
}

func (uc *ProductUseCase) DeleteProduct(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.BadRequest("invalid product id")
	}

	if err := uc.repo.DeleteTemplate(ctx, id); err != nil {
		return err
	}

	uc.logger.Info("product template soft-deleted successfully", "id", id)
	return nil
}

func (uc *ProductUseCase) ListProducts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[product.ProductTemplate], error) {
	return uc.repo.ListTemplates(ctx, f, page)
}

// ─────────────────────────────────────────────────────────────────────────────
// Variants
// ─────────────────────────────────────────────────────────────────────────────

func (uc *ProductUseCase) CreateProductVariant(ctx context.Context, templateID int64, in CreateVariantInput) (*product.ProductVariant, error) {
	if templateID <= 0 {
		return nil, platformerrors.BadRequest("invalid template id")
	}

	// Verify template exists
	if _, err := uc.repo.GetTemplateByID(ctx, templateID); err != nil {
		return nil, err
	}

	pv := &product.ProductVariant{
		TemplateID: templateID,
		SKU:        in.SKU,
		Barcode:    in.Barcode,
		ExtraPrice: in.ExtraPrice,
		Active:     true,
		Audit:      audit.NewFields(ctx),
	}

	if err := pv.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreateVariant(ctx, pv); err != nil {
		uc.logger.Error("failed to create product variant", "error", err, "template_id", templateID)
		return nil, err
	}

	uc.logger.Info("product variant created successfully", "id", pv.ID, "template_id", templateID)
	return pv, nil
}

func (uc *ProductUseCase) GetVariant(ctx context.Context, id int64) (*product.ProductVariant, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid variant id")
	}
	return uc.repo.GetVariantByID(ctx, id)
}

func (uc *ProductUseCase) GetProductVariants(ctx context.Context, templateID int64) ([]product.ProductVariant, error) {
	if templateID <= 0 {
		return nil, platformerrors.BadRequest("invalid template id")
	}
	return uc.repo.GetVariantsByTemplateID(ctx, templateID)
}

func (uc *ProductUseCase) DeleteVariant(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.BadRequest("invalid variant id")
	}
	return uc.repo.DeleteVariant(ctx, id)
}

// ─────────────────────────────────────────────────────────────────────────────
// Categories
// ─────────────────────────────────────────────────────────────────────────────

func (uc *ProductUseCase) CreateCategory(ctx context.Context, in CreateCategoryInput) (*product.ProductCategory, error) {
	completeName := in.Name
	if in.ParentID != nil && *in.ParentID > 0 {
		parent, err := uc.repo.GetCategoryByID(ctx, *in.ParentID)
		if err != nil {
			return nil, platformerrors.Validation("parent category does not exist", map[string]string{
				"parent_id": fmt.Sprintf("category %d not found", *in.ParentID),
			})
		}
		completeName = fmt.Sprintf("%s / %s", parent.CompleteName, in.Name)
	}

	cat := &product.ProductCategory{
		Name:         in.Name,
		ParentID:     in.ParentID,
		CompleteName: completeName,
		Active:       true,
	}

	if err := cat.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreateCategory(ctx, cat); err != nil {
		uc.logger.Error("failed to create product category", "error", err, "name", cat.Name)
		return nil, err
	}

	uc.logger.Info("product category created successfully", "id", cat.ID, "complete_name", cat.CompleteName)
	return cat, nil
}

func (uc *ProductUseCase) GetCategory(ctx context.Context, id int64) (*product.ProductCategory, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid category id")
	}
	return uc.repo.GetCategoryByID(ctx, id)
}

func (uc *ProductUseCase) UpdateCategory(ctx context.Context, id int64, in UpdateCategoryInput) (*product.ProductCategory, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid category id")
	}

	cat, err := uc.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		cat.Name = *in.Name
	}
	if in.ParentID != nil {
		if *in.ParentID == id {
			return nil, platformerrors.Validation("circular reference", map[string]string{
				"parent_id": "category cannot be its own parent",
			})
		}
		if *in.ParentID > 0 {
			parent, err := uc.repo.GetCategoryByID(ctx, *in.ParentID)
			if err != nil {
				return nil, platformerrors.Validation("parent category does not exist", map[string]string{
					"parent_id": fmt.Sprintf("category %d not found", *in.ParentID),
				})
			}
			cat.CompleteName = fmt.Sprintf("%s / %s", parent.CompleteName, cat.Name)
		} else {
			cat.CompleteName = cat.Name
		}
		cat.ParentID = in.ParentID
	} else if in.Name != nil {
		// Update complete name if only name changed and has parent
		if cat.ParentID != nil && *cat.ParentID > 0 {
			if parent, err := uc.repo.GetCategoryByID(ctx, *cat.ParentID); err == nil {
				cat.CompleteName = fmt.Sprintf("%s / %s", parent.CompleteName, cat.Name)
			}
		} else {
			cat.CompleteName = cat.Name
		}
	}

	if err := cat.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateCategory(ctx, cat); err != nil {
		return nil, err
	}

	uc.logger.Info("product category updated successfully", "id", cat.ID)
	return cat, nil
}

func (uc *ProductUseCase) DeleteCategory(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.BadRequest("invalid category id")
	}
	return uc.repo.DeleteCategory(ctx, id)
}

func (uc *ProductUseCase) ListCategories(ctx context.Context) ([]product.ProductCategory, error) {
	return uc.repo.ListCategories(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// Units of Measure (UoM)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *ProductUseCase) CreateUoM(ctx context.Context, in CreateUoMInput) (*product.UnitOfMeasure, error) {
	uom := &product.UnitOfMeasure{
		Name:     in.Name,
		Category: in.Category,
		Ratio:    in.Ratio,
		Rounding: in.Rounding,
		Active:   true,
	}

	if err := uom.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreateUoM(ctx, uom); err != nil {
		uc.logger.Error("failed to create unit of measure", "error", err, "name", uom.Name)
		return nil, err
	}

	uc.logger.Info("unit of measure created successfully", "id", uom.ID, "name", uom.Name)
	return uom, nil
}

func (uc *ProductUseCase) GetUoM(ctx context.Context, id int64) (*product.UnitOfMeasure, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid unit of measure id")
	}
	return uc.repo.GetUoMByID(ctx, id)
}

func (uc *ProductUseCase) UpdateUoM(ctx context.Context, id int64, in UpdateUoMInput) (*product.UnitOfMeasure, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid unit of measure id")
	}

	uom, err := uc.repo.GetUoMByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		uom.Name = *in.Name
	}
	if in.Category != nil {
		uom.Category = *in.Category
	}
	if in.Ratio != nil {
		uom.Ratio = *in.Ratio
	}
	if in.Rounding != nil {
		uom.Rounding = *in.Rounding
	}

	if err := uom.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateUoM(ctx, uom); err != nil {
		return nil, err
	}

	uc.logger.Info("unit of measure updated successfully", "id", uom.ID)
	return uom, nil
}

func (uc *ProductUseCase) DeleteUoM(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.BadRequest("invalid unit of measure id")
	}
	return uc.repo.DeleteUoM(ctx, id)
}

func (uc *ProductUseCase) ListUoMs(ctx context.Context) ([]product.UnitOfMeasure, error) {
	return uc.repo.ListUoMs(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// Pricelists
// ─────────────────────────────────────────────────────────────────────────────

func (uc *ProductUseCase) CreatePricelist(ctx context.Context, in CreatePricelistInput) (*product.Pricelist, error) {
	pl := &product.Pricelist{
		Name:     in.Name,
		Currency: in.Currency,
		Active:   true,
		Audit:    audit.NewFields(ctx),
	}

	if err := pl.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreatePricelist(ctx, pl); err != nil {
		uc.logger.Error("failed to create pricelist", "error", err, "name", pl.Name)
		return nil, err
	}

	uc.logger.Info("pricelist created successfully", "id", pl.ID, "name", pl.Name)
	return pl, nil
}

func (uc *ProductUseCase) GetPricelist(ctx context.Context, id int64) (*product.Pricelist, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid pricelist id")
	}
	return uc.repo.GetPricelistByID(ctx, id)
}

func (uc *ProductUseCase) UpdatePricelist(ctx context.Context, id int64, in UpdatePricelistInput) (*product.Pricelist, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid pricelist id")
	}

	pl, err := uc.repo.GetPricelistByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		pl.Name = *in.Name
	}
	if in.Currency != nil {
		pl.Currency = *in.Currency
	}

	pl.Audit.Touch(ctx)

	if err := pl.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdatePricelist(ctx, pl); err != nil {
		return nil, err
	}

	uc.logger.Info("pricelist updated successfully", "id", pl.ID)
	return pl, nil
}

func (uc *ProductUseCase) DeletePricelist(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.BadRequest("invalid pricelist id")
	}
	return uc.repo.DeletePricelist(ctx, id)
}

func (uc *ProductUseCase) ListPricelists(ctx context.Context) ([]product.Pricelist, error) {
	return uc.repo.ListPricelists(ctx)
}

func (uc *ProductUseCase) AddPricelistItem(ctx context.Context, pricelistID int64, in CreatePricelistItemInput) (*product.PricelistItem, error) {
	if pricelistID <= 0 {
		return nil, platformerrors.BadRequest("invalid pricelist id")
	}

	// Verify pricelist exists
	if _, err := uc.repo.GetPricelistByID(ctx, pricelistID); err != nil {
		return nil, err
	}

	item := &product.PricelistItem{
		PricelistID:  pricelistID,
		AppliedOn:    in.AppliedOn,
		CategoryID:   in.CategoryID,
		TemplateID:   in.TemplateID,
		VariantID:    in.VariantID,
		MinQuantity:  in.MinQuantity,
		ComputePrice: in.ComputePrice,
		FixedPrice:   in.FixedPrice,
		PercentPrice: in.PercentPrice,
		DateStart:    in.DateStart,
		DateEnd:      in.DateEnd,
	}

	if err := item.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.AddPricelistItem(ctx, item); err != nil {
		uc.logger.Error("failed to add pricelist item", "error", err, "pricelist_id", pricelistID)
		return nil, err
	}

	uc.logger.Info("pricelist item added successfully", "id", item.ID, "pricelist_id", pricelistID)
	return item, nil
}

func (uc *ProductUseCase) DeletePricelistItem(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.BadRequest("invalid pricelist item id")
	}
	return uc.repo.DeletePricelistItem(ctx, id)
}

func (uc *ProductUseCase) ComputePrice(ctx context.Context, pricelistID int64, productID int64, variantID *int64, quantity float64) (float64, error) {
	if pricelistID <= 0 || productID <= 0 {
		return 0, platformerrors.BadRequest("invalid pricelist or product id")
	}
	if quantity <= 0 {
		quantity = 1.0
	}

	tmpl, err := uc.repo.GetTemplateByID(ctx, productID)
	if err != nil {
		return 0, err
	}

	basePrice := tmpl.SalePrice
	if variantID != nil && *variantID > 0 {
		variant, err := uc.repo.GetVariantByID(ctx, *variantID)
		if err == nil && variant.TemplateID == productID {
			basePrice += variant.ExtraPrice
		}
	}

	items, err := uc.repo.GetPricelistItems(ctx, pricelistID)
	if err != nil {
		return basePrice, nil // Fallback to base price
	}

	// Rule evaluation precedence (Odoo order: Variant -> Template -> Category -> All)
	var matchedItem *product.PricelistItem

	// 1. Check variant match
	if variantID != nil && *variantID > 0 {
		for i := range items {
			it := &items[i]
			if it.AppliedOn == product.PricelistAppliedOnVariant && it.VariantID != nil && *it.VariantID == *variantID {
				matchedItem = it
				break
			}
		}
	}

	// 2. Check template match
	if matchedItem == nil {
		for i := range items {
			it := &items[i]
			if it.AppliedOn == product.PricelistAppliedOnTemplate && it.TemplateID != nil && *it.TemplateID == productID {
				matchedItem = it
				break
			}
		}
	}

	// 3. Check category match
	if matchedItem == nil && tmpl.CategoryID != nil {
		for i := range items {
			it := &items[i]
			if it.AppliedOn == product.PricelistAppliedOnCategory && it.CategoryID != nil && *it.CategoryID == *tmpl.CategoryID {
				matchedItem = it
				break
			}
		}
	}

	// 4. Check global match (all products)
	if matchedItem == nil {
		for i := range items {
			it := &items[i]
			if it.AppliedOn == product.PricelistAppliedOnAll {
				matchedItem = it
				break
			}
		}
	}

	if matchedItem == nil {
		return basePrice, nil
	}

	return matchedItem.CalculatePrice(basePrice, quantity), nil
}
