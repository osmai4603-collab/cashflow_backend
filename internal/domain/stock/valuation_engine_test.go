package stock_test

import (
	"testing"
	"time"

	"cashflow_backend/internal/domain/stock"
)

func TestRunFIFO_ConsumesOldestFirst(t *testing.T) {
	stack := stock.FIFOStack{
		{MoveID: 1, Qty: 10, Value: 100},  // $10/unit
		{MoveID: 2, Qty: 5, Value: 75},    // $15/unit
		{MoveID: 3, Qty: 3, Value: 90},    // $30/unit
	}

	value, remaining := stock.RunFIFO(stack, 12)
	if value != 130 { // 10*10 + 2*15
		t.Fatalf("expected value 130, got %v", value)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 remaining layers, got %d", len(remaining))
	}
	if remaining[0].MoveID != 2 || remaining[0].Qty != 3 || remaining[0].Value != 45 {
		t.Fatalf("expected partial layer qty=3 value=45, got %+v", remaining[0])
	}
	if remaining[1].MoveID != 3 || remaining[1].Qty != 3 {
		t.Fatalf("expected untouched layer 3, got %+v", remaining[1])
	}
}

func TestRunFIFO_ExactConsumption(t *testing.T) {
	stack := stock.FIFOStack{
		{MoveID: 1, Qty: 10, Value: 100},
	}

	value, remaining := stock.RunFIFO(stack, 10)
	if value != 100 {
		t.Fatalf("expected value 100, got %v", value)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected empty remaining stack, got %d layers", len(remaining))
	}
}

func TestRunFIFO_ExtrapolatesBeyondStock(t *testing.T) {
	stack := stock.FIFOStack{
		{MoveID: 1, Qty: 10, Value: 100}, // $10/unit
	}

	value, remaining := stock.RunFIFO(stack, 15)
	if value != 150 { // 10 @ $10 + 5 extrapolated @ $10
		t.Fatalf("expected value 150, got %v", value)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected empty remaining stack, got %d layers", len(remaining))
	}
}

func TestRunFIFO_ZeroQty(t *testing.T) {
	stack := stock.FIFOStack{{MoveID: 1, Qty: 10, Value: 100}}
	value, remaining := stock.RunFIFO(stack, 0)
	if value != 0 {
		t.Fatalf("expected value 0, got %v", value)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected untouched stack, got %d layers", len(remaining))
	}
}

func TestFIFOStackUnitPrice(t *testing.T) {
	stack := stock.FIFOStack{
		{MoveID: 1, Qty: 10, Value: 100},
		{MoveID: 2, Qty: 10, Value: 200},
	}
	if unit := stack.UnitPrice(); unit != 15 { // (100+200)/20
		t.Fatalf("expected unit price 15, got %v", unit)
	}
	if empty := (stock.FIFOStack{}).UnitPrice(); empty != 0 {
		t.Fatalf("expected unit price 0 for empty stack, got %v", empty)
	}
}

func TestRunAverageBatch(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// in 10 @ 100, in 10 @ 200, out 10 → avg = (100+200)/20 = 15 before the out
	moves := []stock.StockMove{
		{ID: 1, Date: base, IsIn: true, ProductQty: 10, QuantityDone: 10, Value: 100, StandardPrice: 10},
		{ID: 2, Date: base.Add(time.Hour), IsIn: true, ProductQty: 10, QuantityDone: 10, Value: 200, StandardPrice: 20},
		{ID: 3, Date: base.Add(2 * time.Hour), IsOut: true, ProductQty: 10, QuantityDone: 10},
	}

	avg := stock.RunAverageBatch(moves)
	if avg != 15 {
		t.Fatalf("expected average cost 15, got %v", avg)
	}
}

func TestComputeOutValue_Standard(t *testing.T) {
	m := stock.StockMove{
		StandardPrice: 12.5,
		ProductQty:    4,
		QuantityDone:  4,
		IsOut:         true,
	}
	cfg := stock.ProductValuationConfig{CostMethod: stock.CostStandard}
	value, remaining := stock.ComputeOutValue(m, cfg, nil)
	if value != 50 {
		t.Fatalf("expected value 50, got %v", value)
	}
	if remaining != nil {
		t.Fatalf("expected unchanged stack for standard cost, got %v", remaining)
	}
}

func TestComputeOutValue_FIFO(t *testing.T) {
	m := stock.StockMove{
		ProductQty:   3,
		QuantityDone: 3,
		IsOut:        true,
	}
	cfg := stock.ProductValuationConfig{CostMethod: stock.CostFIFO}
	stack := stock.FIFOStack{
		{MoveID: 1, Qty: 10, Value: 100},
	}
	value, remaining := stock.ComputeOutValue(m, cfg, stack)
	if value != 30 { // 3 @ $10
		t.Fatalf("expected value 30, got %v", value)
	}
	if remaining[0].Qty != 7 {
		t.Fatalf("expected remaining qty 7, got %v", remaining[0].Qty)
	}
}

