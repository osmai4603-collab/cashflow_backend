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

func TestMemoryRepoReserveQuantity(t *testing.T) {
	ctx := context.Background()
	repo := stockstorage.NewMemoryRepo()
	if err := repo.SetQuantQuantity(ctx, 10, 8, 5); err != nil {
		t.Fatalf("set quant: %v", err)
	}

	reserved, err := repo.ReserveQuantity(ctx, 10, 8, 3)
	if err != nil {
		t.Fatalf("reserve quantity: %v", err)
	}
	if reserved != 3 {
		t.Fatalf("reserved %v, want 3", reserved)
	}

	reserved, err = repo.ReserveQuantity(ctx, 10, 8, 3)
	if err != nil {
		t.Fatalf("reserve partially available quantity: %v", err)
	}
	if reserved != 2 {
		t.Fatalf("reserved partial quantity %v, want 2", reserved)
	}

	quant, err := repo.GetQuant(ctx, 10, 8)
	if err != nil {
		t.Fatalf("get quant: %v", err)
	}
	if quant.ReservedQuantity != 5 || quant.AvailableQuantity() != 0 {
		t.Fatalf("quant reservation = %v, available = %v; want 5 and 0", quant.ReservedQuantity, quant.AvailableQuantity())
	}
}

func TestMemoryRepoMoveLineLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := stockstorage.NewMemoryRepo()
	move := &stock.StockMove{
		Name:           "component",
		ProductID:      10,
		ProductQty:     2,
		LocationID:     8,
		LocationDestID: 12,
	}
	if err := repo.CreateMove(ctx, move); err != nil {
		t.Fatalf("create move: %v", err)
	}
	line := &stock.StockMoveLine{
		MoveID:           move.ID,
		ProductID:        move.ProductID,
		LocationID:       move.LocationID,
		LocationDestID:   move.LocationDestID,
		ReservedQuantity: 1,
	}
	if err := repo.CreateMoveLine(ctx, line); err != nil {
		t.Fatalf("create move line: %v", err)
	}
	loaded, err := repo.GetMoveByID(ctx, move.ID)
	if err != nil || len(loaded.MoveLines) != 1 || loaded.MoveLines[0].ID != line.ID {
		t.Fatalf("expected move line attached to move, got %+v (err: %v)", loaded, err)
	}
	if err := repo.DeleteMoveLine(ctx, line.ID); err != nil {
		t.Fatalf("delete move line: %v", err)
	}
}

func TestMemoryRepoReserveMoveCreatesMoveLine(t *testing.T) {
	ctx := context.Background()
	repo := stockstorage.NewMemoryRepo()
	if err := repo.SetQuantQuantity(ctx, 10, 8, 3); err != nil {
		t.Fatalf("set quant: %v", err)
	}
	move := &stock.StockMove{
		Name:           "component",
		ProductID:      10,
		ProductQty:     5,
		LocationID:     8,
		LocationDestID: 12,
		State:          stock.MoveStateConfirmed,
	}
	if err := repo.CreateMove(ctx, move); err != nil {
		t.Fatalf("create move: %v", err)
	}
	if err := repo.ReserveMove(ctx, move); err != nil {
		t.Fatalf("reserve move: %v", err)
	}
	if move.ReservedQuantity != 3 || move.State != stock.MoveStateConfirmed {
		t.Fatalf("move reservation = %v, state = %q; want 3 and confirmed", move.ReservedQuantity, move.State)
	}
	lines, err := repo.ListMoveLinesByMoveID(ctx, move.ID)
	if err != nil || len(lines) != 1 || lines[0].ReservedQuantity != 3 {
		t.Fatalf("move lines = %+v (err: %v), want one line with 3 reserved", lines, err)
	}
	move.QuantityDone = 2
	if err := repo.ValidateMovesTx(ctx, []stock.StockMove{*move}); err != nil {
		t.Fatalf("validate move: %v", err)
	}
	lines, err = repo.ListMoveLinesByMoveID(ctx, move.ID)
	if err != nil || len(lines) != 1 || lines[0].QuantityDone != 2 || lines[0].ReservedQuantity != 1 {
		t.Fatalf("completed move lines = %+v (err: %v), want done 2 and reserved 1", lines, err)
	}
	quant, err := repo.GetQuant(ctx, 10, 8)
	if err != nil || quant.ReservedQuantity != 1 {
		t.Fatalf("source reservation = %v (err: %v), want 1", quant.ReservedQuantity, err)
	}
}

func TestMemoryRepoLotAndSerialValidation(t *testing.T) {
	ctx := context.Background()
	repo := stockstorage.NewMemoryRepo()
	serial := &stock.StockLot{ProductID: 10, Name: "SN-001", TrackingMode: stock.TrackingSerial}
	if err := repo.CreateLot(ctx, serial); err != nil {
		t.Fatalf("create serial: %v", err)
	}
	move := &stock.StockMove{
		Name:           "serialized component",
		ProductID:      10,
		ProductQty:     2,
		LocationID:     8,
		LocationDestID: 12,
	}
	if err := repo.CreateMove(ctx, move); err != nil {
		t.Fatalf("create move: %v", err)
	}
	line := &stock.StockMoveLine{
		MoveID:           move.ID,
		ProductID:        10,
		LotID:            &serial.ID,
		LocationID:       8,
		LocationDestID:   12,
		ReservedQuantity: 2,
	}
	if err := repo.CreateMoveLine(ctx, line); err == nil {
		t.Fatal("expected serial quantity greater than one to fail")
	}

	lot := &stock.StockLot{ProductID: 10, Name: "LOT-001", TrackingMode: stock.TrackingLot}
	if err := repo.CreateLot(ctx, lot); err != nil {
		t.Fatalf("create lot: %v", err)
	}
	line.LotID = &lot.ID
	line.ReservedQuantity = 2
	if err := repo.CreateMoveLine(ctx, line); err != nil {
		t.Fatalf("expected lot line to succeed: %v", err)
	}
}
