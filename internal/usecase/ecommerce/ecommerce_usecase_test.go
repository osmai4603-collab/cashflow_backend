package ecommerceusecase

import (
	"context"
	"testing"
	"time"

	ecommercestorage "cashflow_backend/internal/adapters/storage/ecommerce"
	"cashflow_backend/internal/domain/ecommerce"
)

func TestEcommerceUseCase_CartAndCatalog(t *testing.T) {
	repo := ecommercestorage.NewMemoryRepo()
	ctx := context.Background()
	now := time.Now().UTC()

	if err := repo.SaveCart(ctx, &ecommerce.Cart{ID: 1, WebsiteID: 10, SessionUUID: "s-1", Currency: "USD", State: ecommerce.CartStateActive, LastActivityAt: now, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("save cart: %v", err)
	}

	uc := New(repo, nil)
	res, err := uc.GetCart(ctx, "s-1", 0)
	if err != nil {
		t.Fatalf("get cart: %v", err)
	}
	if res == nil || res.SessionUUID != "s-1" {
		t.Fatalf("unexpected cart: %#v", res)
	}

	if err := uc.ApplyCoupon(ctx, 1, "SAVE10"); err != nil {
		t.Fatalf("expected valid coupon to be accepted: %v", err)
	}
}
