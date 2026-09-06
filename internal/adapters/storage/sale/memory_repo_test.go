package salestorage_test

import (
	"context"
	"testing"
	"time"

	salestorage "cashflow_backend/internal/adapters/storage/sale"
	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

func TestMemoryRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := salestorage.NewMemoryRepo()

	// 1. NextSequence
	seq, err := repo.NextSequence(ctx, 2026)
	if err != nil {
		t.Fatalf("failed to generate sequence: %v", err)
	}
	if seq != "SO/2026/00001" {
		t.Errorf("expected sequence SO/2026/00001, got %s", seq)
	}

	// 2. CreateOrder
	order := &sale.SaleOrder{
		Name:      seq,
		PartnerID: 10,
		DateOrder: time.Now(),
		State:     sale.OrderStateDraft,
		Currency:  "USD",
		Lines: []sale.SaleOrderLine{
			{
				ProductID:     1,
				Name:          "Product A",
				ProductUomQty: 2.0,
				UnitPrice:     50.0,
				PriceSubtotal: 100.0,
				PriceTotal:    100.0,
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
		t.Fatalf("expected order line ID > 0")
	}

	// 3. GetOrderByID
	fetched, err := repo.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to get order by id: %v", err)
	}
	if fetched.Name != seq {
		t.Errorf("expected name %s, got %s", seq, fetched.Name)
	}
	if len(fetched.Lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(fetched.Lines))
	}

	// 4. GetOrderByName
	byName, err := repo.GetOrderByName(ctx, seq)
	if err != nil {
		t.Fatalf("failed to get order by name: %v", err)
	}
	if byName.ID != order.ID {
		t.Errorf("expected id %d, got %d", order.ID, byName.ID)
	}

	// 5. UpdateOrder
	fetched.Lines = append(fetched.Lines, sale.SaleOrderLine{
		ProductID:     2,
		Name:          "Product B",
		ProductUomQty: 1.0,
		UnitPrice:     200.0,
		PriceSubtotal: 200.0,
		PriceTotal:    200.0,
	})
	fetched.RecomputeTotals()
	if err := repo.UpdateOrder(ctx, fetched); err != nil {
		t.Fatalf("failed to update order: %v", err)
	}

	updated, err := repo.GetOrderByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to get updated order: %v", err)
	}
	if len(updated.Lines) != 2 {
		t.Errorf("expected 2 lines after update, got %d", len(updated.Lines))
	}
	if updated.AmountTotal != 300.0 {
		t.Errorf("expected total 300.0, got %f", updated.AmountTotal)
	}

	// 6. LinkInvoice
	if err := repo.LinkInvoice(ctx, order.ID, 42); err != nil {
		t.Fatalf("failed to link invoice: %v", err)
	}
	invoices, err := repo.GetLinkedInvoiceIDs(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to get linked invoices: %v", err)
	}
	if len(invoices) != 1 || invoices[0] != 42 {
		t.Errorf("expected invoice 42 linked, got %v", invoices)
	}

	// 7. ListOrders with filter
	f := filter.NewFilter()
	f.Add("partner_id", filter.OpEqual, 10)
	page := pagination.PageRequest{Page: 1, Limit: 10}
	res, err := repo.ListOrders(ctx, f, page)
	if err != nil {
		t.Fatalf("failed to list orders: %v", err)
	}
	if res.TotalItems != 1 {
		t.Errorf("expected 1 item, got %d", res.TotalItems)
	}

	// 8. DeleteOrder
	if err := repo.DeleteOrder(ctx, order.ID); err != nil {
		t.Fatalf("failed to delete order: %v", err)
	}
	_, err = repo.GetOrderByID(ctx, order.ID)
	if err == nil {
		t.Errorf("expected not found error after deletion")
	}
}
