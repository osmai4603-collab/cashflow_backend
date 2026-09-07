package stockusecase_test

import (
	"context"
	"testing"
	stockstorage "cashflow_backend/internal/adapters/storage/stock"
	"cashflow_backend/internal/domain/stock"
	"cashflow_backend/internal/platform/pagination"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

func TestLocationUseCases(t *testing.T) {
	ctx := context.Background()
	repo := stockstorage.NewMemoryRepo()
	uc := stockusecase.New(repo, nil, nil, nil, nil, nil, nil)

	// Create child location under WH/Stock (id: 8)
	parentID := int64(8)
	loc, err := uc.CreateLocation(ctx, stockusecase.CreateLocationInput{
		Name:     "Zone 1",
		Usage:    stock.LocationUsageInternal,
		ParentID: &parentID,
	})
	if err != nil {
		t.Fatalf("CreateLocation failed: %v", err)
	}
	if loc.CompleteName != "WH/Stock/Zone 1" {
		t.Fatalf("expected 'WH/Stock/Zone 1', got '%s'", loc.CompleteName)
	}

	// Update location
	newName := "Zone A"
	updated, err := uc.UpdateLocation(ctx, loc.ID, stockusecase.UpdateLocationInput{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("UpdateLocation failed: %v", err)
	}
	if updated.CompleteName != "WH/Stock/Zone A" {
		t.Fatalf("expected 'WH/Stock/Zone A', got '%s'", updated.CompleteName)
	}

	// Delete
	if err := uc.DeleteLocation(ctx, loc.ID); err != nil {
		t.Fatalf("DeleteLocation failed: %v", err)
	}
	if _, err := uc.GetLocation(ctx, loc.ID); err == nil {
		t.Fatalf("expected error getting deleted location")
	}
}

func TestWarehouseUseCases(t *testing.T) {
	ctx := context.Background()
	repo := stockstorage.NewMemoryRepo()
	uc := stockusecase.New(repo, nil, nil, nil, nil, nil, nil)

	// 1. Invalid lot stock usage
	viewLocID := int64(7) // WH
	_, err := uc.CreateWarehouse(ctx, stockusecase.CreateWarehouseInput{
		Name:       "Secondary WH",
		Code:       "SEC",
		LotStockID: viewLocID, // View location, not internal!
	})
	if err == nil {
		t.Fatalf("expected error when lot stock location is not internal")
	}

	// 2. Valid warehouse with internal lot stock #8
	wh, err := uc.CreateWarehouse(ctx, stockusecase.CreateWarehouseInput{
		Name:       "Secondary WH",
		Code:       "SEC",
		LotStockID: 8,
	})
	if err != nil {
		t.Fatalf("CreateWarehouse failed: %v", err)
	}
	if wh.Code != "SEC" {
		t.Fatalf("expected SEC, got %s", wh.Code)
	}
}

func TestPickingReceiptAndDeliveryUseCases(t *testing.T) {
	ctx := context.Background()
	repo := stockstorage.NewMemoryRepo()
	uc := stockusecase.New(repo, nil, nil, nil, nil, nil, nil)

	// 1. Create Receipt (Incoming)
	receipt, err := uc.CreatePicking(ctx, stockusecase.CreatePickingInput{
		PickingType: stock.PickingTypeIncoming,
		Origin:      "PO/2026/00001",
		Moves: []stockusecase.CreateMoveInput{
			{
				ProductID:  10,
				ProductQty: 100,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreatePicking failed: %v", err)
	}

	// 2. Validate receipt
	validatedReceipt, err := uc.ValidatePicking(ctx, receipt.ID, stockusecase.ValidatePickingInput{})
	if err != nil {
		t.Fatalf("ValidatePicking failed: %v", err)
	}
	if validatedReceipt.State != stock.PickingStateDone {
		t.Fatalf("expected state done, got %s", validatedReceipt.State)
	}

	// Verify on hand
	onHand, err := uc.GetOnHandStock(ctx, nil, nil, nil)
	if err != nil || len(onHand) != 1 || onHand[0].Quantity != 100 {
		t.Fatalf("expected 100 on hand, got %+v", onHand)
	}

	// 3. Outgoing Delivery with Insufficient Stock (trying 150)
	overDelivery, err := uc.CreatePicking(ctx, stockusecase.CreatePickingInput{
		PickingType: stock.PickingTypeOutgoing,
		Origin:      "SO/2026/00001",
		Moves: []stockusecase.CreateMoveInput{
			{
				ProductID:  10,
				ProductQty: 150,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreatePicking (over) failed: %v", err)
	}

	_, err = uc.ValidatePicking(ctx, overDelivery.ID, stockusecase.ValidatePickingInput{})
	if err == nil {
		t.Fatalf("expected insufficient stock error, got nil")
	}

	// 4. Outgoing Delivery within Stock (40 items)
	validDelivery, err := uc.CreatePicking(ctx, stockusecase.CreatePickingInput{
		PickingType: stock.PickingTypeOutgoing,
		Origin:      "SO/2026/00002",
		Moves: []stockusecase.CreateMoveInput{
			{
				ProductID:  10,
				ProductQty: 40,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreatePicking (valid) failed: %v", err)
	}

	valDel, err := uc.ValidatePicking(ctx, validDelivery.ID, stockusecase.ValidatePickingInput{})
	if err != nil {
		t.Fatalf("ValidatePicking failed: %v", err)
	}
	if valDel.State != stock.PickingStateDone {
		t.Fatalf("expected done state, got %s", valDel.State)
	}

	// 5. Check remaining balance (should be 60)
	quant, err := repo.GetQuant(ctx, 10, 8)
	if err != nil || quant.Quantity != 60 {
		t.Fatalf("expected remaining stock 60, got %v (err: %v)", quant.Quantity, err)
	}
}

func TestStockAdjustmentUseCase(t *testing.T) {
	ctx := context.Background()
	repo := stockstorage.NewMemoryRepo()
	uc := stockusecase.New(repo, nil, nil, nil, nil, nil, nil)

	// Set initial stock to 25 via adjustment
	q, err := uc.AdjustStock(ctx, stockusecase.StockAdjustmentInput{
		ProductID:   101,
		LocationID:  8, // WH/Stock
		NewQuantity: 25,
		Note:        "Initial inventory count",
	})
	if err != nil {
		t.Fatalf("AdjustStock (gain) failed: %v", err)
	}
	if q.Quantity != 25 {
		t.Fatalf("expected 25, got %v", q.Quantity)
	}

	// Adjust down to 20 (shrinkage/damage)
	q2, err := uc.AdjustStock(ctx, stockusecase.StockAdjustmentInput{
		ProductID:   101,
		LocationID:  8,
		NewQuantity: 20,
		Note:        "Damaged item write-off",
	})
	if err != nil {
		t.Fatalf("AdjustStock (loss) failed: %v", err)
	}
	if q2.Quantity != 20 {
		t.Fatalf("expected 20, got %v", q2.Quantity)
	}

	// Check move history contains 2 adjustment moves
	moves, err := uc.ListMoves(ctx, nil, pagination.PageRequest{Page: 1, Limit: 10})
	if err != nil || moves.TotalItems != 2 {
		t.Fatalf("expected 2 moves, got %d (err: %v)", moves.TotalItems, err)
	}
}
