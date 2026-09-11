package stockusecase

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"cashflow_backend/internal/domain/product"
	"cashflow_backend/internal/domain/stock"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

// StockAccountService generates automatic double-entry accounting records for stock operations.
// Corresponds to Odoo 19 addons/stock_account.
type StockAccountService struct {
	stockRepo     stock.Repository
	accountingSvc AccountingGateway
	productRepo   product.Repository
	logger        *slog.Logger
}

// NewStockAccountService creates a new instance of StockAccountService.
func NewStockAccountService(
	stockRepo stock.Repository,
	accountingSvc AccountingGateway,
	productRepo product.Repository,
	logger *slog.Logger,
) *StockAccountService {
	if logger == nil {
		logger = slog.Default()
	}
	return &StockAccountService{
		stockRepo:     stockRepo,
		accountingSvc: accountingSvc,
		productRepo:   productRepo,
		logger:        logger,
	}
}

// OnPickingValidated implements StockIntegrationHook.
func (s *StockAccountService) OnPickingValidated(ctx context.Context, picking *stock.StockPicking) error {
	return s.CreateValuationEntries(ctx, picking)
}

// OnPickingCancelled implements StockIntegrationHook.
func (s *StockAccountService) OnPickingCancelled(ctx context.Context, picking *stock.StockPicking) error {
	s.logger.InfoContext(ctx, "stock picking cancelled, checking for valuation reversals", "picking_id", picking.ID)
	return nil
}

