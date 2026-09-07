package accountingusecase

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"cashflow_backend/internal/domain/accounting"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// ─────────────────────────────────────────────────────────────────────────────
// Input DTOs
// ─────────────────────────────────────────────────────────────────────────────

type JournalEntryLineInput struct {
	AccountID        int64   `json:"account_id"`
	PartnerID        *int64  `json:"partner_id"`
	ProductID        *int64  `json:"product_id"`
	Name             string  `json:"name"`
	Debit            float64 `json:"debit"`
	Credit           float64 `json:"credit"`
	StatementLineID  *int64  `json:"statement_line_id,omitempty"`
}

type CreateJournalEntryInput struct {
	JournalID int64                   `json:"journal_id"`
	Date      time.Time               `json:"date"`
	Ref       string                  `json:"ref"`
	Lines     []JournalEntryLineInput `json:"lines"`
}

type InvoiceLineItemInput struct {
	ProductID *int64  `json:"product_id"`
	AccountID *int64  `json:"account_id"`
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	PriceUnit float64 `json:"price_unit"`
	Discount  float64 `json:"discount"` // percentage e.g. 10.0 for 10%
	TaxIDs    []int64 `json:"tax_ids"`
	// CogsAmount books a cost-of-goods-sold counterpart on this line (Anglo-Saxon).
	// When > 0 an additional pair of display_type='cogs' lines is generated:
	//   debit  → COGS account (12 / 500000 by default)
	//   credit → Stock/Inventory account (5 / 140000 by default)
	CogsAmount      float64 `json:"cogs_amount,omitempty"`
	CogsAccountID   *int64  `json:"cogs_account_id,omitempty"`
	StockAccountID  *int64  `json:"stock_account_id,omitempty"`
}

type CreateInvoiceInput struct {
	MoveType      accounting.MoveType    `json:"move_type"` // out_invoice, in_invoice, out_refund, in_refund
	PartnerID     int64                  `json:"partner_id"`
	JournalID     int64                  `json:"journal_id"`
	Date          time.Time              `json:"date"`
	InvoiceDate   *time.Time             `json:"invoice_date"`
	PaymentTermID *int64                 `json:"payment_term_id"`
	Currency      string                 `json:"currency"`
	Ref           string                 `json:"ref"`
	Items         []InvoiceLineItemInput `json:"items"`
}

