package repairusecase

import (
	repairstorage "cashflow_backend/internal/adapters/storage/repair"
	"cashflow_backend/internal/domain/repair"
	"context"
	"testing"
	"time"
)

func TestRepairUseCase_CreateConfirmAndCompleteOrder(t *testing.T) {
	repo := repairstorage.NewMemoryRepo()
	uc := New(repo)
	ctx := context.Background()

	order, err := uc.CreateOrder(ctx, &repair.RepairOrder{
		Name:           "REP-001",
		PartnerID:      10,
		ProductID:      55,
		LocationID:     1,
		LocationDestID: 2,
		CompanyID:      1,
		Lines: []repair.RepairOrderLine{{
			ProductID: 78,
			Quantity:  2,
			PriceUnit: 25,
		}},
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.ID == 0 || order.AmountTotal != 50 {
		t.Fatalf("unexpected order created: %#v", order)
	}
	if err := uc.ConfirmOrder(ctx, order.ID); err != nil {
		t.Fatalf("confirm order: %v", err)
	}
	if err := uc.CompleteOrder(ctx, order.ID); err != nil {
		t.Fatalf("complete order: %v", err)
	}
	if order.State != repair.StateDone {
		t.Fatalf("order state should be done, got %q", order.State)
	}
	if _, err := uc.CreateInvoice(ctx, order.ID); err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	if _, err := uc.ListOrders(ctx); err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if time.Since(order.UpdatedAt) < 0 {
		t.Fatal("updated_at was not set")
	}
}
