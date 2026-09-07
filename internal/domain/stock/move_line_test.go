package stock

import "testing"

func TestStockMoveLineValidate(t *testing.T) {
	line := &StockMoveLine{
		MoveID:           1,
		ProductID:        2,
		LocationID:       8,
		LocationDestID:   12,
		ReservedQuantity: 2,
		QuantityDone:     1,
	}
	if err := line.Validate(); err != nil {
		t.Fatalf("expected valid move line, got %v", err)
	}

	invalid := *line
	invalid.LocationDestID = invalid.LocationID
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected identical locations to fail")
	}

	invalid = *line
	invalid.ReservedQuantity = -1
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected negative reservation to fail")
	}
}
