package stockstorage_test

import (
	"context"
	"testing"
	"time"

	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	"cashflow_backend/internal/domain/stock"
	"cashflow_backend/internal/platform/pagination"
)

func TestMemoryRepoSeeds(t *testing.T) {
	ctx := context.Background()
	repo := stockstorage.NewMemoryRepo()

	// Verify default locations
	loc, err := repo.GetLocationByID(ctx, 8)
	if err != nil {
		t.Fatalf("expected seeded WH/Stock location #8, got error: %v", err)
	}
	if loc.CompleteName != "WH/Stock" || loc.Usage != stock.LocationUsageInternal {
		t.Fatalf("unexpected location data: %+v", loc)
	}

	// Verify default warehouse
	wh, err := repo.GetDefaultWarehouse(ctx)
	if err != nil {
		t.Fatalf("expected default warehouse, got: %v", err)
	}
	if wh.Code != "WH" || wh.LotStockID != 8 {
		t.Fatalf("unexpected default warehouse: %+v", wh)
	}
}

func TestMemoryRepoSequences(t *testing.T) {
	ctx := context.Background()
	repo := stockstorage.NewMemoryRepo()

	seq1, err := repo.NextSequence(ctx, stock.PickingTypeIncoming, 2026)
	if err != nil || seq1 != "WH/IN/2026/00001" {
		t.Fatalf("expected WH/IN/2026/00001, got %s, err %v", seq1, err)
	}

	seq2, err := repo.NextSequence(ctx, stock.PickingTypeOutgoing, 2026)
	if err != nil || seq2 != "WH/OUT/2026/00001" {
		t.Fatalf("expected WH/OUT/2026/00001, got %s, err %v", seq2, err)
	}

	seq3, err := repo.NextSequence(ctx, stock.PickingTypeInternal, 2026)
	if err != nil || seq3 != "WH/INT/2026/00001" {
		t.Fatalf("expected WH/INT/2026/00001, got %s, err %v", seq3, err)
	}
}

func TestMemoryRepoPickingValidationAndQuants(t *testing.T) {
	ctx := context.Background()
	repo := stockstorage.NewMemoryRepo()

	// 1. Create incoming receipt (Supplier Loc #2 -> Internal WH/Stock #8)
	picking := &stock.StockPicking{
		Name:           "WH/IN/2026/00001",
		PickingType:    stock.PickingTypeIncoming,
		State:          stock.PickingStateDraft,
		LocationID:     2,
		LocationDestID: 8,
		ScheduledDate:  time.Now(),
		Moves: []stock.StockMove{
			{
				Name:           "Laptop receipt",
				ProductID:      55,
				ProductQty:     20,
				LocationID:     2,
				LocationDestID: 8,
				State:          stock.MoveStateDraft,
			},
		},
	}

	if err := repo.CreatePicking(ctx, picking); err != nil {
		t.Fatalf("CreatePicking failed: %v", err)
	}
	if picking.ID == 0 {
		t.Fatalf("expected picking ID to be assigned")
	}

	// 2. Validate picking via transaction
	if err := repo.ValidatePickingTx(ctx, picking); err != nil {
		t.Fatalf("ValidatePickingTx failed: %v", err)
	}

	// 3. Verify destination quant increased
	destQuant, err := repo.GetQuant(ctx, 55, 8)
	if err != nil {
		t.Fatalf("GetQuant failed: %v", err)
	}
	if destQuant.Quantity != 20 {
		t.Fatalf("expected dest quant 20, got %v", destQuant.Quantity)
	}

	// 4. Verify On-Hand report
	items, err := repo.GetOnHandStock(ctx, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetOnHandStock failed: %v", err)
	}
	if len(items) != 1 || items[0].Quantity != 20 || items[0].LocationID != 8 {
		t.Fatalf("unexpected on-hand report items: %+v", items)
	}

	// 5. Test outgoing delivery (Internal WH/Stock #8 -> Customer Loc #3)
	outPicking := &stock.StockPicking{
		Name:           "WH/OUT/2026/00001",
		PickingType:    stock.PickingTypeOutgoing,
		State:          stock.PickingStateDraft,
		LocationID:     8,
		LocationDestID: 3,
		ScheduledDate:  time.Now(),
		Moves: []stock.StockMove{
			{
				Name:           "Laptop delivery",
				ProductID:      55,
				ProductQty:     5,
				LocationID:     8,
				LocationDestID: 3,
				State:          stock.MoveStateDraft,
			},
		},
	}

	if err := repo.CreatePicking(ctx, outPicking); err != nil {
		t.Fatalf("CreatePicking (out) failed: %v", err)
	}
	if err := repo.ValidatePickingTx(ctx, outPicking); err != nil {
		t.Fatalf("ValidatePickingTx (out) failed: %v", err)
	}

	// WH/Stock should now have 15
	stockQuant, err := repo.GetQuant(ctx, 55, 8)
	if err != nil || stockQuant.Quantity != 15 {
		t.Fatalf("expected stock quant to be 15, got %v (err: %v)", stockQuant.Quantity, err)
	}
}

func TestMemoryRepoDirectQuantAdjustment(t *testing.T) {
	ctx := context.Background()
	repo := stockstorage.NewMemoryRepo()

	if err := repo.SetQuantQuantity(ctx, 10, 8, 50); err != nil {
		t.Fatalf("SetQuantQuantity failed: %v", err)
	}

	q, err := repo.GetQuant(ctx, 10, 8)
	if err != nil || q.Quantity != 50 {
		t.Fatalf("expected 50, got %v", q.Quantity)
	}

	if err := repo.UpdateQuantQuantity(ctx, 10, 8, -10); err != nil {
		t.Fatalf("UpdateQuantQuantity failed: %v", err)
	}

	q2, err := repo.GetQuant(ctx, 10, 8)
	if err != nil || q2.Quantity != 40 {
		t.Fatalf("expected 40, got %v", q2.Quantity)
	}

	res, err := repo.ListQuants(ctx, nil, pagination.PageRequest{Page: 1, Limit: 10})
	if err != nil || res.TotalItems != 1 {
		t.Fatalf("expected 1 quant item, got %d", res.TotalItems)
	}
}
