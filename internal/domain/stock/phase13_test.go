package stock

import (
	"math"
	"testing"
	"time"
)

// round4 mirrors the usecase rounding helper used across tests.
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

func mustSplitTotal(t *testing.T, split SplitMethod, priceUnit, totalQty, totalWeight, totalVolume, totalCost float64, sizes []struct{ qty, weight, volume, cost, lines float64 }) float64 {
	t.Helper()
	sum := 0.0
	for _, s := range sizes {
		got := ComputeSplitValue(split, priceUnit, totalQty, totalWeight, totalVolume, totalCost, s.lines, s.qty, s.weight, s.volume, s.cost)
		sum += got
		if got < 0 {
			t.Fatalf("split %s produced a negative allocation: %f", split, got)
		}
	}
	return sum
}

func TestComputeSplitValueByQuantity(t *testing.T) {
	total := mustSplitTotal(t, SplitByQuantity, 100, 50, 0, 0, 0, []struct {
		qty, weight, volume, cost, lines float64
	}{
		{qty: 30}, {qty: 20},
	})
	// 100/50*30 = 60 ; 100/50*20 = 40 → total == priceUnit
	if total != 100 {
		t.Fatalf("by_quantity allocations must sum to price_unit, got %f", total)
	}
}

func TestComputeSplitValueByWeight(t *testing.T) {
	total := mustSplitTotal(t, SplitByWeight, 90, 0, 180, 0, 0, []struct {
		qty, weight, volume, cost, lines float64
	}{
		{weight: 120}, {weight: 60},
	})
	if total != 90 {
		t.Fatalf("by_weight allocations must sum to price_unit, got %f", total)
	}
}

func TestComputeSplitValueByVolume(t *testing.T) {
	total := mustSplitTotal(t, SplitByVolume, 45, 0, 0, 150, 0, []struct {
		qty, weight, volume, cost, lines float64
	}{
		{volume: 100}, {volume: 50},
	})
	if total != 45 {
		t.Fatalf("by_volume allocations must sum to price_unit, got %f", total)
	}
}

func TestComputeSplitValueByCurrentCost(t *testing.T) {
	total := mustSplitTotal(t, SplitByCost, 80, 0, 0, 0, 400, []struct {
		qty, weight, volume, cost, lines float64
	}{
		{cost: 250}, {cost: 150},
	})
	// 80/400*250 = 50 ; 80/400*150 = 30 → total == priceUnit
	if total != 80 {
		t.Fatalf("by_current_cost allocations must sum to price_unit, got %f", total)
	}
}

func TestComputeSplitValueEqual(t *testing.T) {
	total := mustSplitTotal(t, SplitEqual, 30, 0, 0, 0, 0, []struct {
		qty, weight, volume, cost, lines float64
	}{
		{lines: 3}, {lines: 3}, {lines: 3},
	})
	if total != 30 {
		t.Fatalf("equal allocations must sum to price_unit, got %f", total)
	}
}

func TestComputeSplitValueZeroGuards(t *testing.T) {
	// totalQty == 0 → falls back to equal split.
	if got := ComputeSplitValue(SplitByQuantity, 10, 0, 0, 0, 0, 2, 100, 0, 0, 0); got != 5 {
		t.Fatalf("expected equal fallback when total is zero, got %f", got)
	}
	// totalLines == 0 → 0 (no crash).
	if got := ComputeSplitValue(SplitByWeight, 10, 0, 0, 0, 0, 0, 5, 5, 0, 0); got != 0 {
		t.Fatalf("expected 0 when nothing to split across, got %f", got)
	}
}

func TestLandedCostJournalAmount(t *testing.T) {
	cases := []struct {
		name                string
		additional, qty, rem float64
		want                float64
	}{
		{"no remaining stock booked", 5, 100, 0, 0},
		{"zero quantity guard", 5, 0, 100, 0},
		{"half remaining", 8, 100, 50, 4},
		{"full remaining", 8, 100, 100, 8},
		{"negative additional reverses", -8, 100, 50, -4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := LandedCostJournalAmount(tc.additional, tc.qty, tc.rem); got != tc.want {
				t.Fatalf("LandedCostJournalAmount(%v, %v, %v) = %v, want %v", tc.additional, tc.qty, tc.rem, got, tc.want)
			}
		})
	}
}

func TestValuationAdjustmentComputeFinalCost(t *testing.T) {
	v := ValuationAdjustment{FormerCost: 10.25, AdditionalCost: 1.75}
	v.ComputeFinalCost()
	if v.FinalCost != 12.0 {
		t.Fatalf("final cost must be 12.0, got %v", v.FinalCost)
	}
}

