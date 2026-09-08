package accountinghttp

import (
	"time"

	"cashflow_backend/internal/domain/accounting"
	accountingusecase "cashflow_backend/internal/usecase/accounting"
)

// ─────────────────────────────────────────────────────────────────────────────
// Chart of Accounts DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateAccountRequest struct {
	Code      string                 `json:"code"`
	Name      string                 `json:"name"`
	Type      accounting.AccountType `json:"type"`
	Reconcile bool                   `json:"reconcile"`
	Currency  string                 `json:"currency"`
	ParentID  *int64                 `json:"parent_id"`
	CompanyID *int64                 `json:"company_id"`
}

func (r CreateAccountRequest) ToInput() accountingusecase.CreateAccountInput {
	return accountingusecase.CreateAccountInput{
		Code:      r.Code,
		Name:      r.Name,
		Type:      r.Type,
		Reconcile: r.Reconcile,
		Currency:  r.Currency,
		ParentID:  r.ParentID,
		CompanyID: r.CompanyID,
	}
}

type UpdateAccountRequest struct {
	Code      *string                 `json:"code"`
	Name      *string                 `json:"name"`
	Type      *accounting.AccountType `json:"type"`
	Reconcile *bool                   `json:"reconcile"`
	Currency  *string                 `json:"currency"`
	ParentID  *int64                  `json:"parent_id"`
	CompanyID *int64                  `json:"company_id"`
}

func (r UpdateAccountRequest) ToInput() accountingusecase.UpdateAccountInput {
	return accountingusecase.UpdateAccountInput{
		Code:      r.Code,
		Name:      r.Name,
		Type:      r.Type,
		Reconcile: r.Reconcile,
		Currency:  r.Currency,
		ParentID:  r.ParentID,
		CompanyID: r.CompanyID,
	}
}

