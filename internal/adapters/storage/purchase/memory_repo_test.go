package purchasestorage_test

import (
	"context"
	"testing"
	"time"

	purchasestorage "cashflow_backend/internal/adapters/storage/purchase"
	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

func TestMemoryRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := purchasestorage.NewMemoryRepo()

	// 1. NextSequence
	seq, err := repo.NextSequence(ctx, 2026)
	if err != nil {
		t.Fatalf("failed to generate sequence: %v", err)
	}
	if seq != "PO/2026/00001" {
		t.Fatalf("expected PO/2026/00001, got %s", seq)
	}

	// 2. CreateOrder
	order := &purchase.PurchaseOrder{
		Name:          seq,
		PartnerID:     12,
		DateOrder:     time.Now().UTC(),
		State:         purchase.OrderStateDraft,
		InvoiceStatus: purchase.InvoiceStatusNo,
		Currency:      "USD",
		Lines: []purchase.PurchaseOrderLine{
			{
				ProductID:     1,
				Name:          "Raw Metal Sheet",
				ProductQty:    10,
				UnitPrice:     50,
				PriceSubtotal: 500,
				PriceTotal:    500,
			},
		},
	}
	order.RecomputeTotals()

	if err := repo.CreateOrder(ctx, order); err != nil {
		t.Fatalf("failed to create order: %v", err)
	}
	if order.ID <= 0 {
		t.Fatalf("expected order ID > 0, got %d", order.ID)
	}
	if len(order.Lines) != 1 || order.Lines[0].ID <= 0 {
		t.Fatalf("expected order lines with IDs assigned, got %+v", order.Lines)
	}

	// 3. GetOrderByID
	fetched, err := repo.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to get order by ID: %v", err)
	}
	if fetched.Name != seq {
		t.Fatalf("expected name %s, got %s", seq, fetched.Name)
	}
	if len(fetched.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(fetched.Lines))
	}

	// 4. UpdateOrder
	fetched.Note = "Updated instructions for vendor"
	fetched.Lines[0].UnitPrice = 55
	fetched.Lines[0].PriceSubtotal = 550
	fetched.Lines[0].PriceTotal = 550
	fetched.RecomputeTotals()

	if err := repo.UpdateOrder(ctx, fetched); err != nil {
		t.Fatalf("failed to update order: %v", err)
	}

	updated, err := repo.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to get updated order: %v", err)
	}
	if updated.AmountTotal != 550 {
		t.Fatalf("expected amount total 550, got %f", updated.AmountTotal)
	}

	// 5. LinkBill
	if err := repo.LinkBill(ctx, order.ID, 42); err != nil {
		t.Fatalf("failed to link bill: %v", err)
	}
	bills, err := repo.GetLinkedBillIDs(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to get linked bill IDs: %v", err)
	}
	if len(bills) != 1 || bills[0] != 42 {
		t.Fatalf("expected linked bill [42], got %+v", bills)
	}

	// 6. ListOrders with filters
	f := filter.NewFilter()
	f.Add("partner_id", filter.OpEqual, "12")
	pRes, err := repo.ListOrders(ctx, f, pagination.PageRequest{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("failed to list orders: %v", err)
	}
	if pRes.TotalItems != 1 {
		t.Fatalf("expected 1 item, got %d", pRes.TotalItems)
	}

	// 7. DeleteOrder
	if err := repo.DeleteOrder(ctx, order.ID); err != nil {
		t.Fatalf("failed to delete order: %v", err)
	}
	_, err = repo.GetOrderByID(ctx, order.ID)
	if err == nil {
		t.Fatalf("expected error fetching deleted order")
	}
}