func TestShouldCreateAccountMove(t *testing.T) {
	srcAcc := int64(100)
	destAcc := int64(200)

	m := stock.StockMove{ProductQty: 5, QuantityDone: 5, IsIn: true}
	cfg := stock.ProductValuationConfig{Valuation: stock.ValuationRealTime}

	// Boundary on destination → create
	if !stock.ShouldCreateAccountMove(m, nil, &destAcc, cfg) {
		t.Fatalf("expected account move to be created when dest has boundary")
	}
	// Boundary on source → create
	if !stock.ShouldCreateAccountMove(m, &srcAcc, nil, cfg) {
		t.Fatalf("expected account move to be created when src has boundary")
	}
	// No boundary → skip
	if stock.ShouldCreateAccountMove(m, nil, nil, cfg) {
		t.Fatalf("expected no account move without boundary")
	}
	// Periodic → skip even with boundary
	periodic := stock.ProductValuationConfig{Valuation: stock.ValuationPeriodic}
	if stock.ShouldCreateAccountMove(m, &srcAcc, &destAcc, periodic) {
		t.Fatalf("expected no account move in periodic mode")
	}
	// Non-valued move → skip
	m2 := m
	m2.IsIn = false
	m2.IsOut = false
	if stock.ShouldCreateAccountMove(m2, &srcAcc, &destAcc, cfg) {
		t.Fatalf("expected no account move for non-valued move")
	}
}

func TestBuildValuationLines(t *testing.T) {
	stockAcc := int64(5)
	srcAcc := int64(8) // e.g. production location boundary
	destAcc := int64(9)

	m := stock.StockMove{Value: 150, ProductQty: 3, QuantityDone: 3, IsOut: true}

	// Source boundary: debit stock valuation, credit source location.
	lines := stock.BuildValuationLines(m, &srcAcc, &destAcc, &stockAcc)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].AccountID != stockAcc || lines[0].Debit != 150 {
		t.Fatalf("expected debit line on stock acc %d = 150, got %+v", stockAcc, lines[0])
	}
	if lines[1].AccountID != srcAcc || lines[1].Credit != 150 {
		t.Fatalf("expected credit line on src acc %d = 150, got %+v", srcAcc, lines[1])
	}

	// No source boundary, dest boundary: debit dest, credit stock.
	lines2 := stock.BuildValuationLines(m, nil, &destAcc, &stockAcc)
	if lines2[0].AccountID != destAcc || lines2[0].Debit != 150 {
		t.Fatalf("expected debit dest acc %d, got %+v", destAcc, lines2[0])
	}
	if lines2[1].AccountID != stockAcc || lines2[1].Credit != 150 {
		t.Fatalf("expected credit stock acc %d, got %+v", stockAcc, lines2[1])
	}

	// Nil stock account → no lines.
	_ = destAcc
	if lines3 := stock.BuildValuationLines(m, &srcAcc, &destAcc, nil); lines3 != nil {
		t.Fatalf("expected nil lines without stock account")
	}
}

func TestAccountingPeriodValidation(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	p := stock.AccountingPeriod{DateFrom: from, DateTo: to, JournalID: 6}
	if err := p.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if p.Name == "" {
		t.Fatalf("expected auto-generated name")
	}
	if p.State != "open" {
		t.Fatalf("expected default state 'open', got %q", p.State)
	}

	bad := stock.AccountingPeriod{DateFrom: to, DateTo: from, JournalID: 6}
	if err := bad.Validate(); err == nil {
		t.Fatalf("expected error for reversed period range")
	}

	noJournal := stock.AccountingPeriod{DateFrom: from, DateTo: to}
	if err := noJournal.Validate(); err == nil {
		t.Fatalf("expected error for missing journal")
	}
}

func TestProductValueValidation(t *testing.T) {
	pv := stock.ProductValue{ProductID: 1, MoveID: ptr64(10), Value: 100}
	if err := pv.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bad := stock.ProductValue{ProductID: 0, MoveID: ptr64(10), Value: 100}
	if err := bad.Validate(); err == nil {
		t.Fatalf("expected error for missing product")
	}

	noRef := stock.ProductValue{ProductID: 1, Value: 100}
	if err := noRef.Validate(); err == nil {
		t.Fatalf("expected error for missing move/lot reference")
	}
}

func ptr64(v int64) *int64 { return &v }