type AccountResponse struct {
	ID        int64                  `json:"id"`
	Code      string                 `json:"code"`
	Name      string                 `json:"name"`
	Type      accounting.AccountType `json:"type"`
	Reconcile bool                   `json:"reconcile"`
	Currency  string                 `json:"currency"`
	ParentID  *int64                 `json:"parent_id,omitempty"`
	CompanyID *int64                 `json:"company_id,omitempty"`
	Active    bool                   `json:"active"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

func ToAccountResponse(a *accounting.Account) AccountResponse {
	return AccountResponse{
		ID:        a.ID,
		Code:      a.Code,
		Name:      string(a.Name),
		Type:      a.Type,
		Reconcile: a.Reconcile,
		Currency:  a.Currency,
		ParentID:  a.ParentID,
		CompanyID: a.CompanyID,
		Active:    a.Active,
		CreatedAt: a.Audit.CreatedAt,
		UpdatedAt: a.Audit.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Journals DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateJournalRequest struct {
	Name              string                 `json:"name"`
	Code              string                 `json:"code"`
	Type              accounting.JournalType `json:"type"`
	DefaultAccountID  *int64                 `json:"default_account_id"`
	SuspenseAccountID *int64                 `json:"suspense_account_id"`
	SequencePrefix    string                 `json:"sequence_prefix"`
	NextNumber        int                    `json:"next_number"`
}

func (r CreateJournalRequest) ToInput() accountingusecase.CreateJournalInput {
	return accountingusecase.CreateJournalInput{
		Name:              r.Name,
		Code:              r.Code,
		Type:              r.Type,
		DefaultAccountID:  r.DefaultAccountID,
		SuspenseAccountID: r.SuspenseAccountID,
		SequencePrefix:    r.SequencePrefix,
		NextNumber:        r.NextNumber,
	}
}

type UpdateJournalRequest struct {
	Name              *string                 `json:"name"`
	Code              *string                 `json:"code"`
	Type              *accounting.JournalType `json:"type"`
	DefaultAccountID  *int64                  `json:"default_account_id"`
	SuspenseAccountID *int64                  `json:"suspense_account_id"`
	SequencePrefix    *string                 `json:"sequence_prefix"`
}

func (r UpdateJournalRequest) ToInput() accountingusecase.UpdateJournalInput {
	return accountingusecase.UpdateJournalInput{
		Name:              r.Name,
		Code:              r.Code,
		Type:              r.Type,
		DefaultAccountID:  r.DefaultAccountID,
		SuspenseAccountID: r.SuspenseAccountID,
		SequencePrefix:    r.SequencePrefix,
	}
}

type JournalResponse struct {
	ID                int64                  `json:"id"`
	Name              string                 `json:"name"`
	Code              string                 `json:"code"`
	Type              accounting.JournalType `json:"type"`
	DefaultAccountID  *int64                 `json:"default_account_id,omitempty"`
	SuspenseAccountID *int64                 `json:"suspense_account_id,omitempty"`
	SequencePrefix    string                 `json:"sequence_prefix"`
	NextNumber        int                    `json:"next_number"`
	Active            bool                   `json:"active"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

func ToJournalResponse(j *accounting.Journal) JournalResponse {
	return JournalResponse{
		ID:                j.ID,
				Name:              string(j.Name),
		Code:              j.Code,
		Type:              j.Type,
		DefaultAccountID:  j.DefaultAccountID,
		SuspenseAccountID: j.SuspenseAccountID,
		SequencePrefix:    j.SequencePrefix,
		NextNumber:        j.NextNumber,
		Active:            j.Active,
		CreatedAt:         j.CreatedAt,
		UpdatedAt:         j.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Taxes DTOs
// ─────────────────────────────────────────────────────────────────────────────

type CreateTaxRequest struct {
	Name            string              `json:"name"`
	Type            accounting.TaxType  `json:"type"`
	TypeTaxUse      accounting.TaxScope `json:"type_tax_use"`
	Amount          float64             `json:"amount"`
	AccountID       int64               `json:"account_id"`
	RefundAccountID *int64              `json:"refund_account_id"`
	PriceInclude    bool                `json:"price_include"`
}

func (r CreateTaxRequest) ToInput() accountingusecase.CreateTaxInput {
	return accountingusecase.CreateTaxInput{
		Name:            r.Name,
		Type:            r.Type,
		TypeTaxUse:      r.TypeTaxUse,
		Amount:          r.Amount,
		AccountID:       r.AccountID,
		RefundAccountID: r.RefundAccountID,
		PriceInclude:    r.PriceInclude,
	}
}

type UpdateTaxRequest struct {
	Name            *string              `json:"name"`
	Type            *accounting.TaxType  `json:"type"`
	TypeTaxUse      *accounting.TaxScope `json:"type_tax_use"`
	Amount          *float64             `json:"amount"`
	AccountID       *int64               `json:"account_id"`
	RefundAccountID *int64               `json:"refund_account_id"`
	PriceInclude    *bool                `json:"price_include"`
}

func (r UpdateTaxRequest) ToInput() accountingusecase.UpdateTaxInput {
	return accountingusecase.UpdateTaxInput{
		Name:            r.Name,
		Type:            r.Type,
		TypeTaxUse:      r.TypeTaxUse,
		Amount:          r.Amount,
		AccountID:       r.AccountID,
		RefundAccountID: r.RefundAccountID,
		PriceInclude:    r.PriceInclude,
	}
}

type TaxResponse struct {
	ID              int64               `json:"id"`
	Name            string              `json:"name"`
	Type            accounting.TaxType  `json:"type"`
	TypeTaxUse      accounting.TaxScope `json:"type_tax_use"`
	Amount          float64             `json:"amount"`
	AccountID       int64               `json:"account_id"`
	RefundAccountID *int64              `json:"refund_account_id,omitempty"`
	PriceInclude    bool                `json:"price_include"`
	Active          bool                `json:"active"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

func ToTaxResponse(t *accounting.Tax) TaxResponse {
	return TaxResponse{
		ID:              t.ID,
			Name:            string(t.Name),
		Type:            t.Type,
		TypeTaxUse:      t.TypeTaxUse,
		Amount:          t.Amount,
		AccountID:       t.AccountID,
		RefundAccountID: t.RefundAccountID,
		PriceInclude:    t.PriceInclude,
		Active:          t.Active,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
}

type ComputeTaxRequest struct {
	TaxID  int64   `json:"tax_id"`
	Amount float64 `json:"amount"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Payment Terms DTOs
// ─────────────────────────────────────────────────────────────────────────────

type PaymentTermLineRequest struct {
	ValueType   accounting.PaymentTermValueType `json:"value_type"`
	ValueAmount float64                         `json:"value_amount"`
	Days        int                             `json:"days"`
	DayOfMonth  int                             `json:"day_of_month"`
}

type CreatePaymentTermRequest struct {
	Name  string                   `json:"name"`
	Note  string                   `json:"note"`
	Lines []PaymentTermLineRequest `json:"lines"`
}

func (r CreatePaymentTermRequest) ToInput() accountingusecase.CreatePaymentTermInput {
	lines := make([]accountingusecase.CreatePaymentTermLineInput, len(r.Lines))
	for i, l := range r.Lines {
		lines[i] = accountingusecase.CreatePaymentTermLineInput{
			ValueType:   l.ValueType,
			ValueAmount: l.ValueAmount,
			Days:        l.Days,
			DayOfMonth:  l.DayOfMonth,
		}
	}
	return accountingusecase.CreatePaymentTermInput{
		Name:  r.Name,
		Note:  r.Note,
		Lines: lines,
	}
}

type UpdatePaymentTermRequest struct {
	Name  *string                  `json:"name"`
	Note  *string                  `json:"note"`
	Lines []PaymentTermLineRequest `json:"lines"`
}

func (r UpdatePaymentTermRequest) ToInput() accountingusecase.UpdatePaymentTermInput {
	var lines []accountingusecase.CreatePaymentTermLineInput
	if r.Lines != nil {
		lines = make([]accountingusecase.CreatePaymentTermLineInput, len(r.Lines))
		for i, l := range r.Lines {
			lines[i] = accountingusecase.CreatePaymentTermLineInput{
				ValueType:   l.ValueType,
				ValueAmount: l.ValueAmount,
				Days:        l.Days,
				DayOfMonth:  l.DayOfMonth,
			}
		}
	}
	return accountingusecase.UpdatePaymentTermInput{
		Name:  r.Name,
		Note:  r.Note,
		Lines: lines,
	}
}

type PaymentTermLineResponse struct {
	ID          int64                           `json:"id"`
	ValueType   accounting.PaymentTermValueType `json:"value_type"`
	ValueAmount float64                         `json:"value_amount"`
	Days        int                             `json:"days"`
	DayOfMonth  int                             `json:"day_of_month"`
}

type PaymentTermResponse struct {
	ID        int64                     `json:"id"`
	Name      string                    `json:"name"`
	Note      string                    `json:"note,omitempty"`
	Active    bool                      `json:"active"`
	Lines     []PaymentTermLineResponse `json:"lines,omitempty"`
	CreatedAt time.Time                 `json:"created_at"`
	UpdatedAt time.Time                 `json:"updated_at"`
}

func ToPaymentTermResponse(pt *accounting.PaymentTerm) PaymentTermResponse {
	lines := make([]PaymentTermLineResponse, len(pt.Lines))
	for i, l := range pt.Lines {
		lines[i] = PaymentTermLineResponse{
			ID:          l.ID,
			ValueType:   l.ValueType,
			ValueAmount: l.ValueAmount,
			Days:        l.Days,
			DayOfMonth:  l.DayOfMonth,
		}
	}
	return PaymentTermResponse{
		ID:        pt.ID,
		Name:      string(pt.Name),
		Note:      string(pt.Note),
		Active:    pt.Active,
		Lines:     lines,
		CreatedAt: pt.CreatedAt,
		UpdatedAt: pt.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Moves & Invoices DTOs
// ─────────────────────────────────────────────────────────────────────────────

type JournalEntryLineRequest struct {
	AccountID int64   `json:"account_id"`
	PartnerID *int64  `json:"partner_id"`
	ProductID *int64  `json:"product_id"`
	Name      string  `json:"name"`
	Debit     float64 `json:"debit"`
	Credit    float64 `json:"credit"`
}

type CreateJournalEntryRequest struct {
	JournalID int64                     `json:"journal_id"`
	Date      time.Time                 `json:"date"`
	Ref       string                    `json:"ref"`
	Lines     []JournalEntryLineRequest `json:"lines"`
}

func (r CreateJournalEntryRequest) ToInput() accountingusecase.CreateJournalEntryInput {
	lines := make([]accountingusecase.JournalEntryLineInput, len(r.Lines))
	for i, l := range r.Lines {
		lines[i] = accountingusecase.JournalEntryLineInput{
			AccountID: l.AccountID,
			PartnerID: l.PartnerID,
			ProductID: l.ProductID,
			Name:      l.Name,
			Debit:     l.Debit,
			Credit:    l.Credit,
		}
	}
	return accountingusecase.CreateJournalEntryInput{
		JournalID: r.JournalID,
		Date:      r.Date,
		Ref:       r.Ref,
		Lines:     lines,
	}
}

type InvoiceLineItemRequest struct {
	ProductID *int64  `json:"product_id"`
	AccountID *int64  `json:"account_id"`
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	PriceUnit float64 `json:"price_unit"`
	Discount  float64 `json:"discount"`
	TaxIDs    []int64 `json:"tax_ids"`
}

type CreateInvoiceRequest struct {
	MoveType      accounting.MoveType      `json:"move_type"` // out_invoice, in_invoice
	PartnerID     int64                    `json:"partner_id"`
	JournalID     int64                    `json:"journal_id"`
	Date          time.Time                `json:"date"`
	InvoiceDate   *time.Time               `json:"invoice_date"`
	PaymentTermID *int64                   `json:"payment_term_id"`
	Currency      string                   `json:"currency"`
	Ref           string                   `json:"ref"`
	Items         []InvoiceLineItemRequest `json:"items"`
}

func (r CreateInvoiceRequest) ToInput() accountingusecase.CreateInvoiceInput {
	items := make([]accountingusecase.InvoiceLineItemInput, len(r.Items))
	for i, it := range r.Items {
		items[i] = accountingusecase.InvoiceLineItemInput{
			ProductID: it.ProductID,
			AccountID: it.AccountID,
			Name:      it.Name,
			Quantity:  it.Quantity,
			PriceUnit: it.PriceUnit,
			Discount:  it.Discount,
			TaxIDs:    it.TaxIDs,
		}
	}
	return accountingusecase.CreateInvoiceInput{
		MoveType:      r.MoveType,
		PartnerID:     r.PartnerID,
		JournalID:     r.JournalID,
		Date:          r.Date,
		InvoiceDate:   r.InvoiceDate,
		PaymentTermID: r.PaymentTermID,
		Currency:      r.Currency,
		Ref:           r.Ref,
		Items:         items,
	}
}

type UpdateMoveRequest struct {
	Date          *time.Time                `json:"date"`
	InvoiceDate   *time.Time                `json:"invoice_date"`
	PaymentTermID *int64                    `json:"payment_term_id"`
	Ref           *string                   `json:"ref"`
	Lines         []JournalEntryLineRequest `json:"lines"`
}

func (r UpdateMoveRequest) ToInput() accountingusecase.UpdateMoveInput {
	var lines []accountingusecase.JournalEntryLineInput
	if r.Lines != nil {
		lines = make([]accountingusecase.JournalEntryLineInput, len(r.Lines))
		for i, l := range r.Lines {
			lines[i] = accountingusecase.JournalEntryLineInput{
				AccountID: l.AccountID,
				PartnerID: l.PartnerID,
				ProductID: l.ProductID,
				Name:      l.Name,
				Debit:     l.Debit,
				Credit:    l.Credit,
			}
		}
	}
	return accountingusecase.UpdateMoveInput{
		Date:          r.Date,
		InvoiceDate:   r.InvoiceDate,
		PaymentTermID: r.PaymentTermID,
		Ref:           r.Ref,
		Lines:         lines,
	}
}

type ReverseMoveRequest struct {
	ReversalDate time.Time `json:"reversal_date"`
	Ref          string    `json:"ref"`
}

type MoveLineResponse struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"account_id"`
	PartnerID *int64    `json:"partner_id,omitempty"`
	ProductID *int64    `json:"product_id,omitempty"`
	Name      string    `json:"name"`
	Quantity  float64   `json:"quantity"`
	PriceUnit float64   `json:"price_unit"`
	Discount  float64   `json:"discount"`
	Debit     float64   `json:"debit"`
	Credit    float64   `json:"credit"`
	Balance   float64   `json:"balance"`
	TaxIDs    []int64   `json:"tax_ids,omitempty"`
	TaxAmount float64   `json:"tax_amount"`
	CreatedAt time.Time `json:"created_at"`
}

type MoveResponse struct {
	ID              int64                   `json:"id"`
	Name            string                  `json:"name"`
	MoveType        accounting.MoveType     `json:"move_type"`
	JournalID       int64                   `json:"journal_id"`
	PartnerID       *int64                  `json:"partner_id,omitempty"`
	Date            time.Time               `json:"date"`
	InvoiceDate     *time.Time              `json:"invoice_date,omitempty"`
	InvoiceDueDate  *time.Time              `json:"invoice_date_due,omitempty"`
	PaymentTermID   *int64                  `json:"payment_term_id,omitempty"`
	State           accounting.MoveState    `json:"state"`
	PaymentState    accounting.PaymentState `json:"payment_state"`
	AmountUntaxed   float64                 `json:"amount_untaxed"`
	AmountTax       float64                 `json:"amount_tax"`
	AmountTotal     float64                 `json:"amount_total"`
	AmountResidual  float64                 `json:"amount_residual"`
	Currency        string                  `json:"currency"`
	Ref             string                  `json:"ref,omitempty"`
	ReversedEntryID *int64                  `json:"reversed_entry_id,omitempty"`
	Lines           []MoveLineResponse      `json:"lines,omitempty"`
	Active          bool                    `json:"active"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

func ToMoveResponse(m *accounting.AccountMove) MoveResponse {
	lines := make([]MoveLineResponse, len(m.Lines))
	for i, l := range m.Lines {
		lines[i] = MoveLineResponse{
			ID:        l.ID,
			AccountID: l.AccountID,
			PartnerID: l.PartnerID,
			ProductID: l.ProductID,
			Name:      l.Name,
			Quantity:  l.Quantity,
			PriceUnit: l.PriceUnit,
			Discount:  l.Discount,
			Debit:     l.Debit,
			Credit:    l.Credit,
			Balance:   l.Balance,
			TaxIDs:    l.TaxIDs,
			TaxAmount: l.TaxAmount,
			CreatedAt: l.CreatedAt,
		}
	}

	return MoveResponse{
		ID:              m.ID,
		Name:            m.Name,
		MoveType:        m.MoveType,
		JournalID:       m.JournalID,
		PartnerID:       m.PartnerID,
		Date:            m.Date,
		InvoiceDate:     m.InvoiceDate,
		InvoiceDueDate:  m.InvoiceDueDate,
		PaymentTermID:   m.PaymentTermID,
		State:           m.State,
		PaymentState:    m.PaymentState,
		AmountUntaxed:   m.AmountUntaxed,
		AmountTax:       m.AmountTax,
		AmountTotal:     m.AmountTotal,
		AmountResidual:  m.AmountResidual,
		Currency:        m.Currency,
		Ref:             m.Ref,
		ReversedEntryID: m.ReversedEntryID,
		Lines:           lines,
		Active:          m.Active,
		CreatedAt:       m.Audit.CreatedAt,
		UpdatedAt:       m.Audit.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// EDI DTOs
// ─────────────────────────────────────────────────────────────────────────────

type EDIDocumentResponse struct {
	ID              int64                         `json:"id"`
	MoveID          int64                         `json:"move_id"`
	Format          accounting.EDIFormat          `json:"format"`
	TransactionType accounting.EDITransactionType `json:"transaction_type"`
	State           accounting.EDIState           `json:"state"`
	QRCode          string                        `json:"qr_code,omitempty"`
	ErrorMsg        string                        `json:"error_msg,omitempty"`
	SentAt          *time.Time                    `json:"sent_at,omitempty"`
	CreatedAt       time.Time                     `json:"created_at"`
}

func ToEDIDocumentResponse(d *accounting.EDIDocument) EDIDocumentResponse {
	return EDIDocumentResponse{
		ID:              d.ID,
		MoveID:          d.MoveID,
		Format:          d.Format,
		TransactionType: d.TransactionType,
		State:           d.State,
		QRCode:          d.QRCode,
		ErrorMsg:        d.ErrorMsg,
		SentAt:          d.SentAt,
		CreatedAt:       d.CreatedAt,
	}
}