func TestOrderpointComputeQtyToOrder(t *testing.T) {
	cases := []struct {
		name           string
		min, max, mult float64
		forecast       float64
		want           float64
	}{
		{"below minimum uses max as target", 10, 20, 0, 8, 12},
		{"max zero uses min", 10, 0, 0, 5, 5},
		{"forecast above target nothing to order", 10, 20, 0, 25, 0},
		{"clamps to multiple of 5", 10, 20, 5, 8, 15},
		{"zero min and max nothing to order", 0, 0, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			op := Orderpoint{MinQty: tc.min, MaxQty: tc.max, QtyMultiple: tc.mult, QtyForecast: tc.forecast}
			op.Validate()
			op.ComputeQtyToOrder()
			if round4(op.QtyToOrder) != tc.want {
				t.Fatalf("QtyToOrder = %v, want %v", op.QtyToOrder, tc.want)
			}
		})
	}
}

func TestOrderpointNeedsOrderAndSnooze(t *testing.T) {
	now := time.Now().UTC()
	inFuture := now.Add(time.Hour)
	op := Orderpoint{
		ProductID: 1, WarehouseID: 1, LocationID: 1,
		MinQty: 10, MaxQty: 20, QtyForecast: 5,
		Trigger:      OrderpointTriggerManual,
		SnoozedUntil: &inFuture,
	}
	if err := op.Validate(); err != nil {
		t.Fatalf("manual snoozed rule must validate: %v", err)
	}
	if !op.IsSnoozed(now) || op.NeedsOrder(now) {
		t.Fatal("snoozed manual rule must not need ordering")
	}
	op.SnoozedUntil = nil
	if !op.NeedsOrder(now) {
		t.Fatal("below-min rule must need ordering once unsnoozed")
	}
}

func TestOrderpointValidateConstraints(t *testing.T) {
	base := &Orderpoint{ProductID: 1, WarehouseID: 1, LocationID: 1, MinQty: 10, MaxQty: 20}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid orderpoint rejected: %v", err)
	}

	maxLessThanMin := &Orderpoint{ProductID: 1, WarehouseID: 1, LocationID: 1, MinQty: 20, MaxQty: 10}
	if err := maxLessThanMin.Validate(); err == nil {
		t.Fatal("max < min must be rejected")
	}

	negManual := &Orderpoint{ProductID: 1, WarehouseID: 1, LocationID: 1, MinQty: 10, MaxQty: 20, QtyToOrderManual: -1}
	if err := negManual.Validate(); err == nil {
		t.Fatal("negative manual order quantity must be rejected")
	}

	badSource := &Orderpoint{ProductID: 1, WarehouseID: 1, LocationID: 1, MinQty: 10, MaxQty: 20, Source: "manufacture"}
	if err := badSource.Validate(); err == nil {
		t.Fatal("unsupported source must be rejected")
	}

	autoSnooze := &Orderpoint{ProductID: 1, WarehouseID: 1, LocationID: 1, MinQty: 10, MaxQty: 20, Trigger: OrderpointTriggerAuto}
	until := time.Now().Add(time.Hour)
	autoSnooze.SnoozedUntil = &until
	if err := autoSnooze.Validate(); err == nil {
		t.Fatal("snooze on auto rule must be rejected")
	}
}

func TestLandedCostValidate(t *testing.T) {
	lc := LandedCost{
		PickingIDs: []int64{1},
		JournalID:  6,
		CostLines: []LandedCostLine{
			{Name: "Freight", ProductID: 1, AccountID: 17, PriceUnit: 50, SplitMethod: SplitByQuantity},
		},
	}
	if err := lc.Validate(); err != nil {
		t.Fatalf("valid landed cost rejected: %v", err)
	}
	lc.ComputeTotalAmount()
	if lc.AmountTotal != 50 {
		t.Fatalf("amount_total must equal 50, got %v", lc.AmountTotal)
	}

	if err := (&LandedCost{JournalID: 6, CostLines: []LandedCostLine{{Name: "x", AccountID: 1}}}).Validate(); err == nil {
		t.Fatal("landed cost without pickings must be rejected")
	}
	badSplit := LandedCost{PickingIDs: []int64{1}, JournalID: 6, CostLines: []LandedCostLine{{Name: "x", AccountID: 1, SplitMethod: "random"}}}
	if err := badSplit.Validate(); err == nil {
		t.Fatal("invalid split method must be rejected")
	}
}