// CreateValuationEntries generates the corresponding journal entries for validated stock moves.
//
// Accounting Logic:
//
//	Incoming (Vendor -> Internal):
//	  Debit:  Stock Valuation Account (Inventory Asset, e.g. #140000 / id 5)
//	  Credit: Stock Interim Received Account (Goods Received Not Invoiced, e.g. #140100 / id 16)
//
//	Outgoing (Internal -> Customer):
//	  Debit:  Stock Interim Delivered / COGS Account (e.g. #500000 / id 12)
//	  Credit: Stock Valuation Account (Inventory Asset, e.g. #140000 / id 5)
//
//	Internal (Internal -> Internal):
//	  Net zero, no financial valuation movement required.
func (s *StockAccountService) CreateValuationEntries(ctx context.Context, picking *stock.StockPicking) error {
	if picking == nil || len(picking.Moves) == 0 {
		return nil
	}

	for i := range picking.Moves {
		move := &picking.Moves[i]

		// Idempotency: skip moves that already generated an accounting entry
		if move.AccountMoveID != nil && *move.AccountMoveID > 0 {
			continue
		}

		srcLoc, err := s.stockRepo.GetLocationByID(ctx, move.LocationID)
		if err != nil {
			s.logger.WarnContext(ctx, "could not load source location for move valuation", "move_id", move.ID, "error", err)
			continue
		}
		destLoc, err := s.stockRepo.GetLocationByID(ctx, move.LocationDestID)
		if err != nil {
			s.logger.WarnContext(ctx, "could not load dest location for move valuation", "move_id", move.ID, "error", err)
			continue
		}

		isIncoming := srcLoc.Usage == stock.LocationUsageSupplier && destLoc.Usage == stock.LocationUsageInternal
		isOutgoing := srcLoc.Usage == stock.LocationUsageInternal && destLoc.Usage == stock.LocationUsageCustomer
		isDropship := srcLoc.Usage == stock.LocationUsageSupplier && destLoc.Usage == stock.LocationUsageCustomer

		if !isIncoming && !isOutgoing && !isDropship {
			// Internal transfers or non-valued locations produce no general ledger impact
			continue
		}

		// Calculate valuation value
		amount := math.Abs(move.Value)
		if amount == 0 {
			qty := move.QuantityDone
			if qty <= 0 {
				qty = move.ProductQty
			}
			unitCost := move.StandardPrice
			if unitCost <= 0 && s.productRepo != nil {
				if pt, ptErr := s.productRepo.GetTemplateByID(ctx, move.ProductID); ptErr == nil && pt != nil {
					if pt.CostPrice > 0 {
						unitCost = pt.CostPrice
					} else if pt.AvgCost > 0 {
						unitCost = pt.AvgCost
					}
				}
			}
			amount = math.Round(qty*unitCost*10000) / 10000
		}

		if amount <= 0 {
			s.logger.DebugContext(ctx, "skipping valuation entry for zero-value move", "move_id", move.ID)
			continue
		}

		// Resolve accounts
		var (
			valuationAccountID int64 = 5  // Default: 140000 Inventory
			inputAccountID     int64 = 16 // Default: 140100 Stock Interim Received
			outputAccountID    int64 = 12 // Default: 500000 COGS
			journalID          int64 = 6  // Default: STJ Stock Operations
		)

		if s.productRepo != nil {
			if pt, ptErr := s.productRepo.GetTemplateByID(ctx, move.ProductID); ptErr == nil && pt != nil {
				if pt.StockValuationAccountID != nil && *pt.StockValuationAccountID > 0 {
					valuationAccountID = *pt.StockValuationAccountID
				}
				if pt.StockJournalID != nil && *pt.StockJournalID > 0 {
					journalID = *pt.StockJournalID
				}
			}
		}

		var debitAccountID, creditAccountID int64
		var moveRef string

		if isIncoming {
			debitAccountID = valuationAccountID
			creditAccountID = inputAccountID
			moveRef = fmt.Sprintf("Stock Valuation (Receipt): %s / %s", picking.Name, move.Name)
		} else if isOutgoing {
			debitAccountID = outputAccountID
			creditAccountID = valuationAccountID
			moveRef = fmt.Sprintf("Stock Valuation (Delivery): %s / %s", picking.Name, move.Name)
		} else { // dropship
			debitAccountID = outputAccountID
			creditAccountID = inputAccountID
			moveRef = fmt.Sprintf("Stock Valuation (Dropship): %s / %s", picking.Name, move.Name)
		}

		entryLines := []accountingusecase.JournalEntryLineInput{
			{
				AccountID: debitAccountID,
				ProductID: &move.ProductID,
				Name:      fmt.Sprintf("%s / %s", picking.Name, move.Name),
				Debit:     amount,
				Credit:    0,
			},
			{
				AccountID: creditAccountID,
				ProductID: &move.ProductID,
				Name:      fmt.Sprintf("%s / %s", picking.Name, move.Name),
				Debit:     0,
				Credit:    amount,
			},
		}

		moveDate := move.Date
		if moveDate.IsZero() && picking.DateDone != nil {
			moveDate = *picking.DateDone
		}
		if moveDate.IsZero() {
			moveDate = picking.ScheduledDate
		}

		entry, err := s.accountingSvc.CreateJournalEntry(ctx, accountingusecase.CreateJournalEntryInput{
			JournalID: journalID,
			Date:      moveDate,
			Ref:       moveRef,
			Lines:     entryLines,
		})
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to create stock valuation journal entry", "move_id", move.ID, "error", err)
			return err
		}

		if _, err := s.accountingSvc.PostMove(ctx, entry.ID); err != nil {
			s.logger.ErrorContext(ctx, "failed to post stock valuation journal entry", "entry_id", entry.ID, "error", err)
			return err
		}

		move.AccountMoveID = &entry.ID
		move.Value = amount
		if err := s.stockRepo.UpdateMoveValue(ctx, move); err != nil {
			s.logger.WarnContext(ctx, "failed to update stock move account_move_id", "move_id", move.ID, "error", err)
		}

		s.logger.InfoContext(ctx, "stock valuation entry created & posted",
			"picking_id", picking.ID,
			"move_id", move.ID,
			"account_move_id", entry.ID,
			"amount", amount,
		)
	}

	return nil
}
