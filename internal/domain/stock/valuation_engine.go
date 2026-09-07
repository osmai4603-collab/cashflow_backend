package stock

import (
	"math"
	"sort"
)

// roundTo4 rounds a monetary value to 4 decimal places (matching accounting domain precision).
func roundTo4(val float64) float64 {
	return math.Round(val*10000) / 10000
}

// ValuedQuantity returns the quantity used for valuation (falls back to ProductQty when done qty is empty).
func (m *StockMove) ValuedQuantity() float64 {
	if m.QuantityDone != 0 {
		return m.QuantityDone
	}
	return m.ProductQty
}

// IsValued reports whether the move carries a valuation (in or out, excluding dropships).
func (m *StockMove) IsValued() bool {
	return m.IsIn || m.IsOut
}

// UnitPrice returns the effective unit valuation price of this move (Value ÷ quantity).
func (m *StockMove) UnitPrice() float64 {
	qty := m.ValuedQuantity()
	if qty <= 0 {
		return 0
	}
	return m.Value / qty
}

// RunFIFO consumes qty from the FIFO stack (oldest layer first), mirroring Odoo's _run_fifo.
// It returns the total value consumed and the remaining stack. When qty exceeds the available
// stock, the last known unit price is extrapolated (Odoo extrapolation behaviour).
func RunFIFO(stack FIFOStack, qty float64) (value float64, remaining FIFOStack) {
	if qty <= 0 {
		return 0, stack
	}
	remainingQty := qty
	consumed := 0.0
	remaining = make(FIFOStack, 0, len(stack))
	for _, entry := range stack {
		if remainingQty <= 0 {
			remaining = append(remaining, entry)
			continue
		}
		if entry.Qty >= remainingQty {
			consumed += remainingQty * (entry.Value / entry.Qty)
			if entry.Qty-remainingQty > 0 {
				remaining = append(remaining, FIFOEntry{
					MoveID: entry.MoveID,
					Qty:    entry.Qty - remainingQty,
					Value:  roundTo4(entry.Value - (entry.Value / entry.Qty * remainingQty)),
				})
			}
			remainingQty = 0
		} else {
			consumed += entry.Value
			remainingQty -= entry.Qty
		}
	}
	// Extrapolate with the last known unit price when the stack is fully consumed.
	if remainingQty > 0 {
		lastPrice := 0.0
		if len(stack) > 0 {
			lastEntry := stack[len(stack)-1]
			if lastEntry.Qty > 0 {
				lastPrice = lastEntry.Value / lastEntry.Qty
			}
		}
		consumed += remainingQty * lastPrice
	}
	return roundTo4(consumed), remaining
}

// RunAverageBatch computes the running average cost of a product from its valued moves
// ordered by (date, id), mirroring Odoo's _run_average_batch. It returns the final avg cost.
// Incoming moves contribute positive quantities; outgoing moves negative. The value of an
// outgoing move is its qty × current avg cost (consumed), matching perpetual average.
func RunAverageBatch(moves []StockMove) float64 {
	type valuedMove struct {
		StockMove
	}
	sorted := make([]valuedMove, 0, len(moves))
	for _, m := range moves {
		sorted = append(sorted, valuedMove{m})
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		if !sorted[i].Date.Equal(sorted[j].Date) {
			return sorted[i].Date.Before(sorted[j].Date)
		}
		return sorted[i].ID < sorted[j].ID
	})

	var quantity, value float64
	var avgCost float64
	for _, vm := range sorted {
		qty := vm.ValuedQuantity()
		if vm.IsIn {
			// incoming: avg cost before increment
			if quantity > 0 {
				avgCost = value / quantity
			}
			value += vm.Value
			quantity += qty
		} else if vm.IsOut {
			// outgoing: consume at current avg cost (value leaves stock)
			if quantity > 0 {
				avgCost = value / quantity
			}
			vmOut := vm.StockMove
			vmOut.Value = roundTo4(avgCost * qty)
			value -= vmOut.Value
			quantity -= qty
		}
		if quantity != 0 {
			avgCost = value / quantity
		} else {
			avgCost = 0
		}
	}
	return roundTo4(avgCost)
}

// ComputeInValue computes the valuation value of an incoming move from its unit cost.
func ComputeInValue(m StockMove, unitCost float64) float64 {
	if unitCost <= 0 {
		return 0
	}
	return roundTo4(unitCost * m.ValuedQuantity())
}

// ComputeOutValue computes the value of an outgoing move according to the cost method.
// For standard/average it uses StandardPrice × qty; for FIFO it consumes the provided
// stack and returns the consumed value along with the remaining stack.
func ComputeOutValue(m StockMove, cfg ProductValuationConfig, stack FIFOStack) (float64, FIFOStack) {
	qty := m.ValuedQuantity()
	switch cfg.CostMethod {
	case CostFIFO:
		value, remaining := RunFIFO(stack, qty)
		return value, remaining
	default: // standard, average
		return roundTo4(m.StandardPrice * qty), stack
	}
}

// ValuationLine is a single accounting leg for a stock-valuation entry (in stock domain terms).
type ValuationLine struct {
	AccountID int64   `json:"account_id"`
	Debit     float64 `json:"debit"`
	Credit    float64 `json:"credit"`
}

// ShouldCreateAccountMove decides whether a journal entry must be generated when a move is done
// (mirrors Odoo _should_create_account_move): real_time valuation, valued move, a valuation
// boundary account must exist on at least one side, and non-zero quantity.
func ShouldCreateAccountMove(m StockMove, srcValuationAccount, destValuationAccount *int64, cfg ProductValuationConfig) bool {
	if cfg.Valuation != ValuationRealTime {
		return false
	}
	if !m.IsValued() {
		return false
	}
	if m.ValuedQuantity() == 0 {
		return false
	}
	return srcValuationAccount != nil || destValuationAccount != nil
}

// BuildValuationLines builds the two accounting legs for a valued move, mirroring Odoo's
// _get_account_move_line_vals:
//
//	if source location has its own valuation account (boundary):
//	    debit  = product stock valuation account
//	    credit = source location valuation account
//	else:
//	    debit  = destination location valuation account (may be 0 for stock-to-stock)
//	    credit = product stock valuation account
//
// It returns nil when no account composition can be derived.
func BuildValuationLines(m StockMove, srcValuationAccount, destValuationAccount, stockValuationAccount *int64) []ValuationLine {
	if stockValuationAccount == nil {
		return nil
	}
	value := math.Abs(m.Value)
	if value == 0 {
		return nil
	}
	if srcValuationAccount != nil {
		// Move out of a valued location (e.g. production / scrap): debit stock, credit location.
		return []ValuationLine{
			{AccountID: *stockValuationAccount, Debit: value, Credit: 0},
			{AccountID: *srcValuationAccount, Debit: 0, Credit: value},
		}
	}
	// Incoming or stock-to-stock: debit destination boundary, credit stock.
	if destValuationAccount == nil {
		// Internal stock-to-stock without a dest boundary → net zero on the stock account.
		return []ValuationLine{
			{AccountID: *stockValuationAccount, Debit: value, Credit: 0},
			{AccountID: *stockValuationAccount, Debit: 0, Credit: value},
		}
	}
	return []ValuationLine{
		{AccountID: *destValuationAccount, Debit: value, Credit: 0},
		{AccountID: *stockValuationAccount, Debit: 0, Credit: value},
	}
}