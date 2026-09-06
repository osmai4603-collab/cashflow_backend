package productusecase_test

import (
	"context"
	"testing"

	productstorage "cashflow_backend/internal/adapters/storage/product"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	productusecase "cashflow_backend/internal/usecase/product"
)

func TestProductUseCase_ProductLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := productstorage.NewMemoryRepo()
	uc := productusecase.New(repo, nil)

	// 1. Create Product
	catID := int64(1)
	uomID := int64(1)
	in := productusecase.CreateProductInput{
		Name:        "Gaming Laptop",
		Type:        product.ProductTypeGoods,
		CategoryID:  &catID,
		UoMID:       &uomID,
		InternalRef: "LAP-001",
		Barcode:     "987654321",
		SalePrice:   2000.0,
		CostPrice:   1500.0,
	}

	pt, err := uc.CreateProduct(ctx, in)
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}
	if pt.ID <= 0 {
		t.Fatalf("expected positive product id, got %d", pt.ID)
	}

	// Verify default variant was created
	variants, err := uc.GetProductVariants(ctx, pt.ID)
	if err != nil {
		t.Fatalf("failed to get product variants: %v", err)
	}
	if len(variants) != 1 {
		t.Fatalf("expected 1 default variant, got %d", len(variants))
	}
	if variants[0].SKU != "LAP-001" {
		t.Errorf("expected variant SKU 'LAP-001', got %q", variants[0].SKU)
	}

	// 2. Get Product
	fetched, err := uc.GetProduct(ctx, pt.ID)
	if err != nil {
		t.Fatalf("failed to get product: %v", err)
	}
	if fetched.Name != "Gaming Laptop" {
		t.Fatalf("unexpected product name: %q", fetched.Name)
	}

	// 3. Update Product
	newName := "Gaming Laptop RTX 4080"
	newPrice := 2200.0
	upIn := productusecase.UpdateProductInput{
		Name:      &newName,
		SalePrice: &newPrice,
	}
	updated, err := uc.UpdateProduct(ctx, pt.ID, upIn)
	if err != nil {
		t.Fatalf("failed to update product: %v", err)
	}
	if updated.Name != newName || updated.SalePrice != newPrice {
		t.Fatalf("product update not reflected: %+v", updated)
	}

	// 4. List Products
	f := filter.NewFilter(
		filter.Criterion{Field: "name", Operator: filter.OpILike, Value: "gaming"},
	)
	pageRes, err := uc.ListProducts(ctx, f, pagination.PageRequest{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("failed to list products: %v", err)
	}
	if pageRes.TotalItems != 1 {
		t.Fatalf("expected 1 item, got %d", pageRes.TotalItems)
	}

	// 5. Delete Product
	if err := uc.DeleteProduct(ctx, pt.ID); err != nil {
		t.Fatalf("failed to delete product: %v", err)
	}
	if _, err := uc.GetProduct(ctx, pt.ID); err == nil {
		t.Fatalf("expected error getting deleted product, got nil")
	}
}

func TestProductUseCase_CategoriesAndUoMs(t *testing.T) {
	ctx := context.Background()
	repo := productstorage.NewMemoryRepo()
	uc := productusecase.New(repo, nil)

	// Create sub-category
	parentID := int64(1)
	cat, err := uc.CreateCategory(ctx, productusecase.CreateCategoryInput{
		Name:     "Computers",
		ParentID: &parentID,
	})
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}
	if cat.CompleteName != "All / Computers" {
		t.Errorf("expected complete name 'All / Computers', got %q", cat.CompleteName)
	}

	// List categories
	cats, err := uc.ListCategories(ctx)
	if err != nil {
		t.Fatalf("failed to list categories: %v", err)
	}
	if len(cats) < 2 {
		t.Fatalf("expected at least 2 categories, got %d", len(cats))
	}

	// Create UoM
	uom, err := uc.CreateUoM(ctx, productusecase.CreateUoMInput{
		Name:     "Box of 100",
		Category: "unit",
		Ratio:    100.0,
		Rounding: 1.0,
	})
	if err != nil {
		t.Fatalf("failed to create uom: %v", err)
	}
	if uom.ID <= 0 {
		t.Fatalf("expected positive uom id, got %d", uom.ID)
	}
}

func TestProductUseCase_PricelistsAndComputePrice(t *testing.T) {
	ctx := context.Background()
	repo := productstorage.NewMemoryRepo()
	uc := productusecase.New(repo, nil)

	// Create Product (Base Price = 100)
	pt, err := uc.CreateProduct(ctx, productusecase.CreateProductInput{
		Name:      "Standard Widget",
		Type:      product.ProductTypeGoods,
		SalePrice: 100.0,
	})
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	// Create Variant with +20 Extra Price
	v, err := uc.CreateProductVariant(ctx, pt.ID, productusecase.CreateVariantInput{
		SKU:        "WIDGET-DELUXE",
		ExtraPrice: 20.0,
	})
	if err != nil {
		t.Fatalf("failed to create variant: %v", err)
	}

	// Create Pricelist "VIP Wholesale"
	pl, err := uc.CreatePricelist(ctx, productusecase.CreatePricelistInput{
		Name:     "VIP Wholesale",
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("failed to create pricelist: %v", err)
	}

	// Add 10% discount rule for 5+ items on this template
	_, err = uc.AddPricelistItem(ctx, pl.ID, productusecase.CreatePricelistItemInput{
		AppliedOn:    product.PricelistAppliedOnTemplate,
		TemplateID:   &pt.ID,
		MinQuantity:  5.0,
		ComputePrice: product.PricelistComputePercentage,
		PercentPrice: 10.0,
	})
	if err != nil {
		t.Fatalf("failed to add pricelist item: %v", err)
	}

	// Compute Price: 1 unit -> No discount -> 100.0
	p1, err := uc.ComputePrice(ctx, pl.ID, pt.ID, nil, 1.0)
	if err != nil {
		t.Fatalf("compute price failed: %v", err)
	}
	if p1 != 100.0 {
		t.Errorf("expected 100.0 for 1 unit, got %f", p1)
	}

	// Compute Price: 5 units -> 10% discount on 100 = 90.0
	p5, err := uc.ComputePrice(ctx, pl.ID, pt.ID, nil, 5.0)
	if err != nil {
		t.Fatalf("compute price failed: %v", err)
	}
	if p5 != 90.0 {
		t.Errorf("expected 90.0 for 5 units, got %f", p5)
	}

	// Compute Price with Variant: 5 units -> 10% discount on (100 + 20 = 120) = 108.0
	pV5, err := uc.ComputePrice(ctx, pl.ID, pt.ID, &v.ID, 5.0)
	if err != nil {
		t.Fatalf("compute price failed with variant: %v", err)
	}
	if pV5 != 108.0 {
		t.Errorf("expected 108.0 for 5 units of deluxe variant, got %f", pV5)
	}
}
