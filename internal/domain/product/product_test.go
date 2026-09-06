package product_test

import (
	"testing"
	"time"

	"cashflow_backend/internal/domain/product"
)

func TestUnitOfMeasure_Validate(t *testing.T) {
	tests := []struct {
		name    string
		uom     product.UnitOfMeasure
		wantErr bool
	}{
		{
			name: "valid uom",
			uom: product.UnitOfMeasure{
				Name:     "Kilogram",
				Category: "weight",
				Ratio:    1.0,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			uom: product.UnitOfMeasure{
				Name:     "",
				Category: "weight",
				Ratio:    1.0,
			},
			wantErr: true,
		},
		{
			name: "empty category",
			uom: product.UnitOfMeasure{
				Name:     "Kilogram",
				Category: "",
				Ratio:    1.0,
			},
			wantErr: true,
		},
		{
			name: "zero or negative ratio",
			uom: product.UnitOfMeasure{
				Name:     "Kilogram",
				Category: "weight",
				Ratio:    0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.uom.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got=%v", tt.wantErr, err)
			}
		})
	}
}

func TestProductCategory_Validate(t *testing.T) {
	parentID := int64(1)
	selfID := int64(5)

	tests := []struct {
		name    string
		cat     product.ProductCategory
		wantErr bool
	}{
		{
			name: "valid root category",
			cat: product.ProductCategory{
				Name: "All",
			},
			wantErr: false,
		},
		{
			name: "valid sub-category",
			cat: product.ProductCategory{
				Name:     "Electronics",
				ParentID: &parentID,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			cat: product.ProductCategory{
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "circular reference to self",
			cat: product.ProductCategory{
				ID:       selfID,
				Name:     "Computers",
				ParentID: &selfID,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cat.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got=%v", tt.wantErr, err)
			}
		})
	}
}

func TestProductTemplate_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    product.ProductTemplate
		wantErr bool
	}{
		{
			name: "valid product goods",
			tmpl: product.ProductTemplate{
				Name:      "Mechanical Keyboard",
				Type:      product.ProductTypeGoods,
				SalePrice: 120.0,
				CostPrice: 60.0,
			},
			wantErr: false,
		},
		{
			name: "valid product service with default type",
			tmpl: product.ProductTemplate{
				Name:      "Consulting Hour",
				SalePrice: 150.0,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			tmpl: product.ProductTemplate{
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "negative sale price",
			tmpl: product.ProductTemplate{
				Name:      "Mouse",
				SalePrice: -10.0,
			},
			wantErr: true,
		},
		{
			name: "negative cost price",
			tmpl: product.ProductTemplate{
				Name:      "Mouse",
				CostPrice: -5.0,
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			tmpl: product.ProductTemplate{
				Name: "Mouse",
				Type: "invalid_type",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tmpl.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got=%v", tt.wantErr, err)
			}
		})
	}
}

func TestProductVariant_Validate(t *testing.T) {
	tests := []struct {
		name    string
		variant product.ProductVariant
		wantErr bool
	}{
		{
			name: "valid variant",
			variant: product.ProductVariant{
				TemplateID: 1,
				SKU:        "KEY-RED",
				ExtraPrice: 15.0,
			},
			wantErr: false,
		},
		{
			name: "missing template ID",
			variant: product.ProductVariant{
				TemplateID: 0,
				SKU:        "KEY-RED",
			},
			wantErr: true,
		},
		{
			name: "negative extra price",
			variant: product.ProductVariant{
				TemplateID: 1,
				ExtraPrice: -5.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.variant.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v, got=%v", tt.wantErr, err)
			}
		})
	}
}

func TestPricelistItem_CalculatePrice(t *testing.T) {
	basePrice := 100.0

	// 1. Fixed price rule
	fixedItem := product.PricelistItem{
		PricelistID:  1,
		ComputePrice: product.PricelistComputeFixed,
		FixedPrice:   80.0,
		MinQuantity:  1.0,
	}
	if got := fixedItem.CalculatePrice(basePrice, 1.0); got != 80.0 {
		t.Errorf("expected fixed price 80.0, got %f", got)
	}

	// 2. Percentage discount rule (15% discount on 100 = 85)
	percentItem := product.PricelistItem{
		PricelistID:  1,
		ComputePrice: product.PricelistComputePercentage,
		PercentPrice: 15.0,
		MinQuantity:  5.0,
	}
	// Below min quantity -> unchanged
	if got := percentItem.CalculatePrice(basePrice, 2.0); got != basePrice {
		t.Errorf("expected unchanged price 100.0 when qty < min_quantity, got %f", got)
	}
	// Equal or above min quantity -> 85.0
	if got := percentItem.CalculatePrice(basePrice, 5.0); got != 85.0 {
		t.Errorf("expected discounted price 85.0, got %f", got)
	}

	// 3. Formula rule (10% discount on 100 + 5 surcharge = 90 + 5 = 95)
	formulaItem := product.PricelistItem{
		PricelistID:  1,
		ComputePrice: product.PricelistComputeFormula,
		PercentPrice: 10.0,
		FixedPrice:   5.0,
		MinQuantity:  1.0,
	}
	if got := formulaItem.CalculatePrice(basePrice, 1.0); got != 95.0 {
		t.Errorf("expected formula price 95.0, got %f", got)
	}

	// 4. Date validity
	past := time.Now().Add(-2 * time.Hour)
	future := time.Now().Add(2 * time.Hour)
	activeItem := product.PricelistItem{
		PricelistID:  1,
		ComputePrice: product.PricelistComputeFixed,
		FixedPrice:   70.0,
		MinQuantity:  1.0,
		DateStart:    &past,
		DateEnd:      &future,
	}
	if got := activeItem.CalculatePrice(basePrice, 1.0); got != 70.0 {
		t.Errorf("expected active promo price 70.0, got %f", got)
	}

	expiredItem := product.PricelistItem{
		PricelistID:  1,
		ComputePrice: product.PricelistComputeFixed,
		FixedPrice:   70.0,
		MinQuantity:  1.0,
		DateEnd:      &past,
	}
	if got := expiredItem.CalculatePrice(basePrice, 1.0); got != basePrice {
		t.Errorf("expected base price for expired promo, got %f", got)
	}
}
