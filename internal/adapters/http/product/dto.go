package producthttp

import (
	"time"

	"cashflow_backend/internal/domain/product"
	productusecase "cashflow_backend/internal/usecase/product"
)

// ─────────────────────────────────────────────────────────────────────────────
// Product Templates DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateProductRequest struct {
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

func (r CreateProductRequest) ToInput() productusecase.CreateProductInput {
	return productusecase.CreateProductInput{
		Name:        r.Name,
		Type:        r.Type,
		CategoryID:  r.CategoryID,
		InternalRef: r.InternalRef,
		Barcode:     r.Barcode,
		SalePrice:   r.SalePrice,
		CostPrice:   r.CostPrice,
		UoMID:       r.UoMID,
		SaleOK:      r.SaleOK,
		PurchaseOK:  r.PurchaseOK,
		Weight:      r.Weight,
		Volume:      r.Volume,
		Description: r.Description,
		CompanyID:   r.CompanyID,
	}
}

type UpdateProductRequest struct {
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

func (r UpdateProductRequest) ToInput() productusecase.UpdateProductInput {
	return productusecase.UpdateProductInput{
		Name:        r.Name,
		Type:        r.Type,
		CategoryID:  r.CategoryID,
		InternalRef: r.InternalRef,
		Barcode:     r.Barcode,
		SalePrice:   r.SalePrice,
		CostPrice:   r.CostPrice,
		UoMID:       r.UoMID,
		SaleOK:      r.SaleOK,
		PurchaseOK:  r.PurchaseOK,
		Weight:      r.Weight,
		Volume:      r.Volume,
		Description: r.Description,
		CompanyID:   r.CompanyID,
	}
}

type ProductResponse struct {
	ID          int64                `json:"id"`
	Name        string               `json:"name"`
	Type        product.ProductType  `json:"type"`
	CategoryID  *int64               `json:"category_id,omitempty"`
	Category    *CategoryResponse    `json:"category,omitempty"`
	InternalRef string               `json:"internal_ref,omitempty"`
	Barcode     string               `json:"barcode,omitempty"`
	SalePrice   float64              `json:"sale_price"`
	CostPrice   float64              `json:"cost_price"`
	UoMID       *int64               `json:"uom_id,omitempty"`
	UoM         *UoMResponse         `json:"uom,omitempty"`
	SaleOK      bool                 `json:"sale_ok"`
	PurchaseOK  bool                 `json:"purchase_ok"`
	Weight      float64              `json:"weight"`
	Volume      float64              `json:"volume"`
	Description string               `json:"description,omitempty"`
	CompanyID   *int64               `json:"company_id,omitempty"`
	Active      bool                 `json:"active"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

func ToProductResponse(pt *product.ProductTemplate) ProductResponse {
	if pt == nil {
		return ProductResponse{}
	}

	var catResp *CategoryResponse
	if pt.Category != nil {
		c := ToCategoryResponse(pt.Category)
		catResp = &c
	}

	var uomResp *UoMResponse
	if pt.UoM != nil {
		u := ToUoMResponse(pt.UoM)
		uomResp = &u
	}

	return ProductResponse{
		ID:          pt.ID,
		Name:        pt.Name,
		Type:        pt.Type,
		CategoryID:  pt.CategoryID,
		Category:    catResp,
		InternalRef: pt.InternalRef,
		Barcode:     pt.Barcode,
		SalePrice:   pt.SalePrice,
		CostPrice:   pt.CostPrice,
		UoMID:       pt.UoMID,
		UoM:         uomResp,
		SaleOK:      pt.SaleOK,
		PurchaseOK:  pt.PurchaseOK,
		Weight:      pt.Weight,
		Volume:      pt.Volume,
		Description: pt.Description,
		CompanyID:   pt.CompanyID,
		Active:      pt.Active,
		CreatedAt:   pt.Audit.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   pt.Audit.UpdatedAt.Format(time.RFC3339),
	}
}

func ToProductResponseList(items []product.ProductTemplate) []ProductResponse {
	result := make([]ProductResponse, len(items))
	for i, item := range items {
		result[i] = ToProductResponse(&item)
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Variant DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateVariantRequest struct {
	SKU        string  `json:"sku"`
	Barcode    string  `json:"barcode"`
	ExtraPrice float64 `json:"extra_price"`
}

func (r CreateVariantRequest) ToInput() productusecase.CreateVariantInput {
	return productusecase.CreateVariantInput{
		SKU:        r.SKU,
		Barcode:    r.Barcode,
		ExtraPrice: r.ExtraPrice,
	}
}

type VariantResponse struct {
	ID         int64   `json:"id"`
	TemplateID int64   `json:"template_id"`
	SKU        string  `json:"sku,omitempty"`
	Barcode    string  `json:"barcode,omitempty"`
	ExtraPrice float64 `json:"extra_price"`
	Active     bool    `json:"active"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

func ToVariantResponse(pv *product.ProductVariant) VariantResponse {
	if pv == nil {
		return VariantResponse{}
	}
	return VariantResponse{
		ID:         pv.ID,
		TemplateID: pv.TemplateID,
		SKU:        pv.SKU,
		Barcode:    pv.Barcode,
		ExtraPrice: pv.ExtraPrice,
		Active:     pv.Active,
		CreatedAt:  pv.Audit.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  pv.Audit.UpdatedAt.Format(time.RFC3339),
	}
}

func ToVariantResponseList(items []product.ProductVariant) []VariantResponse {
	result := make([]VariantResponse, len(items))
	for i, item := range items {
		result[i] = ToVariantResponse(&item)
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Category DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateCategoryRequest struct {
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id"`
}

func (r CreateCategoryRequest) ToInput() productusecase.CreateCategoryInput {
	return productusecase.CreateCategoryInput{
		Name:     r.Name,
		ParentID: r.ParentID,
	}
}

type UpdateCategoryRequest struct {
	Name     *string `json:"name"`
	ParentID *int64  `json:"parent_id"`
}

func (r UpdateCategoryRequest) ToInput() productusecase.UpdateCategoryInput {
	return productusecase.UpdateCategoryInput{
		Name:     r.Name,
		ParentID: r.ParentID,
	}
}

type CategoryResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	ParentID     *int64 `json:"parent_id,omitempty"`
	CompleteName string `json:"complete_name"`
	Active       bool   `json:"active"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func ToCategoryResponse(c *product.ProductCategory) CategoryResponse {
	if c == nil {
		return CategoryResponse{}
	}
	return CategoryResponse{
		ID:           c.ID,
		Name:         c.Name,
		ParentID:     c.ParentID,
		CompleteName: c.CompleteName,
		Active:       c.Active,
		CreatedAt:    c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    c.UpdatedAt.Format(time.RFC3339),
	}
}

func ToCategoryResponseList(items []product.ProductCategory) []CategoryResponse {
	result := make([]CategoryResponse, len(items))
	for i, item := range items {
		result[i] = ToCategoryResponse(&item)
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Unit of Measure (UoM) DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateUoMRequest struct {
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Ratio    float64 `json:"ratio"`
	Rounding float64 `json:"rounding"`
}

func (r CreateUoMRequest) ToInput() productusecase.CreateUoMInput {
	return productusecase.CreateUoMInput{
		Name:     r.Name,
		Category: r.Category,
		Ratio:    r.Ratio,
		Rounding: r.Rounding,
	}
}

type UpdateUoMRequest struct {
	Name     *string  `json:"name"`
	Category *string  `json:"category"`
	Ratio    *float64 `json:"ratio"`
	Rounding *float64 `json:"rounding"`
}

func (r UpdateUoMRequest) ToInput() productusecase.UpdateUoMInput {
	return productusecase.UpdateUoMInput{
		Name:     r.Name,
		Category: r.Category,
		Ratio:    r.Ratio,
		Rounding: r.Rounding,
	}
}

type UoMResponse struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Category  string  `json:"category"`
	Ratio     float64 `json:"ratio"`
	Rounding  float64 `json:"rounding"`
	Active    bool    `json:"active"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func ToUoMResponse(u *product.UnitOfMeasure) UoMResponse {
	if u == nil {
		return UoMResponse{}
	}
	return UoMResponse{
		ID:        u.ID,
		Name:      u.Name,
		Category:  u.Category,
		Ratio:     u.Ratio,
		Rounding:  u.Rounding,
		Active:    u.Active,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
}

func ToUoMResponseList(items []product.UnitOfMeasure) []UoMResponse {
	result := make([]UoMResponse, len(items))
	for i, item := range items {
		result[i] = ToUoMResponse(&item)
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Pricelist DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreatePricelistRequest struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

func (r CreatePricelistRequest) ToInput() productusecase.CreatePricelistInput {
	return productusecase.CreatePricelistInput{
		Name:     r.Name,
		Currency: r.Currency,
	}
}

type UpdatePricelistRequest struct {
	Name     *string `json:"name"`
	Currency *string `json:"currency"`
}

func (r UpdatePricelistRequest) ToInput() productusecase.UpdatePricelistInput {
	return productusecase.UpdatePricelistInput{
		Name:     r.Name,
		Currency: r.Currency,
	}
}

type CreatePricelistItemRequest struct {
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

func (r CreatePricelistItemRequest) ToInput() productusecase.CreatePricelistItemInput {
	return productusecase.CreatePricelistItemInput{
		AppliedOn:    r.AppliedOn,
		CategoryID:   r.CategoryID,
		TemplateID:   r.TemplateID,
		VariantID:    r.VariantID,
		MinQuantity:  r.MinQuantity,
		ComputePrice: r.ComputePrice,
		FixedPrice:   r.FixedPrice,
		PercentPrice: r.PercentPrice,
		DateStart:    r.DateStart,
		DateEnd:      r.DateEnd,
	}
}

type PricelistItemResponse struct {
	ID           int64                        `json:"id"`
	PricelistID  int64                        `json:"pricelist_id"`
	AppliedOn    product.PricelistAppliedOn   `json:"applied_on"`
	CategoryID   *int64                       `json:"category_id,omitempty"`
	TemplateID   *int64                       `json:"template_id,omitempty"`
	VariantID    *int64                       `json:"variant_id,omitempty"`
	MinQuantity  float64                      `json:"min_quantity"`
	ComputePrice product.PricelistComputeType `json:"compute_price"`
	FixedPrice   float64                      `json:"fixed_price"`
	PercentPrice float64                      `json:"percent_price"`
	DateStart    *string                      `json:"date_start,omitempty"`
	DateEnd      *string                      `json:"date_end,omitempty"`
	CreatedAt    string                       `json:"created_at"`
	UpdatedAt    string                       `json:"updated_at"`
}

func ToPricelistItemResponse(item *product.PricelistItem) PricelistItemResponse {
	if item == nil {
		return PricelistItemResponse{}
	}
	var startStr, endStr *string
	if item.DateStart != nil {
		s := item.DateStart.Format(time.RFC3339)
		startStr = &s
	}
	if item.DateEnd != nil {
		e := item.DateEnd.Format(time.RFC3339)
		endStr = &e
	}
	return PricelistItemResponse{
		ID:           item.ID,
		PricelistID:  item.PricelistID,
		AppliedOn:    item.AppliedOn,
		CategoryID:   item.CategoryID,
		TemplateID:   item.TemplateID,
		VariantID:    item.VariantID,
		MinQuantity:  item.MinQuantity,
		ComputePrice: item.ComputePrice,
		FixedPrice:   item.FixedPrice,
		PercentPrice: item.PercentPrice,
		DateStart:    startStr,
		DateEnd:      endStr,
		CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    item.UpdatedAt.Format(time.RFC3339),
	}
}

func ToPricelistItemResponseList(items []product.PricelistItem) []PricelistItemResponse {
	result := make([]PricelistItemResponse, len(items))
	for i, item := range items {
		result[i] = ToPricelistItemResponse(&item)
	}
	return result
}

type PricelistResponse struct {
	ID        int64                   `json:"id"`
	Name      string                  `json:"name"`
	Currency  string                  `json:"currency"`
	Active    bool                    `json:"active"`
	Items     []PricelistItemResponse `json:"items,omitempty"`
	CreatedAt string                  `json:"created_at"`
	UpdatedAt string                  `json:"updated_at"`
}

func ToPricelistResponse(pl *product.Pricelist) PricelistResponse {
	if pl == nil {
		return PricelistResponse{}
	}
	var itemsResp []PricelistItemResponse
	if len(pl.Items) > 0 {
		itemsResp = ToPricelistItemResponseList(pl.Items)
	}
	return PricelistResponse{
		ID:        pl.ID,
		Name:      pl.Name,
		Currency:  pl.Currency,
		Active:    pl.Active,
		Items:     itemsResp,
		CreatedAt: pl.Audit.CreatedAt.Format(time.RFC3339),
		UpdatedAt: pl.Audit.UpdatedAt.Format(time.RFC3339),
	}
}

func ToPricelistResponseList(items []product.Pricelist) []PricelistResponse {
	result := make([]PricelistResponse, len(items))
	for i, item := range items {
		result[i] = ToPricelistResponse(&item)
	}
	return result
}

type ComputePriceRequest struct {
	ProductID int64   `json:"product_id"`
	VariantID *int64  `json:"variant_id"`
	Quantity  float64 `json:"quantity"`
}

type ComputePriceResponse struct {
	PricelistID int64   `json:"pricelist_id"`
	ProductID   int64   `json:"product_id"`
	VariantID   *int64  `json:"variant_id,omitempty"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
}
