package stock_test

import (
	"testing"
	"math"
	"cashflow_backend/internal/domain/stock"
)

func TestComputeSplitValue(t *testing.T) {
	tests := []struct {
		name       string
		split      stock.SplitMethod
		priceUnit  float64
		totalQty   float64
		totalCost  float64
		totalLines float64
		qty        float64
		formerCost float64
		expected   float64
	}{
		{
			name:      "Split Equal",
			split:     stock.SplitEqual,
			priceUnit: 100,
			totalLines: 4,
			expected:  25,
		},
		{
			name:      "Split By Quantity",
			split:     stock.SplitByQuantity,
			priceUnit: 100,
			totalQty:  10,
			qty:       3,
			expected:  30,
		},
		{
			name:      "Split By Cost",
			split:     stock.SplitByCost,
			priceUnit: 1000,
			totalCost: 500,
			formerCost: 100,
			expected:  200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stock.ComputeSplitValue(tt.split, tt.priceUnit, tt.totalQty, 0, 0, tt.totalCost, tt.totalLines, tt.qty, 0, 0, tt.formerCost)
			if math.Abs(got-tt.expected) > 0.0001 {
				t.Errorf("ComputeSplitValue() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLandedCostJournalAmount(t *testing.T) {
	// Formula: additionalCost * (moveRemainingQty / quantity)
	tests := []struct {
		name             string
		additionalCost   float64
		quantity         float64
		moveRemainingQty float64
		expected         float64
	}{
		{
			name:             "Full Remaining",
			additionalCost:   50,
			quantity:         10,
			moveRemainingQty: 10,
			expected:         50,
		},
		{
			name:             "Half Remaining",
			additionalCost:   50,
			quantity:         10,
			moveRemainingQty: 5,
			expected:         25,
		},
		{
			name:             "None Remaining",
			additionalCost:   50,
			quantity:         10,
			moveRemainingQty: 0,
			expected:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stock.LandedCostJournalAmount(tt.additionalCost, tt.quantity, tt.moveRemainingQty)
			if math.Abs(got-tt.expected) > 0.0001 {
				t.Errorf("LandedCostJournalAmount() = %v, want %v", got, tt.expected)
			}
		})
	}
}
