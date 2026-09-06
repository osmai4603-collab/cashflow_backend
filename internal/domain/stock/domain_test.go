package stock_test

import (
	"testing"
	"time"

	"cashflow_backend/internal/domain/stock"
)

func TestLocationValidation(t *testing.T) {
	tests := []struct {
		name    string
		loc     stock.StockLocation
		wantErr bool
	}{
		{
			name: "valid internal location",
			loc: stock.StockLocation{
				Name:  "Shelf A",
				Usage: stock.LocationUsageInternal,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			loc: stock.StockLocation{
				Name:  "   ",
				Usage: stock.LocationUsageInternal,
			},
			wantErr: true,
		},
		{
			name: "invalid usage",
			loc: stock.StockLocation{
				Name:  "Shelf B",
				Usage: "invalid_usage",
			},
			wantErr: true,
		},
		{
			name: "circular parent reference",
			loc: func() stock.StockLocation {
				id := int64(10)
				return stock.StockLocation{
					ID:       id,
					Name:     "Self Parent",
					Usage:    stock.LocationUsageInternal,
					ParentID: &id,
				}
			}(),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.loc.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestLocationCompleteName(t *testing.T) {
	loc := stock.StockLocation{
		Name: "Stock",
	}
	loc.ComputeCompleteName("WH")
	if loc.CompleteName != "WH/Stock" {
		t.Fatalf("expected 'WH/Stock', got '%s'", loc.CompleteName)
	}

	loc2 := stock.StockLocation{
		Name: "Root",
	}
	loc2.ComputeCompleteName("")
	if loc2.CompleteName != "Root" {
		t.Fatalf("expected 'Root', got '%s'", loc2.CompleteName)
	}
}

func TestWarehouseValidation(t *testing.T) {
	tests := []struct {
		name    string
		wh      stock.Warehouse
		wantErr bool
	}{
		{
			name: "valid warehouse",
			wh: stock.Warehouse{
				Name:       "Main Warehouse",
				Code:       "wh",
				LotStockID: 8,
			},
			wantErr: false,
		},
		{
			name: "empty name",
			wh: stock.Warehouse{
				Name:       "",
				Code:       "WH",
				LotStockID: 8,
			},
			wantErr: true,
		},
		{
			name: "empty code",
			wh: stock.Warehouse{
				Name:       "Warehouse 2",
				Code:       " ",
				LotStockID: 8,
			},
			wantErr: true,
		},
		{
			name: "missing lot stock",
			wh: stock.Warehouse{
				Name:       "Warehouse 3",
				Code:       "WH3",
				LotStockID: 0,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.wh.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err == nil && tc.wh.Code != "WH" && tc.wh.Code != "WH3" {
				t.Errorf("expected code to be upper-cased, got %s", tc.wh.Code)
			}
		})
	}
}

func TestStockMoveLifecycle(t *testing.T) {
	move := stock.StockMove{
		Name:           "Item A move",
		ProductID:      101,
		ProductQty:     5,
		LocationID:     8,
		LocationDestID: 3,
		State:          stock.MoveStateDraft,
	}

	if err := move.Validate(); err != nil {
		t.Fatalf("expected valid move, got %v", err)
	}

	// Same src and dest validation failure
	sameLocMove := move
	sameLocMove.LocationDestID = 8
	if err := sameLocMove.Validate(); err == nil {
		t.Fatalf("expected error when src and dest locations are identical")
	}

	// ActionConfirm
	if err := move.ActionConfirm(); err != nil {
		t.Fatalf("ActionConfirm failed: %v", err)
	}
	if move.State != stock.MoveStateConfirmed {
		t.Fatalf("expected state confirmed, got %s", move.State)
	}

	// ActionDone
	if err := move.ActionDone(5); err != nil {
		t.Fatalf("ActionDone failed: %v", err)
	}
	if move.State != stock.MoveStateDone {
		t.Fatalf("expected state done, got %s", move.State)
	}
	if move.QuantityDone != 5 {
		t.Fatalf("expected QuantityDone 5, got %v", move.QuantityDone)
	}

	// Cannot cancel done move
	if err := move.ActionCancel(); err == nil {
		t.Fatalf("expected error cancelling completed move")
	}
}

func TestStockPickingLifecycle(t *testing.T) {
	picking := stock.StockPicking{
		PickingType:    stock.PickingTypeOutgoing,
		LocationID:     8,
		LocationDestID: 3,
		ScheduledDate:  time.Now(),
		Moves: []stock.StockMove{
			{
				Name:           "Delivery move",
				ProductID:      201,
				ProductQty:     10,
				LocationID:     8,
				LocationDestID: 3,
			},
		},
	}

	if err := picking.Validate(); err != nil {
		t.Fatalf("expected valid picking, got %v", err)
	}

	// Confirm
	if err := picking.ActionConfirm("WH/OUT/2026/00001"); err != nil {
		t.Fatalf("ActionConfirm failed: %v", err)
	}
	if picking.State != stock.PickingStateConfirmed {
		t.Fatalf("expected confirmed state, got %s", picking.State)
	}
	if picking.Name != "WH/OUT/2026/00001" {
		t.Fatalf("expected sequence WH/OUT/2026/00001, got %s", picking.Name)
	}

	// Assign
	if err := picking.ActionAssign(); err != nil {
		t.Fatalf("ActionAssign failed: %v", err)
	}
	if picking.State != stock.PickingStateAssigned {
		t.Fatalf("expected assigned state, got %s", picking.State)
	}

	// Validate
	if err := picking.ActionValidate(time.Now()); err != nil {
		t.Fatalf("ActionValidate failed: %v", err)
	}
	if picking.State != stock.PickingStateDone {
		t.Fatalf("expected done state, got %s", picking.State)
	}
	if picking.DateDone == nil {
		t.Fatalf("expected DateDone to be populated")
	}
	if picking.Moves[0].State != stock.MoveStateDone {
		t.Fatalf("expected move to be done, got %s", picking.Moves[0].State)
	}
}

func TestQuantAvailableQuantity(t *testing.T) {
	quant := stock.StockQuant{
		ProductID:        100,
		LocationID:       8,
		Quantity:         50,
		ReservedQuantity: 15,
	}

	if err := quant.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if avail := quant.AvailableQuantity(); avail != 35 {
		t.Fatalf("expected available quantity 35, got %v", avail)
	}

	// Over-reserved check clamps to 0
	quant.ReservedQuantity = 60
	if avail := quant.AvailableQuantity(); avail != 0 {
		t.Fatalf("expected available quantity clamped to 0, got %v", avail)
	}
}
