package stockstorage

import (
	"context"
	"testing"

	"cashflow_backend/internal/domain/stock"
)

func TestMemoryRepoPersistsPackagingAndPackage(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()
	packaging := &stock.ProductPackaging{Name: "Box of 12", ProductID: 10, Qty: 12, CompanyID: 1, Active: true}
	if err := repo.CreatePackaging(ctx, packaging); err != nil { t.Fatal(err) }
	loadedPackaging, err := repo.GetPackagingByID(ctx, packaging.ID); if err != nil { t.Fatal(err) }
	if loadedPackaging.Qty != 12 { t.Fatalf("unexpected packaging: %+v", loadedPackaging) }
	packageItem := &stock.StockPackage{Name: "PACK/2026/00001", LocationID: 8, CompanyID: 1}
	if err := repo.CreatePackage(ctx, packageItem); err != nil { t.Fatal(err) }
	loadedPackage, err := repo.GetPackageByID(ctx, packageItem.ID); if err != nil { t.Fatal(err) }
	if loadedPackage.Name != packageItem.Name { t.Fatalf("unexpected package: %+v", loadedPackage) }
}