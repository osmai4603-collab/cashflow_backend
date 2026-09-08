package productstorage_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	productstorage "cashflow_backend/internal/adapters/storage/product"
	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/i18n"
	"cashflow_backend/internal/platform/pagination"
)

func TestMemoryRepo_TemplateLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := productstorage.NewMemoryRepo()

	// 1. Create
	catID := int64(1)
	uomID := int64(1)
	tmpl := &product.ProductTemplate{
		Name:        "Wireless Mouse",
		Type:        product.ProductTypeGoods,
		CategoryID:  &catID,
		InternalRef: "WM-01",
		Barcode:     "1234567890",
		SalePrice:   25.0,
		CostPrice:   12.5,
		UoMID:       &uomID,
		SaleOK:      true,
		PurchaseOK:  true,
	}

	if err := repo.CreateTemplate(ctx, tmpl); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}
	if tmpl.ID <= 0 {
		t.Fatalf("expected positive template id, got %d", tmpl.ID)
	}

	// 2. GetByID
	fetched, err := repo.GetTemplateByID(ctx, tmpl.ID)
	if err != nil {
		t.Fatalf("failed to get template by id: %v", err)
	}
	if fetched.Name != tmpl.Name || fetched.SalePrice != 25.0 {
		t.Fatalf("unexpected fetched template: %+v", fetched)
	}
	if fetched.Category == nil || fetched.Category.Name != "All" {
		t.Errorf("expected enriched Category 'All', got %+v", fetched.Category)
	}
	if fetched.UoM == nil || fetched.UoM.Name != "Units" {
		t.Errorf("expected enriched UoM 'Units', got %+v", fetched.UoM)
	}

	// 3. Update
	fetched.SalePrice = 29.99
	fetched.Name = "Wireless Mouse Pro"
	if err := repo.UpdateTemplate(ctx, fetched); err != nil {
		t.Fatalf("failed to update template: %v", err)
	}

	updated, err := repo.GetTemplateByID(ctx, tmpl.ID)
	if err != nil {
		t.Fatalf("failed to get updated template: %v", err)
	}
	if updated.SalePrice != 29.99 || updated.Name != "Wireless Mouse Pro" {
		t.Fatalf("template update not reflected: %+v", updated)
	}

	// 4. List with filter
	f := filter.NewFilter(
		filter.Criterion{Field: "name", Operator: filter.OpILike, Value: "mouse"},
	)
	pageReq := pagination.PageRequest{Page: 1, Limit: 10, SortBy: "name"}
	pageRes, err := repo.ListTemplates(ctx, f, pageReq)
	if err != nil {
		t.Fatalf("failed to list templates: %v", err)
	}
	if pageRes.TotalItems != 1 || len(pageRes.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", pageRes.TotalItems)
	}

	// 5. Delete (soft delete)
	if err := repo.DeleteTemplate(ctx, tmpl.ID); err != nil {
		t.Fatalf("failed to delete template: %v", err)
	}
	_, err = repo.GetTemplateByID(ctx, tmpl.ID)
	if err == nil {
		t.Fatalf("expected error getting soft-deleted template, got nil")
	}
}

func TestMemoryRepo_Variants(t *testing.T) {
	ctx := context.Background()
	repo := productstorage.NewMemoryRepo()

	tmpl := &product.ProductTemplate{
		Name:      "T-Shirt",
		Type:      product.ProductTypeGoods,
		SalePrice: 20.0,
	}
	if err := repo.CreateTemplate(ctx, tmpl); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Create Variant 1
	v1 := &product.ProductVariant{
		TemplateID: tmpl.ID,
		SKU:        "TSHIRT-RED-M",
		ExtraPrice: 0.0,
	}
	if err := repo.CreateVariant(ctx, v1); err != nil {
		t.Fatalf("failed to create variant 1: %v", err)
	}

	// Create Variant 2
	v2 := &product.ProductVariant{
		TemplateID: tmpl.ID,
		SKU:        "TSHIRT-BLUE-L",
		ExtraPrice: 2.5,
	}
	if err := repo.CreateVariant(ctx, v2); err != nil {
		t.Fatalf("failed to create variant 2: %v", err)
	}

	variants, err := repo.GetVariantsByTemplateID(ctx, tmpl.ID)
	if err != nil {
		t.Fatalf("failed to get variants: %v", err)
	}
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}
}

func TestMemoryRepo_CategoriesAndUoMs(t *testing.T) {
	ctx := context.Background()
	repo := productstorage.NewMemoryRepo()

	// Initial UoMs check
	uoms, err := repo.ListUoMs(ctx)
	if err != nil {
		t.Fatalf("failed to list uoms: %v", err)
	}
	if len(uoms) < 6 {
		t.Fatalf("expected at least 6 standard uoms, got %d", len(uoms))
	}

	// Create sub-category
	parentID := int64(1)
	cat := &product.ProductCategory{
		Name:         "Hardware",
		ParentID:     &parentID,
		CompleteName: "All / Hardware",
	}
	if err := repo.CreateCategory(ctx, cat); err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	cats, err := repo.ListCategories(ctx)
	if err != nil {
		t.Fatalf("failed to list categories: %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories (All + Hardware), got %d", len(cats))
	}

	// Attach product to category and verify delete prevention
	tmpl := &product.ProductTemplate{
		Name:       "Hammer",
		CategoryID: &cat.ID,
	}
	if err := repo.CreateTemplate(ctx, tmpl); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	if err := repo.DeleteCategory(ctx, cat.ID); err == nil {
		t.Fatalf("expected conflict error deleting category referenced by active product")
	}
}

func TestMemoryRepo_Pricelists(t *testing.T) {
	ctx := context.Background()
	repo := productstorage.NewMemoryRepo()

	pl := &product.Pricelist{
		Name:     "Wholesale",
		Currency: "USD",
	}
	if err := repo.CreatePricelist(ctx, pl); err != nil {
		t.Fatalf("failed to create pricelist: %v", err)
	}

	tmplID := int64(10)
	item := &product.PricelistItem{
		PricelistID:  pl.ID,
		AppliedOn:    product.PricelistAppliedOnTemplate,
		TemplateID:   &tmplID,
		MinQuantity:  10,
		ComputePrice: product.PricelistComputePercentage,
		PercentPrice: 20.0,
	}
	if err := repo.AddPricelistItem(ctx, item); err != nil {
		t.Fatalf("failed to add pricelist item: %v", err)
	}

	fetchedPL, err := repo.GetPricelistByID(ctx, pl.ID)
	if err != nil {
		t.Fatalf("failed to get pricelist: %v", err)
	}
	if len(fetchedPL.Items) != 1 {
		t.Fatalf("expected 1 item attached to pricelist, got %d", len(fetchedPL.Items))
	}
}

func TestMemoryRepo_ConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	repo := productstorage.NewMemoryRepo()

	var wg sync.WaitGroup
	workers := 20

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tmpl := &product.ProductTemplate{
				Name:      i18n.NewTranslation(fmt.Sprintf("Product %d", idx)),
				SalePrice: float64(idx * 10),
			}
			_ = repo.CreateTemplate(ctx, tmpl)

			pageReq := pagination.PageRequest{Page: 1, Limit: 10}
			_, _ = repo.ListTemplates(ctx, nil, pageReq)
		}(i)
	}

	wg.Wait()
}
