package product

import (
	"fmt"
	"strings"
	"time"

	"cashflow_backend/internal/domain/stock"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// ProductType represents the classification of a product.
type ProductType string

const (
	ProductTypeGoods   ProductType = "consu"   // Physical goods / consumable materials
	ProductTypeService ProductType = "service" // Non-material service provided
	ProductTypeCombo   ProductType = "combo"   // Bundle / combo of multiple products
)

// UnitOfMeasure represents a measurement unit (uom.uom in Odoo).
type UnitOfMeasure struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"` // e.g. "unit", "weight", "volume", "length", "time"
	Ratio     float64   `json:"ratio"`    // Ratio relative to base reference unit
	Rounding  float64   `json:"rounding"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate verifies UnitOfMeasure constraints.
func (u *UnitOfMeasure) Validate() error {
	u.Name = strings.TrimSpace(u.Name)
	if u.Name == "" {
		return platformerrors.Validation("unit of measure name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	u.Category = strings.TrimSpace(strings.ToLower(u.Category))
	if u.Category == "" {
		return platformerrors.Validation("unit of measure category is required", map[string]string{
			"category": "cannot be empty",
		})
	}
	if u.Ratio <= 0 {
		return platformerrors.Validation("unit of measure ratio must be strictly positive", map[string]string{
			"ratio": "must be greater than 0",
		})
	}
	if u.Rounding <= 0 {
		u.Rounding = 0.001
	}
	return nil
}

// ProductCategory represents a hierarchical categorization (product.category in Odoo).
type ProductCategory struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	ParentID     *int64 `json:"parent_id,omitempty"`
	CompleteName string `json:"complete_name"` // e.g. "All / Electronics / Laptops"
	// Valuation defaults (company_templates — inherited by products, Phase 12).
	PropertyCostMethod               *stock.CostMethod    `json:"property_cost_method,omitempty"`
	PropertyValuation                *stock.ValuationMode `json:"property_valuation,omitempty"`
	PropertyLotValuated              *bool                `json:"property_lot_valuated,omitempty"`
	PropertyStockValuationAccountID  *int64               `json:"property_stock_valuation_account_id,omitempty"`
	PropertyPriceDifferenceAccountID *int64               `json:"property_price_difference_account_id,omitempty"`
	PropertyStockJournalID           *int64               `json:"property_stock_journal_id,omitempty"`
	Active                           bool                 `json:"active"`
	CreatedAt                        time.Time            `json:"created_at"`
	UpdatedAt                        time.Time            `json:"updated_at"`
}

// Validate verifies ProductCategory invariants.
func (c *ProductCategory) Validate() error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return platformerrors.Validation("category name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if c.ParentID != nil && c.ID > 0 && *c.ParentID == c.ID {
		return platformerrors.Validation("circular reference in category hierarchy", map[string]string{
			"parent_id": "category cannot be its own parent",
		})
	}
	if c.CompleteName == "" {
		c.CompleteName = c.Name
	}
	return nil
}

// ProductTemplate represents the master definition of a product (product.template in Odoo).
type ProductTemplate struct {
	ID          int64            `json:"id"`
	Name        string           `json:"name"`
	Type        ProductType      `json:"type"`
	CategoryID  *int64           `json:"category_id,omitempty"`
	Category    *ProductCategory `json:"category,omitempty"`
	InternalRef string           `json:"internal_ref,omitempty"` // SKU / default_code
	Barcode     string           `json:"barcode,omitempty"`
	SalePrice   float64          `json:"sale_price"`
	CostPrice   float64          `json:"cost_price"`
	// Valuation configuration (Phase 12 — resolved from category/company when empty).
	CostMethod               stock.CostMethod    `json:"cost_method,omitempty"` // standard | fifo | average
	Valuation                stock.ValuationMode `json:"valuation,omitempty"`   // real_time | periodic
	LotValuated              bool                `json:"lot_valuated,omitempty"`
	AvgCost                  float64             `json:"avg_cost,omitempty"`
	TotalValue               float64             `json:"total_value,omitempty"`
	StockValuationAccountID  *int64              `json:"stock_valuation_account_id,omitempty"`
	PriceDifferenceAccountID *int64              `json:"price_difference_account_id,omitempty"`
	StockJournalID           *int64              `json:"stock_journal_id,omitempty"`
	// Landed cost capability (Odoo stock_landed_costs: landed_cost_ok / split_method_landed_cost).
	LandedCostOK          bool              `json:"landed_cost_ok,omitempty"`
	SplitMethodLandedCost stock.SplitMethod `json:"split_method_landed_cost,omitempty"`
	UoMID                 *int64            `json:"uom_id,omitempty"`
	UoM                   *UnitOfMeasure    `json:"uom,omitempty"`
	SaleOK                bool              `json:"sale_ok"`
	PurchaseOK            bool              `json:"purchase_ok"`
	Weight                float64           `json:"weight"`
	Volume                float64           `json:"volume"`
	Description           string            `json:"description,omitempty"`
	CompanyID             *int64            `json:"company_id,omitempty"`
	Active                bool              `json:"active"`
	Audit                 audit.Fields      `json:"audit"`
}

// Validate ensures ProductTemplate invariants are strictly upheld.
func (pt *ProductTemplate) Validate() error {
	pt.Name = strings.TrimSpace(pt.Name)
	if pt.Name == "" {
		return platformerrors.Validation("product name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if len(pt.Name) > 255 {
		return platformerrors.Validation("product name exceeds maximum length", map[string]string{
			"name": "must not exceed 255 characters",
		})
	}

	if pt.Type == "" {
		pt.Type = ProductTypeGoods
	}
	if pt.Type != ProductTypeGoods && pt.Type != ProductTypeService && pt.Type != ProductTypeCombo {
		return platformerrors.Validation("invalid product type", map[string]string{
			"type": fmt.Sprintf("must be '%s', '%s', or '%s'", ProductTypeGoods, ProductTypeService, ProductTypeCombo),
		})
	}

	if pt.SalePrice < 0 {
		return platformerrors.Validation("sale price cannot be negative", map[string]string{
			"sale_price": "must be greater than or equal to 0",
		})
	}

	if pt.CostPrice < 0 {
		return platformerrors.Validation("cost price cannot be negative", map[string]string{
			"cost_price": "must be greater than or equal to 0",
		})
	}

	if pt.Weight < 0 {
		return platformerrors.Validation("weight cannot be negative", map[string]string{
			"weight": "must be greater than or equal to 0",
		})
	}

	if pt.Volume < 0 {
		return platformerrors.Validation("volume cannot be negative", map[string]string{
			"volume": "must be greater than or equal to 0",
		})
	}

	pt.InternalRef = strings.TrimSpace(pt.InternalRef)
	pt.Barcode = strings.TrimSpace(pt.Barcode)

	if pt.SplitMethodLandedCost == "" {
		pt.SplitMethodLandedCost = stock.SplitEqual
	}

	return nil
}

// ProductAttribute represents an attribute name like "Color" or "Size" (product.attribute).
type ProductAttribute struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Sequence  int       `json:"sequence"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProductAttributeValue represents an attribute value like "Red" or "XL" (product.attribute.value).
type ProductAttributeValue struct {
	ID          int64     `json:"id"`
	AttributeID int64     `json:"attribute_id"`
	Name        string    `json:"name"`
	Sequence    int       `json:"sequence"`
	ExtraPrice  float64   `json:"extra_price"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// VariantAttributeValue represents the value of an attribute assigned to a variant.
type VariantAttributeValue struct {
	AttributeID   int64   `json:"attribute_id"`
	AttributeName string  `json:"attribute_name"`
	ValueID       int64   `json:"value_id"`
	ValueName     string  `json:"value_name"`
	ExtraPrice    float64 `json:"extra_price"`
}

// ProductVariant represents a concrete SKU variant of a product template (product.product in Odoo).
type ProductVariant struct {
	ID         int64                   `json:"id"`
	TemplateID int64                   `json:"template_id"`
	SKU        string                  `json:"sku,omitempty"`
	Barcode    string                  `json:"barcode,omitempty"`
	ExtraPrice float64                 `json:"extra_price"`
	Attributes []VariantAttributeValue `json:"attributes,omitempty"`
	Active     bool                    `json:"active"`
	Audit      audit.Fields            `json:"audit"`
}

// Validate ensures ProductVariant invariants.
func (pv *ProductVariant) Validate() error {
	if pv.TemplateID <= 0 {
		return platformerrors.Validation("template ID is required for variant", map[string]string{
			"template_id": "must be a valid product template ID",
		})
	}
	if pv.ExtraPrice < 0 {
		return platformerrors.Validation("extra price cannot be negative", map[string]string{
			"extra_price": "must be greater than or equal to 0",
		})
	}
	pv.SKU = strings.TrimSpace(pv.SKU)
	pv.Barcode = strings.TrimSpace(pv.Barcode)
	return nil
}

// PricelistAppliedOn defines what target a pricelist rule applies to.
type PricelistAppliedOn string

const (
	PricelistAppliedOnAll      PricelistAppliedOn = "all"
	PricelistAppliedOnCategory PricelistAppliedOn = "category"
	PricelistAppliedOnTemplate PricelistAppliedOn = "template"
	PricelistAppliedOnVariant  PricelistAppliedOn = "variant"
)

// PricelistComputeType defines how prices are calculated.
type PricelistComputeType string

const (
	PricelistComputeFixed      PricelistComputeType = "fixed"      // Fixed price override
	PricelistComputePercentage PricelistComputeType = "percentage" // Discount percentage
	PricelistComputeFormula    PricelistComputeType = "formula"    // Cost plus margin
)

// PricelistItem represents a pricing rule (product.pricelist.item in Odoo).
type PricelistItem struct {
	ID           int64                `json:"id"`
	PricelistID  int64                `json:"pricelist_id"`
	AppliedOn    PricelistAppliedOn   `json:"applied_on"`
	CategoryID   *int64               `json:"category_id,omitempty"`
	TemplateID   *int64               `json:"template_id,omitempty"`
	VariantID    *int64               `json:"variant_id,omitempty"`
	MinQuantity  float64              `json:"min_quantity"`
	ComputePrice PricelistComputeType `json:"compute_price"`
	FixedPrice   float64              `json:"fixed_price"`
	PercentPrice float64              `json:"percent_price"` // e.g. 10.0 for 10% discount
	DateStart    *time.Time           `json:"date_start,omitempty"`
	DateEnd      *time.Time           `json:"date_end,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// Validate checks PricelistItem constraints.
func (pi *PricelistItem) Validate() error {
	if pi.PricelistID <= 0 {
		return platformerrors.Validation("pricelist id is required", map[string]string{
			"pricelist_id": "must be a valid ID",
		})
	}
	if pi.AppliedOn == "" {
		pi.AppliedOn = PricelistAppliedOnAll
	}
	switch pi.AppliedOn {
	case PricelistAppliedOnAll:
	case PricelistAppliedOnCategory:
		if pi.CategoryID == nil || *pi.CategoryID <= 0 {
			return platformerrors.Validation("category_id is required when applied_on is category", map[string]string{
				"category_id": "must not be empty",
			})
		}
	case PricelistAppliedOnTemplate:
		if pi.TemplateID == nil || *pi.TemplateID <= 0 {
			return platformerrors.Validation("template_id is required when applied_on is template", map[string]string{
				"template_id": "must not be empty",
			})
		}
	case PricelistAppliedOnVariant:
		if pi.VariantID == nil || *pi.VariantID <= 0 {
			return platformerrors.Validation("variant_id is required when applied_on is variant", map[string]string{
				"variant_id": "must not be empty",
			})
		}
	default:
		return platformerrors.Validation("invalid applied_on value", map[string]string{
			"applied_on": "must be all, category, template, or variant",
		})
	}

	if pi.ComputePrice == "" {
		pi.ComputePrice = PricelistComputeFixed
	}
	if pi.ComputePrice != PricelistComputeFixed && pi.ComputePrice != PricelistComputePercentage && pi.ComputePrice != PricelistComputeFormula {
		return platformerrors.Validation("invalid compute_price value", map[string]string{
			"compute_price": "must be fixed, percentage, or formula",
		})
	}

	if pi.DateStart != nil && pi.DateEnd != nil && pi.DateEnd.Before(*pi.DateStart) {
		return platformerrors.Validation("invalid date range", map[string]string{
			"date_end": "cannot be before date_start",
		})
	}

	if pi.MinQuantity < 0 {
		pi.MinQuantity = 1.0
	}

	return nil
}

// CalculatePrice applies the rule to compute the final price given a base price and quantity.
func (pi *PricelistItem) CalculatePrice(basePrice float64, qty float64) float64 {
	if qty < pi.MinQuantity {
		return basePrice
	}

	now := time.Now()
	if pi.DateStart != nil && now.Before(*pi.DateStart) {
		return basePrice
	}
	if pi.DateEnd != nil && now.After(*pi.DateEnd) {
		return basePrice
	}

	switch pi.ComputePrice {
	case PricelistComputeFixed:
		return pi.FixedPrice
	case PricelistComputePercentage:
		discount := basePrice * (pi.PercentPrice / 100.0)
		finalPrice := basePrice - discount
		if finalPrice < 0 {
			return 0
		}
		return finalPrice
	case PricelistComputeFormula:
		discount := basePrice * (pi.PercentPrice / 100.0)
		return basePrice - discount + pi.FixedPrice
	default:
		return basePrice
	}
}

// Pricelist represents a collection of pricing rules (product.pricelist in Odoo).
type Pricelist struct {
	ID       int64           `json:"id"`
	Name     string          `json:"name"`
	Currency string          `json:"currency"`
	Active   bool            `json:"active"`
	Items    []PricelistItem `json:"items,omitempty"`
	Audit    audit.Fields    `json:"audit"`
}

// Validate checks Pricelist constraints.
func (pl *Pricelist) Validate() error {
	pl.Name = strings.TrimSpace(pl.Name)
	if pl.Name == "" {
		return platformerrors.Validation("pricelist name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	pl.Currency = strings.TrimSpace(strings.ToUpper(pl.Currency))
	if pl.Currency == "" {
		pl.Currency = "USD"
	}
	return nil
}
