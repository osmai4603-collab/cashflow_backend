package stockusecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/stock"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

// ─────────────────────────────────────────────────────────────────────────────
// Input DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateLandedCostLineInput struct {
	Name        string            `json:"name"`
	ProductID   int64             `json:"product_id"`
	AccountID   int64             `json:"account_id"`
	PriceUnit   float64           `json:"price_unit"`
	SplitMethod stock.SplitMethod `json:"split_method"`
}

type CreateLandedCostInput struct {
	PickingIDs   []int64                     `json:"picking_ids"`
	Date         time.Time                   `json:"date"`
	JournalID    int64                       `json:"journal_id"`
	VendorBillID *int64                      `json:"vendor_bill_id,omitempty"`
	Description  string                      `json:"description,omitempty"`
	CompanyID    int64                       `json:"company_id"`
	CostLines    []CreateLandedCostLineInput `json:"cost_lines"`
}

type UpdateLandedCostInput struct {
	PickingIDs  *[]int64                     `json:"picking_ids,omitempty"`
	Date        *time.Time                   `json:"date,omitempty"`
	Description *string                      `json:"description,omitempty"`
	CostLines   *[]CreateLandedCostLineInput `json:"cost_lines,omitempty"`
}

type CreateLandedCostFromBillInput struct {
	VendorBillID int64   `json:"vendor_bill_id"`
	PickingIDs   []int64 `json:"picking_ids"`
	Description  string  `json:"description,omitempty"`
	CompanyID    int64   `json:"company_id"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Landed Cost CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateLandedCost(ctx context.Context, in CreateLandedCostInput) (*stock.LandedCost, error) {
	if uc.logger == nil {
		uc.logger = slog.Default()
	}
	journalID := in.JournalID
	if journalID <= 0 {
		resolved, err := uc.resolveLandedCostJournal(ctx, &stock.LandedCost{CompanyID: in.CompanyID})
		if err != nil {
			return nil, err
		}
		journalID = resolved
	}
	lc, err := buildLandedCost(in.PickingIDs, in.Date, journalID, in.VendorBillID, in.Description, in.CompanyID, in.CostLines)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.CreateLandedCost(ctx, lc); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "stock landed cost created", "id", lc.ID, "name", lc.Name, "amount_total", lc.AmountTotal)
	return lc, nil
}

func (uc *UseCase) GetLandedCost(ctx context.Context, id int64) (*stock.LandedCost, error) {
	return uc.repo.GetLandedCostByID(ctx, id)
}

func (uc *UseCase) ListLandedCosts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.LandedCost], error) {
	return uc.repo.ListLandedCosts(ctx, f, page)
}

func (uc *UseCase) DeleteLandedCost(ctx context.Context, id int64) error {
	lc, err := uc.repo.GetLandedCostByID(ctx, id)
	if err != nil {
		return err
	}
	if lc.State != stock.LandedCostDraft && lc.State != stock.LandedCostCancel {
		return platformerrors.Conflict("only draft or cancelled landed costs can be deleted")
	}
	return uc.repo.DeleteLandedCost(ctx, id)
}

func (uc *UseCase) UpdateLandedCost(ctx context.Context, id int64, in UpdateLandedCostInput) (*stock.LandedCost, error) {
	lc, err := uc.repo.GetLandedCostByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lc.State != stock.LandedCostDraft {
		return nil, platformerrors.Conflict("only draft landed costs can be edited")
	}
	if in.PickingIDs != nil {
		lc.PickingIDs = *in.PickingIDs
	}
	if in.Date != nil {
		lc.Date = *in.Date
	}
	if in.Description != nil {
		lc.Description = *in.Description
	}
	if in.CostLines != nil {
		lines := make([]stock.LandedCostLine, 0, len(*in.CostLines))
		for _, l := range *in.CostLines {
			lines = append(lines, stock.LandedCostLine{
				Name:        l.Name,
				ProductID:   l.ProductID,
				AccountID:   l.AccountID,
				PriceUnit:   l.PriceUnit,
				SplitMethod: l.SplitMethod,
			})
		}
		lc.CostLines = lines
	}
	lc.ValuationAdjustments = nil
	if err := validateLandedCost(lc); err != nil {
		return nil, err
	}
	lc.ComputeTotalAmount()
	if err := uc.repo.UpdateLandedCost(ctx, lc); err != nil {
		return nil, err
	}
	return lc, nil
}

func (uc *UseCase) CancelLandedCost(ctx context.Context, id int64) (*stock.LandedCost, error) {
	lc, err := uc.repo.GetLandedCostByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lc.State != stock.LandedCostDraft {
		return nil, platformerrors.Conflict(fmt.Sprintf("cannot cancel landed cost in state '%s'", lc.State))
	}
	lc.State = stock.LandedCostCancel
	if err := uc.repo.UpdateLandedCost(ctx, lc); err != nil {
		return nil, err
	}
	return lc, nil
}

// CreateLandedCostFromVendorBill drafts a landed cost whose cost lines are derived
// from the *billable* product lines of an existing vendor bill (Odoo purchase →
// stock_landed_costs "Create Bill" flow).
func (uc *UseCase) CreateLandedCostFromVendorBill(ctx context.Context, in CreateLandedCostFromBillInput) (*stock.LandedCost, error) {
	if uc.logger == nil {
		uc.logger = slog.Default()
	}
	if uc.accountingSvc == nil {
		uc.accountingSvc = noopAccountingGateway{}
	}
	if in.VendorBillID <= 0 {
		return nil, platformerrors.Validation("vendor_bill_id is required", nil)
	}
	bill, err := uc.accountingSvc.GetMove(ctx, in.VendorBillID)
	if err != nil {
		return nil, err
	}
	if bill == nil || bill.ID <= 0 {
		return nil, platformerrors.NotFound(fmt.Sprintf("vendor bill #%d not found", in.VendorBillID))
	}
	if bill.MoveType != accounting.MoveTypeInInvoice {
		return nil, platformerrors.Conflict("selected move is not a vendor bill")
	}
	if uc.productRepo == nil {
		return nil, platformerrors.Conflict("product repository is not wired")
	}

	var lines []stock.LandedCostLine
	for _, l := range bill.Lines {
		if l.ProductID == nil {
			continue
		}
		pt, err := uc.productRepo.GetTemplateByID(ctx, *l.ProductID)
		if err != nil || !pt.LandedCostOK {
			continue
		}
		amount := l.Balance
		if amount == 0 {
			amount = l.Debit + l.Credit
		}
		if amount < 0 {
			amount = -amount
		}
		if amount == 0 {
			continue
		}
		accountID := int64(17) // 510000 Price Difference (fallback)
		if pt.PriceDifferenceAccountID != nil && *pt.PriceDifferenceAccountID > 0 {
			accountID = *pt.PriceDifferenceAccountID
		}
		split := pt.SplitMethodLandedCost
		if split == "" {
			split = stock.SplitEqual
		}
		lines = append(lines, stock.LandedCostLine{
			Name:        string(pt.Name),
			ProductID:   pt.ID,
			AccountID:   accountID,
			PriceUnit:   amount,
			SplitMethod: split,
		})
	}
	if len(lines) == 0 {
		return nil, platformerrors.Validation("no billable landed-cost products found on this vendor bill", nil)
	}

	companyID := in.CompanyID
	if companyID <= 0 {
		companyID = 1
	}
	journalID := bill.JournalID
	if uc.companyRepo != nil && companyID > 0 {
		resolved, err := uc.resolveLandedCostJournal(ctx, &stock.LandedCost{CompanyID: companyID})
		if err != nil {
			return nil, err
		}
		journalID = resolved
	} else if journalID <= 0 {
		resolved, err := uc.resolveLandedCostJournal(ctx, &stock.LandedCost{CompanyID: companyID})
		if err != nil {
			return nil, err
		}
		journalID = resolved
	}
	billID := bill.ID
	lc, err := buildLandedCost(in.PickingIDs, time.Now().UTC(), journalID, &billID, in.Description, companyID, toLineInputs(lines))
	if err != nil {
		return nil, err
	}
	if err := uc.repo.CreateLandedCost(ctx, lc); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "landed cost drafted from vendor bill", "id", lc.ID, "bill", billID)
	return lc, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Compute (reallocate) & Validate (post)
// ─────────────────────────────────────────────────────────────────────────────

// ComputeLandedCost reallocates every cost line across the eligible received moves
// and persists the valuation adjustments (Odoo stock_landed_cost.compute_landed_cost).
func (uc *UseCase) ComputeLandedCost(ctx context.Context, id int64) (*stock.LandedCost, error) {
	if uc.logger == nil {
		uc.logger = slog.Default()
	}
	lc, err := uc.repo.GetLandedCostByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lc.State != stock.LandedCostDraft {
		return nil, platformerrors.Conflict(fmt.Sprintf("cannot compute landed cost in state '%s'", lc.State))
	}
	if uc.productRepo == nil {
		return nil, platformerrors.Conflict("product repository is not wired")
	}
	if lc.JournalID <= 0 {
		journal, err := uc.resolveLandedCostJournal(ctx, lc)
		if err != nil {
			return nil, err
		}
		lc.JournalID = journal
	}

	type target struct {
		move       *stock.StockMove
		qty        float64
		weight     float64
		volume     float64
		formerCost float64
		totalCost  float64
	}
	var targets []target

	for _, pid := range lc.PickingIDs {
		moves, err := uc.repo.GetMovesByPickingID(ctx, pid)
		if err != nil {
			return nil, err
		}
		for i := range moves {
			m := &moves[i]
			if m.State != stock.MoveStateDone || m.ProductQty <= 0 {
				continue
			}
			pt, err := uc.productRepo.GetTemplateByID(ctx, m.ProductID)
			if err != nil || !pt.LandedCostOK {
				continue
			}
			if pt.CostMethod != stock.CostFIFO && pt.CostMethod != stock.CostAverage {
				continue
			}
			qty := m.QuantityDone
			if qty <= 0 {
				qty = m.ProductQty
			}
			totalCost := m.Value
			if totalCost == 0 {
				totalCost = m.StandardPrice * qty
			}
			targets = append(targets, target{
				move:       m,
				qty:        qty,
				weight:     pt.Weight * qty,
				volume:     pt.Volume * qty,
				formerCost: m.StandardPrice,
				totalCost:  totalCost,
			})
		}
	}
	if len(targets) == 0 {
		return nil, platformerrors.Validation("no eligible received moves found for the selected receipts", nil)
	}

	var totalQty, totalWeight, totalVolume, totalCost float64
	for _, t := range targets {
		totalQty += t.qty
		totalWeight += t.weight
		totalVolume += t.volume
		totalCost += t.totalCost
	}

	lc.ValuationAdjustments = nil
	for _, cl := range lc.CostLines {
		allocs := make([]float64, len(targets))
		var sum float64
		for i, t := range targets {
			v := stock.ComputeSplitValue(
				cl.SplitMethod, cl.PriceUnit,
				totalQty, totalWeight, totalVolume, totalCost, float64(len(targets)),
				t.qty, t.weight, t.volume, t.formerCost,
			)
			v = round4(v)
			allocs[i] = v
			sum += v
		}
		// Absorb the HALF-UP rounding drift on the last move so the cost line books out exactly.
		if len(allocs) > 0 {
			allocs[len(allocs)-1] = round4(allocs[len(allocs)-1] + (cl.PriceUnit - sum))
		}
		for i, t := range targets {
			perUnit := 0.0
			if t.qty != 0 {
				perUnit = round4(allocs[i] / t.qty)
			}
			adj := stock.ValuationAdjustment{
				LandedCostID:     lc.ID,
				CostLineID:       cl.ID,
				MoveID:           t.move.ID,
				ProductID:        t.move.ProductID,
				Quantity:         t.qty,
				Weight:           t.weight,
				Volume:           t.volume,
				FormerCost:       t.formerCost,
				AdditionalCost:   perUnit,
				MoveRemainingQty: t.move.RemainingQty,
			}
			adj.ComputeFinalCost()
			lc.ValuationAdjustments = append(lc.ValuationAdjustments, adj)
		}
	}

	if err := uc.repo.UpdateLandedCost(ctx, lc); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "stock landed cost computed", "id", lc.ID, "adjustments", len(lc.ValuationAdjustments))
	return lc, nil
}

// ValidateLandedCost posts the valuation journal entry for the computed adjustments
// and revalues the affected moves (Odoo stock_landed_cost.action_validate).
func (uc *UseCase) ValidateLandedCost(ctx context.Context, id int64) (*stock.LandedCost, error) {
	if uc.logger == nil {
		uc.logger = slog.Default()
	}
	lc, err := uc.repo.GetLandedCostByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lc.State != stock.LandedCostDraft {
		return nil, platformerrors.Conflict(fmt.Sprintf("cannot validate landed cost in state '%s'", lc.State))
	}
	if len(lc.ValuationAdjustments) == 0 {
		lc, err = uc.ComputeLandedCost(ctx, id)
		if err != nil {
			return nil, err
		}
	}

	var entryLines []accountingusecase.JournalEntryLineInput
	type moveDelta struct {
		moveID            int64
		amount            float64 // additional value applied to remaining qty (+ = increase)
		perUnitAdditional float64
		finalCost         float64
	}
	perMove := map[int64]*moveDelta{}

	for _, adj := range lc.ValuationAdjustments {
		remainingQty := adj.MoveRemainingQty
		if remainingQty == 0 && adj.Quantity > 0 {
			remainingQty = adj.Quantity
		}
		amount := stock.LandedCostJournalAmount(adj.AdditionalCost, adj.Quantity, remainingQty)
		if amount == 0 {
			continue
		}
		stockAccount := uc.stockValuationAccountForProduct(ctx, adj.ProductID)
		expenseAccount := uc.costLineAccount(lc, adj.CostLineID)
		name := fmt.Sprintf("%s / move %d", lc.Name, adj.MoveID)
		if amount > 0 {
			entryLines = append(entryLines,
				accountingusecase.JournalEntryLineInput{AccountID: stockAccount, ProductID: &adj.ProductID, Name: name, Debit: amount, Credit: 0, IsLandedCostsLine: true},
				accountingusecase.JournalEntryLineInput{AccountID: expenseAccount, Name: name, Debit: 0, Credit: amount, IsLandedCostsLine: true},
			)
		} else {
			entryLines = append(entryLines,
				accountingusecase.JournalEntryLineInput{AccountID: stockAccount, ProductID: &adj.ProductID, Name: name, Debit: 0, Credit: -amount, IsLandedCostsLine: true},
				accountingusecase.JournalEntryLineInput{AccountID: expenseAccount, Name: name, Debit: -amount, Credit: 0, IsLandedCostsLine: true},
			)
		}

		if delta, ok := perMove[adj.MoveID]; ok {
			delta.amount = round4(delta.amount + amount)
			delta.perUnitAdditional = round4(delta.perUnitAdditional + adj.AdditionalCost)
		} else {
			perMove[adj.MoveID] = &moveDelta{
				moveID:            adj.MoveID,
				amount:            amount,
				perUnitAdditional: adj.AdditionalCost,
				finalCost:         adj.FinalCost,
			}
		}
	}

	// Create the valuation entry even for zero-amount leftovers (Odoo always posts).
	if len(entryLines) == 0 {
		// nothing is remaining in stock: fully consumed by deliveries → no book entry needed.
		uc.logger.InfoContext(ctx, "landed cost has no remaining stock to value; skipping journal entry", "id", lc.ID)
	} else {
		entry, err := uc.accountingSvc.CreateJournalEntry(ctx, accountingusecase.CreateJournalEntryInput{
			JournalID: lc.JournalID,
			Date:      lc.Date,
			Ref:       lc.Name,
			Lines:     entryLines,
		})
		if err != nil {
			return nil, err
		}
		posted, err := uc.accountingSvc.PostMove(ctx, entry.ID)
		if err != nil {
			return nil, err
		}
		if posted != nil && posted.ID > 0 {
			lc.AccountMoveID = &posted.ID
		} else if entry.ID > 0 {
			lc.AccountMoveID = &entry.ID
		}
	}

	// Revalue each affected move: grow remaining_value, refresh unit cost, link the entry.
	for _, delta := range perMove {
		move, err := uc.repo.GetMoveByID(ctx, delta.moveID)
		if err != nil {
			return nil, err
		}
		move.RemainingValue = round4(move.RemainingValue + delta.amount)
		if delta.finalCost > 0 {
			move.StandardPrice = delta.finalCost
		} else {
			move.StandardPrice = round4(move.StandardPrice + delta.perUnitAdditional)
		}
		if lc.AccountMoveID != nil {
			move.AccountMoveID = lc.AccountMoveID
		}
		if err := uc.repo.UpdateMoveValue(ctx, move); err != nil {
			return nil, err
		}
	}

	lc.State = stock.LandedCostDone
	if err := uc.repo.UpdateLandedCost(ctx, lc); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "stock landed cost validated", "id", lc.ID, "account_move_id", lc.AccountMoveID)
	return lc, nil
}

// costLineAccount resolves the expense account of a cost line (fallback to Price Difference).
func (uc *UseCase) costLineAccount(lc *stock.LandedCost, costLineID int64) int64 {
	for _, cl := range lc.CostLines {
		if cl.ID == costLineID {
			return cl.AccountID
		}
	}
	return 17 // 510000 Price Difference fallback
}

// stockValuationAccountForProduct resolves the product's stock valuation account
// (product → fallback to the Inventory account).
func (uc *UseCase) stockValuationAccountForProduct(ctx context.Context, productID int64) int64 {
	if uc.productRepo != nil {
		if pt, err := uc.productRepo.GetTemplateByID(ctx, productID); err == nil {
			if pt.StockValuationAccountID != nil && *pt.StockValuationAccountID > 0 {
				return *pt.StockValuationAccountID
			}
		}
	}
	return 5 // 140000 Inventory
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) resolveLandedCostJournal(ctx context.Context, lc *stock.LandedCost) (int64, error) {
	if lc != nil && lc.JournalID > 0 {
		return lc.JournalID, nil
	}
	if uc.companyRepo != nil {
		companyID := lc.CompanyID
		if companyID <= 0 {
			companyID = 1
		}
		if c, err := uc.companyRepo.GetByID(ctx, companyID); err == nil && c != nil && c.LandedCostJournalID != nil && *c.LandedCostJournalID > 0 {
			return *c.LandedCostJournalID, nil
		}
	}
	return 6, nil
}

func buildLandedCost(pickingIDs []int64, date time.Time, journalID int64, vendorBillID *int64, description string, companyID int64, lineInputs []CreateLandedCostLineInput) (*stock.LandedCost, error) {
	if date.IsZero() {
		date = time.Now().UTC()
	}
	if journalID <= 0 {
		journalID = 6 // STJ — Stock Operations
	}
	if companyID <= 0 {
		companyID = 1
	}
	lc := &stock.LandedCost{
		PickingIDs:   pickingIDs,
		Date:         date,
		State:        stock.LandedCostDraft,
		JournalID:    journalID,
		VendorBillID: vendorBillID,
		Description:  description,
		CompanyID:    companyID,
	}
	for _, l := range lineInputs {
		lc.CostLines = append(lc.CostLines, stock.LandedCostLine{
			Name:        l.Name,
			ProductID:   l.ProductID,
			AccountID:   l.AccountID,
			PriceUnit:   l.PriceUnit,
			SplitMethod: l.SplitMethod,
		})
	}
	if err := validateLandedCost(lc); err != nil {
		return nil, err
	}
	lc.ComputeTotalAmount()
	return lc, nil
}

func validateLandedCost(lc *stock.LandedCost) error {
	return lc.Validate()
}

func toLineInputs(lines []stock.LandedCostLine) []CreateLandedCostLineInput {
	out := make([]CreateLandedCostLineInput, 0, len(lines))
	for _, l := range lines {
		out = append(out, CreateLandedCostLineInput{
			Name:        l.Name,
			ProductID:   l.ProductID,
			AccountID:   l.AccountID,
			PriceUnit:   l.PriceUnit,
			SplitMethod: l.SplitMethod,
		})
	}
	return out
}
