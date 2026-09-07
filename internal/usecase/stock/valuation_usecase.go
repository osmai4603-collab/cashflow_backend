package stockusecase

import (
	"context"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/stock"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

// AccountingGateway narrows the accounting operations needed by stock valuation.
type AccountingGateway interface {
	CreateJournalEntry(ctx context.Context, in accountingusecase.CreateJournalEntryInput) (*accounting.AccountMove, error)
	PostMove(ctx context.Context, id int64) (*accounting.AccountMove, error)
	GetMove(ctx context.Context, id int64) (*accounting.AccountMove, error)
}

// noopAccountingGateway is a safe default when no accounting service is wired in.
type noopAccountingGateway struct{}

func (noopAccountingGateway) CreateJournalEntry(ctx context.Context, in accountingusecase.CreateJournalEntryInput) (*accounting.AccountMove, error) {
	return &accounting.AccountMove{ID: 0}, nil
}

func (noopAccountingGateway) PostMove(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	return &accounting.AccountMove{ID: id}, nil
}

func (noopAccountingGateway) GetMove(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	return nil, platformerrors.Conflict("accounting service is not wired")
}

// ─────────────────────────────────────────────────────────────────────────────
// Input DTOs
// ─────────────────────────────────────────────────────────────────────────────

type AdjustMoveValueInput struct {
	MoveID      int64   `json:"move_id"`
	Value       float64 `json:"value"`
	Description string  `json:"description"`
	UserID      int64   `json:"user_id"`
	CompanyID   int64   `json:"company_id"`
}

type ClosePeriodValuationInput struct {
	PeriodID int64 `json:"period_id"`
}

type CreateAccountingPeriodInput struct {
	Name      string    `json:"name"`
	DateFrom  time.Time `json:"date_from"`
	DateTo    time.Time `json:"date_to"`
	JournalID int64     `json:"journal_id"`
	CompanyID int64     `json:"company_id"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Valuation (Phase 12 — stock-account integration)
// ─────────────────────────────────────────────────────────────────────────────

// ResolveValuationConfig resolves the valuation settings for a move's product,
// falling back to company defaults (standard / real_time).
func (uc *UseCase) ResolveValuationConfig(ctx context.Context, move *stock.StockMove) (stock.ProductValuationConfig, error) {
	cfg := stock.ProductValuationConfig{
		CostMethod: stock.CostStandard,
		Valuation:  stock.ValuationRealTime,
	}
	if uc.productRepo == nil {
		return cfg, nil
	}
	pt, err := uc.productRepo.GetTemplateByID(ctx, move.ProductID)
	if err != nil {
		return cfg, err
	}
	if pt.CostMethod != "" {
		cfg.CostMethod = pt.CostMethod
	}
	if pt.Valuation != "" {
		cfg.Valuation = pt.Valuation
	}
	cfg.LotValuated = pt.LotValuated
	cfg.StockValuationAccountID = pt.StockValuationAccountID
	cfg.PriceDifferenceAccountID = pt.PriceDifferenceAccountID
	cfg.StockJournalID = pt.StockJournalID
	return cfg, nil
}

// companyIDForPicking resolves the picking's company for FIFO stack scoping.
func (uc *UseCase) companyIDForPicking(ctx context.Context, move *stock.StockMove) int64 {
	if move.PickingID != nil {
		if p, err := uc.repo.GetPickingByID(ctx, *move.PickingID); err == nil && p.CompanyID != nil {
			return *p.CompanyID
		}
	}
	return 1
}

// doneValuedMoves returns the done, valued moves of a product in engine order.
func (uc *UseCase) doneValuedMoves(ctx context.Context, productID int64) ([]stock.StockMove, error) {
	f := filter.NewFilter().
		Add("product_id", filter.OpEqual, productID).
		Add("state", filter.OpEqual, string(stock.MoveStateDone))
	res, err := uc.repo.ListMoves(ctx, f, pagination.PageRequest{Page: 1, Limit: 100})
	if err != nil {
		return nil, err
	}
	var valued []stock.StockMove
	for _, m := range res.Items {
		if m.IsIn || m.IsOut {
			valued = append(valued, m)
		}
	}
	return valued, nil
}

// syncProductValuation refreshes the product's avg_cost and total_value.
func (uc *UseCase) syncProductValuation(ctx context.Context, move *stock.StockMove, cfg stock.ProductValuationConfig) {
	if uc.productRepo == nil {
		return
	}
	pt, err := uc.productRepo.GetTemplateByID(ctx, move.ProductID)
	if err != nil {
		return
	}
	switch cfg.CostMethod {
	case stock.CostFIFO:
		if stack, err := uc.repo.GetFIFOStack(ctx, move.ProductID, uc.companyIDForPicking(ctx, move)); err == nil {
			pt.AvgCost = stack.UnitPrice()
			pt.TotalValue = stack.TotalValue()
		}
	case stock.CostStandard:
		pt.AvgCost = pt.CostPrice
	default: // average
		if moves, err := uc.doneValuedMoves(ctx, move.ProductID); err == nil && len(moves) > 0 {
			pt.AvgCost = stock.RunAverageBatch(moves)
		}
	}
	if err := uc.productRepo.UpdateTemplate(ctx, pt); err != nil {
		uc.logger.WarnContext(ctx, "failed to sync product valuation", "product_id", move.ProductID, "error", err)
	}
}

// valuateMove assigns the valuation fields of a single move (is_in/is_out/dropship,
// unit cost, value, FIFO stack consumption) mirroring Odoo's _action_done valuation.
func (uc *UseCase) valuateMove(ctx context.Context, move *stock.StockMove, cfg stock.ProductValuationConfig) error {
	srcLoc, err := uc.repo.GetLocationByID(ctx, move.LocationID)
	if err != nil {
		return err
	}
	destLoc, err := uc.repo.GetLocationByID(ctx, move.LocationDestID)
	if err != nil {
		return err
	}

	qty := move.ValuedQuantity()
	move.IsIn = destLoc.Usage == stock.LocationUsageInternal
	move.IsOut = srcLoc.Usage == stock.LocationUsageInternal
	// Direct supplier → customer moves are valued manually, not automatically.
	if srcLoc.Usage == stock.LocationUsageSupplier && destLoc.Usage == stock.LocationUsageCustomer {
		move.IsDropship = true
		move.IsIn = false
		move.IsOut = false
	}

	// Unit cost captured at valuation time.
	if uc.productRepo != nil {
		if pt, err := uc.productRepo.GetTemplateByID(ctx, move.ProductID); err == nil {
			move.StandardPrice = pt.CostPrice
		}
	}

	if move.IsDropship || (!move.IsIn && !move.IsOut) {
		if move.Value == 0 {
			move.Value = 0
		}
		return uc.repo.UpdateMoveValue(ctx, move)
	}

	if move.IsIn {
		if move.ValueManual != nil && *move.ValueManual > 0 {
			move.Value = *move.ValueManual
		} else {
			unitCost := move.StandardPrice
			if cfg.CostMethod == stock.CostAverage && uc.productRepo != nil {
				if pt, err := uc.productRepo.GetTemplateByID(ctx, move.ProductID); err == nil && pt.AvgCost > 0 {
					unitCost = pt.AvgCost
				}
			}
			move.Value = stock.ComputeInValue(*move, unitCost)
		}
		move.RemainingQty = qty
		move.RemainingValue = move.Value
	}

	if move.IsOut {
		if cfg.CostMethod == stock.CostFIFO {
			stack, stackErr := uc.repo.GetFIFOStack(ctx, move.ProductID, uc.companyIDForPicking(ctx, move))
			if stackErr == nil {
				value, remaining := stock.ComputeOutValue(*move, cfg, stack)
				move.Value = value
				for _, layer := range remaining {
					inMove, err := uc.repo.GetMoveByID(ctx, layer.MoveID)
					if err == nil {
						inMove.RemainingQty = layer.Qty
						inMove.RemainingValue = layer.Value
						_ = uc.repo.UpdateMoveValue(ctx, inMove)
					}
				}
			} else {
				value, _ := stock.ComputeOutValue(*move, cfg, nil)
				move.Value = value
			}
		} else {
			value, _ := stock.ComputeOutValue(*move, cfg, nil)
			move.Value = value
		}
		move.RemainingQty = 0
		move.RemainingValue = 0
	}

	if err := uc.repo.UpdateMoveValue(ctx, move); err != nil {
		return err
	}

	// Keep the product's current average cost / total value fresh.
	uc.syncProductValuation(ctx, move, cfg)

	return nil
}

// ValuatePicking values all moves of a validated picking and posts their accounting
// entries when the valuation boundary is configured (mirrors Odoo _action_done).
func (uc *UseCase) ValuatePicking(ctx context.Context, picking *stock.StockPicking) error {
	for i := range picking.Moves {
		move := &picking.Moves[i]
		cfg, err := uc.ResolveValuationConfig(ctx, move)
		if err != nil {
			return err
		}
		if err := uc.valuateMove(ctx, move, cfg); err != nil {
			return err
		}

		// Post entry when the valuation boundary exists on at least one location.
		srcLoc, _ := uc.repo.GetLocationByID(ctx, move.LocationID)
		destLoc, _ := uc.repo.GetLocationByID(ctx, move.LocationDestID)
		var srcVA, destVA *int64
		if srcLoc != nil {
			srcVA = srcLoc.ValuationAccountID
		}
		if destLoc != nil {
			destVA = destLoc.ValuationAccountID
		}
		if !stock.ShouldCreateAccountMove(*move, srcVA, destVA, cfg) {
			continue
		}

		lines := stock.BuildValuationLines(*move, srcVA, destVA, cfg.StockValuationAccountID)
		if len(lines) == 0 {
			continue
		}

		journalID := int64(6) // STJ — Stock Operations
		if cfg.StockJournalID != nil && *cfg.StockJournalID > 0 {
			journalID = *cfg.StockJournalID
		}

		entryLines := make([]accountingusecase.JournalEntryLineInput, 0, len(lines))
		for _, l := range lines {
			entryLines = append(entryLines, accountingusecase.JournalEntryLineInput{
				AccountID: l.AccountID,
				ProductID: &move.ProductID,
				Name:      fmt.Sprintf("%s / %s", picking.Name, move.Name),
				Debit:     l.Debit,
				Credit:    l.Credit,
			})
		}

		entry, err := uc.accountingSvc.CreateJournalEntry(ctx, accountingusecase.CreateJournalEntryInput{
			JournalID: journalID,
			Date:      move.Date,
			Ref:       fmt.Sprintf("Stock Valuation: %s", picking.Name),
			Lines:     entryLines,
		})
		if err != nil {
			return err
		}
		if _, err := uc.accountingSvc.PostMove(ctx, entry.ID); err != nil {
			return err
		}

		move.AccountMoveID = &entry.ID
		if err := uc.repo.UpdateMoveValue(ctx, move); err != nil {
			return err
		}
		uc.logger.InfoContext(ctx, "stock valuation entry posted", "move_id", move.ID, "account_move_id", entry.ID)
	}
	return nil
}

// AdjustMoveValue applies a manual valuation override on a move and records a history entry.
func (uc *UseCase) AdjustMoveValue(ctx context.Context, in AdjustMoveValueInput) (*stock.StockMove, error) {
	move, err := uc.repo.GetMoveByID(ctx, in.MoveID)
	if err != nil {
		return nil, err
	}
	if in.Value < 0 {
		return nil, platformerrors.Validation("value cannot be negative", nil)
	}

	v := in.Value
	move.ValueManual = &v
	move.Value = in.Value

	if err := uc.repo.UpdateMoveValue(ctx, move); err != nil {
		return nil, err
	}

	pv := &stock.ProductValue{
		ProductID:   move.ProductID,
		MoveID:      &move.ID,
		Value:       in.Value,
		CompanyID:   in.CompanyID,
		Date:        time.Now().UTC(),
		UserID:      in.UserID,
		Description: in.Description,
	}
	if pv.CompanyID == 0 {
		pv.CompanyID = 1
	}
	if pv.UserID == 0 {
		pv.UserID = 1
	}
	if err := pv.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.CreateProductValue(ctx, pv); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "stock move value adjusted", "move_id", move.ID, "value", in.Value)
	return move, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Periodic Valuation & Queries
// ─────────────────────────────────────────────────────────────────────────────

// CreateAccountingPeriod opens a new closing period for periodic valuation.
func (uc *UseCase) CreateAccountingPeriod(ctx context.Context, in CreateAccountingPeriodInput) (*stock.AccountingPeriod, error) {
	p := &stock.AccountingPeriod{
		Name:      in.Name,
		DateFrom:  in.DateFrom,
		DateTo:    in.DateTo,
		State:     "open",
		JournalID: in.JournalID,
		CompanyID: in.CompanyID,
	}
	if p.JournalID == 0 {
		p.JournalID = 6 // STJ — Stock Operations
	}
	if p.CompanyID == 0 {
		p.CompanyID = 1
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.CreateAccountingPeriod(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// ClosePeriodValuation closes a period and posts a single closing entry for the
// period's net stock variation.
func (uc *UseCase) ClosePeriodValuation(ctx context.Context, in ClosePeriodValuationInput) (*stock.AccountingPeriod, error) {
	p, err := uc.repo.GetAccountingPeriodByID(ctx, in.PeriodID)
	if err != nil {
		return nil, err
	}
	if p.State == "closed" {
		return nil, platformerrors.Conflict("accounting period is already closed")
	}

	summaries, err := uc.repo.ComputeTotalValuation(ctx, nil, nil)
	if err != nil {
		return nil, err
	}
	var total float64
	for _, s := range summaries {
		total += s.Value
	}
	if total == 0 {
		if err := uc.repo.CloseAccountingPeriod(ctx, p); err != nil {
			return nil, err
		}
		uc.logger.InfoContext(ctx, "accounting period closed (no valuation to post)", "id", p.ID)
		return p, nil
	}

	stockVarAccount := int64(16) // 140100 Stock Variation
	stockAccount := int64(5)     // 140000 Inventory

	entry, err := uc.accountingSvc.CreateJournalEntry(ctx, accountingusecase.CreateJournalEntryInput{
		JournalID: p.JournalID,
		Date:      p.DateTo,
		Ref:       fmt.Sprintf("Periodic Stock Valuation: %s", p.Name),
		Lines: []accountingusecase.JournalEntryLineInput{
			{AccountID: stockVarAccount, Name: "Periodic stock variation", Debit: total, Credit: 0},
			{AccountID: stockAccount, Name: "Inventory closing", Debit: 0, Credit: total},
		},
	})
	if err != nil {
		return nil, err
	}
	if _, err := uc.accountingSvc.PostMove(ctx, entry.ID); err != nil {
		return nil, err
	}

	p.AccountMoveID = &entry.ID
	if err := uc.repo.CloseAccountingPeriod(ctx, p); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "accounting period closed with valuation", "id", p.ID, "total", total)
	return p, nil
}

// GetValuationSummaries returns the current valuation snapshot across products/locations.
func (uc *UseCase) GetValuationSummaries(ctx context.Context, productID *int64, locationID *int64) ([]stock.ValuationSummary, error) {
	return uc.repo.ComputeTotalValuation(ctx, productID, locationID)
}

// ListProductValues returns the manual valuation history.
func (uc *UseCase) ListProductValues(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.ProductValue], error) {
	return uc.repo.ListProductValues(ctx, f, page)
}

// GetAccountingPeriod fetches a single closing period.
func (uc *UseCase) GetAccountingPeriod(ctx context.Context, id int64) (*stock.AccountingPeriod, error) {
	return uc.repo.GetAccountingPeriodByID(ctx, id)
}

// ListAccountingPeriods returns a paginated list of closing periods.
func (uc *UseCase) ListAccountingPeriods(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.AccountingPeriod], error) {
	return uc.repo.ListAccountingPeriods(ctx, f, page)
}
