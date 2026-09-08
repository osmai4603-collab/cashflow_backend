package accounting

import (
	"fmt"
	"math"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// MoveType represents the category of accounting document (account.move in Odoo).
type MoveType string

const (
	MoveTypeEntry      MoveType = "entry"       // Journal Entry (general / adjustment)
	MoveTypeInReceipt  MoveType = "in_receipt"  // Employee expense receipt
	MoveTypeOutInvoice MoveType = "out_invoice" // Customer Invoice
	MoveTypeOutRefund  MoveType = "out_refund"  // Customer Credit Note / Refund
	MoveTypeInInvoice  MoveType = "in_invoice"  // Vendor Bill
	MoveTypeInRefund   MoveType = "in_refund"   // Vendor Credit Note / Refund
)

// MoveState represents the lifecycle status of an account move.
type MoveState string

const (
	MoveStateDraft  MoveState = "draft"
	MoveStatePosted MoveState = "posted"
	MoveStateCancel MoveState = "cancel"
)

// PaymentState indicates whether an invoice is settled.
type PaymentState string

const (
	PaymentStateNotPaid   PaymentState = "not_paid"
	PaymentStateInPayment PaymentState = "in_payment"
	PaymentStatePaid      PaymentState = "paid"
	PaymentStatePartial   PaymentState = "partial"
	PaymentStateReversed  PaymentState = "reversed"
)

// AccountMoveLine represents a single debit/credit line in a journal entry (account.move.line in Odoo).
type AccountMoveLine struct {
	ID        int64   `json:"id"`
	MoveID    int64   `json:"move_id"`
	AccountID int64   `json:"account_id"`
	PartnerID *int64  `json:"partner_id,omitempty"`
	ProductID *int64  `json:"product_id,omitempty"`
	Name      string  `json:"name"` // Label / Description
	Quantity  float64 `json:"quantity"`
	PriceUnit float64 `json:"price_unit"`
	Discount  float64 `json:"discount"` // percentage discount e.g. 10.0 for 10%
	Debit     float64 `json:"debit"`
	Credit    float64 `json:"credit"`
	Balance   float64 `json:"balance"` // debit - credit
	TaxIDs    []int64 `json:"tax_ids,omitempty"`
	TaxAmount float64 `json:"tax_amount"`
	// Reconcile reports whether the underlying account allows reconciliation.
	Reconcile bool `json:"reconcile"`
	// Reconciled flags a line whose outstanding balance has been fully settled.
	Reconciled bool `json:"reconciled"`
	// AmountResidual is the outstanding amount still to be reconciled (always non-negative).
	AmountResidual float64 `json:"amount_residual"`
	// MatchingNumber groups lines participating in the same (full or partial) reconciliation.
	MatchingNumber *string `json:"matching_number,omitempty"`
	// StatementLineID links this line to the bank statement line that created/cleared it.
	StatementLineID *int64 `json:"statement_line_id,omitempty"`
	// DisplayType classifies generated lines ("" default/product, "cogs" for Anglo-Saxon cost lines).
	DisplayType string `json:"display_type,omitempty"`
	// CogsOriginID references the invoiced line that generated a COGS counterpart line.
	CogsOriginID *int64 `json:"cogs_origin_id,omitempty"`
	// IsLandedCostsLine marks cost-allocation journal lines created from landed costs.
	IsLandedCostsLine bool      `json:"is_landed_costs_line,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// IsDebit reports whether the line carries a debit balance.
func (l *AccountMoveLine) IsDebit() bool {
	return l.Debit > l.Credit
}

// AbsBalance returns the absolute value of the line balance.
func (l *AccountMoveLine) AbsBalance() float64 {
	if l.Balance < 0 {
		return -l.Balance
	}
	return l.Balance
}

// ComputeResidual derives the outstanding reconciliation amount from the line balance.
func (l *AccountMoveLine) ComputeResidual() {
	if l.Reconcile && !l.Reconciled {
		l.AmountResidual = roundTo4(math.Max(0, l.AbsBalance()-l.AmountMatched()))
	} else {
		l.AmountResidual = 0
	}
}

// AmountMatched returns the portion of the line balance already settled by reconciliations.
func (l *AccountMoveLine) AmountMatched() float64 {
	if l.Reconcile && l.AmountResidual >= 0 {
		return roundTo4(l.AbsBalance() - l.AmountResidual)
	}
	return 0
}

// MarkReconciled sets/clears the reconciled flag and zeroes the residual accordingly.
func (l *AccountMoveLine) MarkReconciled(reconciled bool) {
	l.Reconciled = reconciled
	if reconciled {
		l.AmountResidual = 0
	}
}

// AccountMove represents a complete journal entry or invoice (account.move in Odoo).
type AccountMove struct {
	ID              int64             `json:"id"`
	Name            string            `json:"name"` // Sequence number e.g. "INV/2026/00001" or "/"
	MoveType        MoveType          `json:"move_type"`
	JournalID       int64             `json:"journal_id"`
	PartnerID       *int64            `json:"partner_id,omitempty"`
	Date            time.Time         `json:"date"` // Accounting date
	InvoiceDate     *time.Time        `json:"invoice_date,omitempty"`
	InvoiceDueDate  *time.Time        `json:"invoice_date_due,omitempty"`
	PaymentTermID   *int64            `json:"payment_term_id,omitempty"`
	State           MoveState         `json:"state"`
	PaymentState    PaymentState      `json:"payment_state"`
	AmountUntaxed   float64           `json:"amount_untaxed"`
	AmountTax       float64           `json:"amount_tax"`
	AmountTotal     float64           `json:"amount_total"`
	AmountResidual  float64           `json:"amount_residual"` // Outstanding balance
	Currency        string            `json:"currency"`
	Ref             string            `json:"ref,omitempty"`
	ReversedEntryID *int64            `json:"reversed_entry_id,omitempty"`
	Lines           []AccountMoveLine `json:"lines,omitempty"`
	// EDI Fields (ZATCA / Odoo compliant)
	IsSimplified  bool         `json:"is_simplified"`
	IsSelfBilling bool         `json:"is_self_billing"`
	IsThirdParty  bool         `json:"is_third_party"`
	Active        bool         `json:"active"`
	Audit         audit.Fields `json:"audit"`
}

// IsInvoice returns true if this move is a customer or vendor invoice/refund.
func (m *AccountMove) IsInvoice() bool {
	return m.MoveType == MoveTypeOutInvoice ||
		m.MoveType == MoveTypeOutRefund ||
		m.MoveType == MoveTypeInInvoice ||
		m.MoveType == MoveTypeInRefund
}

// TotalDebit calculates the sum of all debits across lines.
func (m *AccountMove) TotalDebit() float64 {
	var total float64
	for _, line := range m.Lines {
		total += line.Debit
	}
	return roundTo4(total)
}

// TotalCredit calculates the sum of all credits across lines.
func (m *AccountMove) TotalCredit() float64 {
	var total float64
	for _, line := range m.Lines {
		total += line.Credit
	}
	return roundTo4(total)
}

// ValidateBalance asserts that SUM(debit) == SUM(credit) within financial rounding precision.
func (m *AccountMove) ValidateBalance() error {
	if len(m.Lines) < 2 {
		return platformerrors.Validation("unbalanced journal entry", map[string]string{
			"lines": "a valid accounting entry must have at least 2 lines",
		})
	}

	debit := m.TotalDebit()
	credit := m.TotalCredit()
	diff := math.Abs(debit - credit)

	// Financial tolerance: 1 cent / penny (0.01)
	if diff > 0.01 {
		return platformerrors.Validation("unbalanced journal entry", map[string]string{
			"balance": fmt.Sprintf("total debit (%.4f) does not equal total credit (%.4f), diff: %.4f", debit, credit, diff),
		})
	}

	return nil
}

// Validate verifies AccountMove constraints.
func (m *AccountMove) Validate() error {
	if m.JournalID <= 0 {
		return platformerrors.Validation("journal is required", map[string]string{
			"journal_id": "must reference a valid journal",
		})
	}

	if m.Date.IsZero() {
		return platformerrors.Validation("accounting date is required", map[string]string{
			"date": "cannot be empty",
		})
	}

	switch m.MoveType {
	case MoveTypeEntry, MoveTypeInReceipt, MoveTypeOutInvoice, MoveTypeOutRefund, MoveTypeInInvoice, MoveTypeInRefund:
	default:
		return platformerrors.Validation("invalid move type", map[string]string{
			"move_type": fmt.Sprintf("unsupported move type '%s'", m.MoveType),
		})
	}

	if m.State == "" {
		m.State = MoveStateDraft
	}

	if m.PaymentState == "" {
		m.PaymentState = PaymentStateNotPaid
	}

	if m.Currency == "" {
		m.Currency = "USD"
	}

	if m.Name == "" {
		m.Name = "/"
	}

	// In posted state, balance MUST be valid
	if m.State == MoveStatePosted {
		if err := m.ValidateBalance(); err != nil {
			return err
		}
	}

	// Validate individual lines
	for i, l := range m.Lines {
		if l.AccountID <= 0 {
			return platformerrors.Validation("invalid line account", map[string]string{
				fmt.Sprintf("lines[%d].account_id", i): "must reference a valid account",
			})
		}
		if l.Debit < 0 || l.Credit < 0 {
			return platformerrors.Validation("negative debit or credit is forbidden", map[string]string{
				fmt.Sprintf("lines[%d]", i): "debit and credit must be non-negative",
			})
		}
		if l.Debit > 0 && l.Credit > 0 {
			return platformerrors.Validation("line cannot have both debit and credit", map[string]string{
				fmt.Sprintf("lines[%d]", i): "a line must be either debit or credit",
			})
		}
	}

	return nil
}

// Post transitions the move from draft to posted with a permanent sequence number.
func (m *AccountMove) Post(sequence string) error {
	if m.State == MoveStatePosted {
		return platformerrors.Conflict("move is already posted")
	}
	if m.State == MoveStateCancel {
		return platformerrors.Conflict("cannot post a cancelled move; reset to draft first")
	}

	if err := m.ValidateBalance(); err != nil {
		return err
	}

	m.State = MoveStatePosted
	if sequence != "" {
		m.Name = sequence
	}
	return nil
}

// Cancel transitions the move to cancelled status.
func (m *AccountMove) Cancel() error {
	if m.State == MoveStateCancel {
		return nil
	}
	m.State = MoveStateCancel
	return nil
}

// CreateReverseMove generates an opposite reversal entry (Storno / Credit Note).
func (m *AccountMove) CreateReverseMove(reversalDate time.Time, ref string) *AccountMove {
	if reversalDate.IsZero() {
		reversalDate = time.Now().UTC()
	}

	revType := MoveTypeEntry
	if m.MoveType == MoveTypeOutInvoice {
		revType = MoveTypeOutRefund
	} else if m.MoveType == MoveTypeInInvoice {
		revType = MoveTypeInRefund
	}

	rev := &AccountMove{
		Name:            "/",
		MoveType:        revType,
		JournalID:       m.JournalID,
		PartnerID:       m.PartnerID,
		Date:            reversalDate,
		State:           MoveStateDraft,
		PaymentState:    PaymentStateNotPaid,
		Currency:        m.Currency,
		Ref:             ref,
		ReversedEntryID: &m.ID,
		Active:          true,
	}

	if rev.Ref == "" {
		rev.Ref = fmt.Sprintf("Reversal of %s", m.Name)
	}

	// Swap debits and credits
	for _, l := range m.Lines {
		rev.Lines = append(rev.Lines, AccountMoveLine{
			AccountID: l.AccountID,
			PartnerID: l.PartnerID,
			ProductID: l.ProductID,
			Name:      fmt.Sprintf("Reversal: %s", l.Name),
			Quantity:  l.Quantity,
			PriceUnit: l.PriceUnit,
			Discount:  l.Discount,
			Debit:     l.Credit,
			Credit:    l.Debit,
			Balance:   l.Credit - l.Debit,
			TaxIDs:    l.TaxIDs,
			TaxAmount: l.TaxAmount,
		})
	}

	rev.AmountUntaxed = m.AmountUntaxed
	rev.AmountTax = m.AmountTax
	rev.AmountTotal = m.AmountTotal
	rev.AmountResidual = m.AmountTotal

	return rev
}