type UpdateMoveInput struct {
	Date          *time.Time              `json:"date"`
	InvoiceDate   *time.Time              `json:"invoice_date"`
	PaymentTermID *int64                  `json:"payment_term_id"`
	Ref           *string                 `json:"ref"`
	Lines         []JournalEntryLineInput `json:"lines"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Moves & Invoices Implementation
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateJournalEntry(ctx context.Context, in CreateJournalEntryInput) (*accounting.AccountMove, error) {
	if in.JournalID <= 0 {
		return nil, platformerrors.Validation("journal is required", map[string]string{
			"journal_id": "cannot be empty",
		})
	}
	if in.Date.IsZero() {
		in.Date = time.Now().UTC()
	}

	move := &accounting.AccountMove{
		Name:         "/",
		MoveType:     accounting.MoveTypeEntry,
		JournalID:    in.JournalID,
		Date:         in.Date,
		State:        accounting.MoveStateDraft,
		PaymentState: accounting.PaymentStateNotPaid,
		Currency:     "USD",
		Ref:          in.Ref,
		Lines:        make([]accounting.AccountMoveLine, len(in.Lines)),
	}

	for i, l := range in.Lines {
		move.Lines[i] = accounting.AccountMoveLine{
			AccountID:      l.AccountID,
			PartnerID:      l.PartnerID,
			ProductID:      l.ProductID,
			Name:           strings.TrimSpace(l.Name),
			Debit:          roundTo4(l.Debit),
			Credit:         roundTo4(l.Credit),
			Balance:        roundTo4(l.Debit - l.Credit),
			StatementLineID: l.StatementLineID,
		}
	}

	move.AmountTotal = move.TotalDebit()

	if err := move.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreateMove(ctx, move); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "journal entry created", "id", move.ID, "lines", len(move.Lines))
	return move, nil
}

func (uc *UseCase) CreateInvoice(ctx context.Context, in CreateInvoiceInput) (*accounting.AccountMove, error) {
	if in.PartnerID <= 0 {
		return nil, platformerrors.Validation("partner is required for invoices", map[string]string{
			"partner_id": "must reference a valid partner",
		})
	}
	if in.JournalID <= 0 {
		return nil, platformerrors.Validation("journal is required", map[string]string{
			"journal_id": "cannot be empty",
		})
	}
	if len(in.Items) == 0 {
		return nil, platformerrors.Validation("invoice items are required", map[string]string{
			"items": "must contain at least one line item",
		})
	}

	journal, err := uc.repo.GetJournalByID(ctx, in.JournalID)
	if err != nil {
		return nil, err
	}

	if in.Date.IsZero() {
		in.Date = time.Now().UTC()
	}
	if in.InvoiceDate == nil {
		in.InvoiceDate = &in.Date
	}
	if in.Currency == "" {
		in.Currency = "USD"
	}

	var invoiceDueDate *time.Time
	if in.PaymentTermID != nil && *in.PaymentTermID > 0 {
		term, err := uc.repo.GetPaymentTermByID(ctx, *in.PaymentTermID)
		if err != nil {
			return nil, err
		}
		due := term.ComputeDueDate(*in.InvoiceDate)
		invoiceDueDate = &due
	} else {
		invoiceDueDate = in.InvoiceDate
	}

	// Determine counterpart partner account (Receivable for Customer, Payable for Vendor)
	var partnerAccountID int64
	isCustomerInvoice := in.MoveType == accounting.MoveTypeOutInvoice || in.MoveType == accounting.MoveTypeOutRefund
	if isCustomerInvoice {
		// Accounts Receivable
		arAcc, err := uc.repo.GetAccountByCode(ctx, "120000")
		if err != nil {
			partnerAccountID = 3 // Standard Seed ID
		} else {
			partnerAccountID = arAcc.ID
		}
	} else {
		// Accounts Payable
		apAcc, err := uc.repo.GetAccountByCode(ctx, "210000")
		if err != nil {
			partnerAccountID = 6 // Standard Seed ID
		} else {
			partnerAccountID = apAcc.ID
		}
	}

	// Calculate line items, taxes, and generate double-entry lines
	var (
		totalUntaxed float64
		totalTax     float64
		moveLines    []accounting.AccountMoveLine
		taxMap       = make(map[int64]float64) // taxAccountID -> amount
	)

	for _, item := range in.Items {
		qty := item.Quantity
		if qty <= 0 {
			qty = 1.0
		}
		discountFactor := 1.0 - (item.Discount / 100.0)
		if discountFactor < 0 {
			discountFactor = 0
		}
		baseLineTotal := roundTo4(qty * item.PriceUnit * discountFactor)

		lineUntaxed := baseLineTotal
		lineTax := 0.0

		// Compute taxes
		for _, taxID := range item.TaxIDs {
			tax, err := uc.repo.GetTaxByID(ctx, taxID)
			if err != nil {
				return nil, err
			}
			res := tax.Compute(baseLineTotal)
			lineUntaxed = res.UntaxedAmount
			lineTax += res.TaxAmount
			taxMap[tax.AccountID] += res.TaxAmount
		}

		totalUntaxed += lineUntaxed
		totalTax += lineTax

		// Determine line revenue/expense account
		accID := item.AccountID
		if accID == nil || *accID <= 0 {
			if journal.DefaultAccountID != nil && *journal.DefaultAccountID > 0 {
				accID = journal.DefaultAccountID
			} else if isCustomerInvoice {
				accID = int64Ptr(10) // Product Sales Revenue
			} else {
				accID = int64Ptr(12) // Cost of Goods Sold / Expense
			}
		}

		// Create item line
		// For customer invoice (out_invoice): Sales Revenue is Credit
		// For vendor bill (in_invoice): Expense is Debit
		var debit, credit float64
		if isCustomerInvoice {
			credit = lineUntaxed
		} else {
			debit = lineUntaxed
		}

		moveLines = append(moveLines, accounting.AccountMoveLine{
			AccountID: *accID,
			PartnerID: &in.PartnerID,
			ProductID: item.ProductID,
			Name:      strings.TrimSpace(item.Name),
			Quantity:  qty,
			PriceUnit: item.PriceUnit,
			Discount:  item.Discount,
			Debit:     debit,
			Credit:    credit,
			Balance:   debit - credit,
			TaxIDs:    item.TaxIDs,
			TaxAmount: lineTax,
		})
	}

	// Create Tax Lines
	for taxAccID, taxAmt := range taxMap {
		if taxAmt <= 0 {
			continue
		}
		var debit, credit float64
		if isCustomerInvoice {
			credit = roundTo4(taxAmt)
		} else {
			debit = roundTo4(taxAmt)
		}
		moveLines = append(moveLines, accounting.AccountMoveLine{
			AccountID: taxAccID,
			PartnerID: &in.PartnerID,
			Name:      "Tax Line",
			Quantity:  1.0,
			PriceUnit: taxAmt,
			Debit:     debit,
			Credit:    credit,
			Balance:   debit - credit,
			TaxAmount: taxAmt,
		})
	}

	totalAmount := roundTo4(totalUntaxed + totalTax)

	// Create Partner Counterpart Line (Receivable / Payable)
	var partnerDebit, partnerCredit float64
	if isCustomerInvoice {
		partnerDebit = totalAmount
	} else {
		partnerCredit = totalAmount
	}

	counterpartLine := accounting.AccountMoveLine{
		AccountID: partnerAccountID,
		PartnerID: &in.PartnerID,
		Name:      fmt.Sprintf("Partner: %s", in.Ref),
		Quantity:  1.0,
		PriceUnit: totalAmount,
		Debit:     partnerDebit,
		Credit:    partnerCredit,
		Balance:   partnerDebit - partnerCredit,
	}
	// Insert counterpart at the beginning for Odoo convention
	moveLines = append([]accounting.AccountMoveLine{counterpartLine}, moveLines...)

	// Anglo-Saxon: book COGS counterpart lines for items that delivered inventory.
	type cogsPair struct {
		itemIdx  int // index in in.Items (line index = itemIdx + 1 in moveLines)
		pairIdx  int // index of the first line of the generated pair
	}
	var cogsPairs []cogsPair
	cogsAccount := int64(12) // 500000 Cost of Goods Sold
	stockAccount := int64(5) // 140000 Inventory / Stock Valuation
	for i, item := range in.Items {
		if item.CogsAmount <= 0 {
			continue
		}
		cogs := roundTo4(item.CogsAmount)
		acc := cogsAccount
		if item.CogsAccountID != nil && *item.CogsAccountID > 0 {
			acc = *item.CogsAccountID
		}
		stockAcc := stockAccount
		if item.StockAccountID != nil && *item.StockAccountID > 0 {
			stockAcc = *item.StockAccountID
		}
		pairIdx := len(moveLines)
		// Debit COGS (cost leaves stock), credit Inventory.
		moveLines = append(moveLines,
			accounting.AccountMoveLine{
				AccountID:   acc,
				PartnerID:   &in.PartnerID,
				ProductID:   item.ProductID,
				Name:        fmt.Sprintf("COGS: %s", strings.TrimSpace(item.Name)),
				Quantity:    item.Quantity,
				Debit:       cogs,
				Credit:      0,
				Balance:     cogs,
				DisplayType: "cogs",
			},
			accounting.AccountMoveLine{
				AccountID:   stockAcc,
				PartnerID:   &in.PartnerID,
				ProductID:   item.ProductID,
				Name:        fmt.Sprintf("Inventory Out: %s", strings.TrimSpace(item.Name)),
				Quantity:    item.Quantity,
				Debit:       0,
				Credit:      cogs,
				Balance:     -cogs,
				DisplayType: "cogs",
			},
		)
		cogsPairs = append(cogsPairs, cogsPair{itemIdx: i, pairIdx: pairIdx})
	}

	move := &accounting.AccountMove{
		Name:           "/",
		MoveType:       in.MoveType,
		JournalID:      in.JournalID,
		PartnerID:      &in.PartnerID,
		Date:           in.Date,
		InvoiceDate:    in.InvoiceDate,
		InvoiceDueDate: invoiceDueDate,
		PaymentTermID:  in.PaymentTermID,
		State:          accounting.MoveStateDraft,
		PaymentState:   accounting.PaymentStateNotPaid,
		AmountUntaxed:  roundTo4(totalUntaxed),
		AmountTax:      roundTo4(totalTax),
		AmountTotal:    totalAmount,
		AmountResidual: totalAmount,
		Currency:       in.Currency,
		Ref:            in.Ref,
		Lines:          moveLines,
	}

	if err := move.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.CreateMove(ctx, move); err != nil {
		return nil, err
	}

	// Link each COGS line back to the originating invoice line (cogs_origin_id).
	if len(cogsPairs) > 0 {
		for _, cp := range cogsPairs {
			itemLineID := move.Lines[cp.itemIdx+1].ID
			move.Lines[cp.pairIdx].CogsOriginID = &itemLineID
			move.Lines[cp.pairIdx+1].CogsOriginID = &itemLineID
		}
		if err := uc.repo.UpdateMove(ctx, move); err != nil {
			return nil, err
		}
	}

	uc.logger.InfoContext(ctx, "invoice created", "id", move.ID, "type", move.MoveType, "total", move.AmountTotal)
	return move, nil
}

func (uc *UseCase) GetMove(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	return uc.repo.GetMoveWithLines(ctx, id)
}

func (uc *UseCase) UpdateMove(ctx context.Context, id int64, in UpdateMoveInput) (*accounting.AccountMove, error) {
	move, err := uc.repo.GetMoveWithLines(ctx, id)
	if err != nil {
		return nil, err
	}

	if move.State != accounting.MoveStateDraft {
		return nil, platformerrors.Conflict("only draft moves can be edited")
	}

	if in.Date != nil {
		move.Date = *in.Date
	}
	if in.InvoiceDate != nil {
		move.InvoiceDate = in.InvoiceDate
	}
	if in.PaymentTermID != nil {
		move.PaymentTermID = in.PaymentTermID
	}
	if in.Ref != nil {
		move.Ref = *in.Ref
	}

	if in.Lines != nil {
		lines := make([]accounting.AccountMoveLine, len(in.Lines))
		for i, l := range in.Lines {
			lines[i] = accounting.AccountMoveLine{
				AccountID: l.AccountID,
				PartnerID: l.PartnerID,
				ProductID: l.ProductID,
				Name:      strings.TrimSpace(l.Name),
				Debit:     roundTo4(l.Debit),
				Credit:    roundTo4(l.Credit),
				Balance:   roundTo4(l.Debit - l.Credit),
			}
		}
		move.Lines = lines
		move.AmountTotal = move.TotalDebit()
		move.AmountResidual = move.AmountTotal
	}

	if err := move.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateMove(ctx, move); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "move updated", "id", move.ID)
	return move, nil
}

func (uc *UseCase) DeleteMove(ctx context.Context, id int64) error {
	move, err := uc.repo.GetMoveByID(ctx, id)
	if err != nil {
		return err
	}
	if move.State != accounting.MoveStateDraft {
		return platformerrors.Conflict("cannot delete posted or cancelled move")
	}
	if err := uc.repo.DeleteMove(ctx, id); err != nil {
		return err
	}
	uc.logger.InfoContext(ctx, "move deleted", "id", id)
	return nil
}

func (uc *UseCase) PostMove(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	move, err := uc.repo.GetMoveWithLines(ctx, id)
	if err != nil {
		return nil, err
	}

	if move.State == accounting.MoveStatePosted {
		return move, nil
	}
	if move.State == accounting.MoveStateCancel {
		return nil, platformerrors.Conflict("cannot post a cancelled move")
	}

	// Validate balanced entry
	if err := move.ValidateBalance(); err != nil {
		return nil, err
	}

	// Generate sequence
	seqYear := move.Date.Year()
	seq, err := uc.repo.GetNextSequence(ctx, move.JournalID, seqYear)
	if err != nil {
		return nil, err
	}

	if err := move.Post(seq); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateMove(ctx, move); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "move posted", "id", move.ID, "sequence", move.Name)
	return move, nil
}

func (uc *UseCase) CancelMove(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	move, err := uc.repo.GetMoveWithLines(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := move.Cancel(); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateMove(ctx, move); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "move cancelled", "id", move.ID)
	return move, nil
}

func (uc *UseCase) ReverseMove(ctx context.Context, id int64, reversalDate time.Time, ref string) (*accounting.AccountMove, error) {
	move, err := uc.repo.GetMoveWithLines(ctx, id)
	if err != nil {
		return nil, err
	}

	if move.State != accounting.MoveStatePosted {
		return nil, platformerrors.Conflict("only posted moves can be reversed")
	}

	rev := move.CreateReverseMove(reversalDate, ref)
	if err := uc.repo.CreateMove(ctx, rev); err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "move reversed", "original_id", move.ID, "reversal_id", rev.ID)
	return rev, nil
}

func (uc *UseCase) ListMoves(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[accounting.AccountMove], error) {
	return uc.repo.ListMoves(ctx, f, page)
}

// UpdatePaymentStatus updates the residual amount and settlement payment state on a move (typically invoked during payment reconciliation).
func (uc *UseCase) UpdatePaymentStatus(ctx context.Context, id int64, state accounting.PaymentState, residual float64) error {
	move, err := uc.repo.GetMoveWithLines(ctx, id)
	if err != nil {
		return err
	}
	move.PaymentState = state
	move.AmountResidual = roundTo4(residual)
	return uc.repo.UpdateMove(ctx, move)
}

// GetMoveLine fetches a single account move line.
func (uc *UseCase) GetMoveLine(ctx context.Context, id int64) (*accounting.AccountMoveLine, error) {
	return uc.repo.GetMoveLineByID(ctx, id)
}

// UpdateMoveLineReconcile updates the reconciliation flags of a single move line.
func (uc *UseCase) UpdateMoveLineReconcile(ctx context.Context, id int64, reconciled bool, residual float64, matchingNumber *string) error {
	return uc.repo.UpdateMoveLineReconcile(ctx, id, reconciled, residual, matchingNumber)
}

// ListReconcilableMoveLines returns posted, reconcilable, unmatched move lines, optionally filtered
// by partner and excluding the given line ids. Used to surface candidates for a statement line.
func (uc *UseCase) ListReconcilableMoveLines(ctx context.Context, partnerID *int64, excludeLineIDs []int64, limit int) ([]accounting.AccountMoveLine, error) {
	return uc.repo.ListReconcilableMoveLines(ctx, partnerID, excludeLineIDs, limit)
}

func roundTo4(val float64) float64 {
	return math.Round(val*10000) / 10000
}

func int64Ptr(v int64) *int64 {
	return &v
}
