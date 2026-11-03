package accounting

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"cashflow_backend/internal/platform/money"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const dateLayout = "2006-01-02"

// CreateAccountRequest contains the fields accepted when creating an account.
type CreateAccountRequest struct {
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	Type           string     `json:"account_type"`
	NormalBalance  string     `json:"normal_balance"`
	CurrencyID     string     `json:"currency_id"`
	ParentID       *uuid.UUID `json:"parent_id,omitempty"`
	IsReconcilable bool       `json:"is_reconcilable"`
}

// Validate normalizes and validates the requested account values.
func (r *CreateAccountRequest) Validate() error {
	r.Code = strings.TrimSpace(r.Code)
	if r.Code == "" {
		return errors.New("code is required")
	}
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return errors.New("name is required")
	}

	accountType := AccountType(strings.ToLower(strings.TrimSpace(r.Type)))
	switch accountType {
	case AccountTypeAsset, AccountTypeLiability, AccountTypeEquity, AccountTypeIncome, AccountTypeExpense:
		r.Type = string(accountType)
	default:
		return errors.New("invalid account_type: must be asset, liability, equity, income, or expense")
	}

	normalBalance := NormalBalance(strings.ToLower(strings.TrimSpace(r.NormalBalance)))
	if normalBalance == "" {
		if accountType == AccountTypeAsset || accountType == AccountTypeExpense {
			normalBalance = NormalBalanceDebit
		} else {
			normalBalance = NormalBalanceCredit
		}
	} else if normalBalance != NormalBalanceDebit && normalBalance != NormalBalanceCredit {
		return errors.New("normal_balance must be debit or credit")
	}
	r.NormalBalance = string(normalBalance)

	currencyID, err := normalizeCurrency(r.CurrencyID, "SAR")
	if err != nil {
		return errors.New("currency_id must be a supported ISO 4217 currency code")
	}
	r.CurrencyID = currencyID
	return nil
}

// AccountResponse is the public representation of an account.
type AccountResponse struct {
	ID             uuid.UUID  `json:"id"`
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	Type           string     `json:"account_type"`
	NormalBalance  string     `json:"normal_balance"`
	CurrencyID     string     `json:"currency_id"`
	ParentID       *uuid.UUID `json:"parent_id,omitempty"`
	IsReconcilable bool       `json:"is_reconcilable"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
}

// AccountTreeNode represents an account and its descendants.
type AccountTreeNode struct {
	AccountResponse
	Children []AccountTreeNode `json:"children,omitempty"`
}

// ToAccountResponse maps an account entity to its public representation.
func ToAccountResponse(a *Account) AccountResponse {
	return AccountResponse{
		ID:             a.ID,
		Code:           a.Code,
		Name:           a.Name,
		Type:           string(a.Type),
		NormalBalance:  string(a.NormalBalance),
		CurrencyID:     a.CurrencyID,
		ParentID:       cloneUUID(a.ParentID),
		IsReconcilable: a.IsReconcilable,
		IsActive:       a.IsActive,
		CreatedAt:      a.CreatedAt,
	}
}

// CreateFiscalYearRequest contains the fields accepted when creating a fiscal year.
type CreateFiscalYearRequest struct {
	Year      int    `json:"year"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// Validate ensures the fiscal year has a supported year and valid date range.
func (r *CreateFiscalYearRequest) Validate() error {
	if r.Year < 2000 || r.Year > 2100 {
		return errors.New("year must be between 2000 and 2100")
	}
	start, err := parseDate(r.StartDate, "start_date")
	if err != nil {
		return err
	}
	end, err := parseDate(r.EndDate, "end_date")
	if err != nil {
		return err
	}
	if !end.After(start) {
		return errors.New("end_date must be strictly after start_date")
	}
	if start.Year() != r.Year {
		return errors.New("year must match start_date year")
	}
	if start.Day() != 1 || !end.Equal(start.AddDate(1, 0, -1)) {
		return errors.New("fiscal year must span 12 complete calendar months")
	}
	return nil
}

// ChangePeriodStateRequest contains a fiscal period's requested state.
type ChangePeriodStateRequest struct {
	State string `json:"state"`
}

// Validate normalizes and validates the requested fiscal period state.
func (r *ChangePeriodStateRequest) Validate() error {
	state := PeriodState(strings.ToLower(strings.TrimSpace(r.State)))
	switch state {
	case PeriodStateOpen, PeriodStateSoftClosed, PeriodStateHardClosed:
		r.State = string(state)
		return nil
	default:
		return errors.New("state must be open, soft_closed, or hard_closed")
	}
}

// FiscalPeriodResponse is the public representation of a fiscal period.
type FiscalPeriodResponse struct {
	ID           uuid.UUID  `json:"id"`
	FiscalYearID uuid.UUID  `json:"fiscal_year_id"`
	PeriodNumber int        `json:"period_number"`
	Name         string     `json:"name"`
	StartDate    string     `json:"start_date"`
	EndDate      string     `json:"end_date"`
	State        string     `json:"state"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
}

// FiscalYearResponse is the public representation of a fiscal year and its periods.
type FiscalYearResponse struct {
	ID        uuid.UUID              `json:"id"`
	Year      int                    `json:"year"`
	StartDate string                 `json:"start_date"`
	EndDate   string                 `json:"end_date"`
	State     string                 `json:"state"`
	Periods   []FiscalPeriodResponse `json:"periods"`
}

// ToFiscalPeriodResponse maps a fiscal period entity to its public representation.
func ToFiscalPeriodResponse(p *FiscalPeriod) FiscalPeriodResponse {
	return FiscalPeriodResponse{
		ID:           p.ID,
		FiscalYearID: p.FiscalYearID,
		PeriodNumber: p.PeriodNumber,
		Name:         p.Name,
		StartDate:    p.StartDate.Format(dateLayout),
		EndDate:      p.EndDate.Format(dateLayout),
		State:        string(p.State),
		ClosedAt:     cloneTime(p.ClosedAt),
	}
}

// ToFiscalYearResponse maps a fiscal year and its periods to the public representation.
func ToFiscalYearResponse(year *FiscalYear, periods []*FiscalPeriod) FiscalYearResponse {
	response := FiscalYearResponse{
		ID:        year.ID,
		Year:      year.Year,
		StartDate: year.StartDate.Format(dateLayout),
		EndDate:   year.EndDate.Format(dateLayout),
		State:     year.State,
		Periods:   make([]FiscalPeriodResponse, len(periods)),
	}
	for i, period := range periods {
		response.Periods[i] = ToFiscalPeriodResponse(period)
	}
	return response
}

// MoveLineRequest contains one debit or credit line for a journal entry.
type MoveLineRequest struct {
	AccountID      uuid.UUID       `json:"account_id"`
	PartnerID      *uuid.UUID      `json:"partner_id,omitempty"`
	Debit          decimal.Decimal `json:"debit"`
	Credit         decimal.Decimal `json:"credit"`
	AmountCurrency decimal.Decimal `json:"amount_currency,omitempty"`
	CurrencyID     string          `json:"currency_id,omitempty"`
	CostCenterID   *uuid.UUID      `json:"cost_center_id,omitempty"`
	MatchingNumber string          `json:"matching_number,omitempty"`
}

// CreateMoveRequest contains the fields accepted when creating a journal entry.
type CreateMoveRequest struct {
	JournalID uuid.UUID         `json:"journal_id"`
	Date      string            `json:"date"`
	Ref       string            `json:"ref,omitempty"`
	Lines     []MoveLineRequest `json:"lines"`
}

// Validate normalizes and validates the journal entry and all of its lines.
func (r *CreateMoveRequest) Validate() error {
	if r.JournalID == uuid.Nil {
		return errors.New("journal_id is required")
	}
	if _, err := parseDate(r.Date, "date"); err != nil {
		return err
	}
	if len(r.Lines) < 2 {
		return errors.New("a move must contain at least 2 lines")
	}

	var totalDebit, totalCredit decimal.Decimal
	for i := range r.Lines {
		line := &r.Lines[i]
		if line.AccountID == uuid.Nil {
			return errors.New("account_id is required on all lines")
		}
		if line.Debit.IsNegative() || line.Credit.IsNegative() {
			return errors.New("negative debit or credit amounts are not permitted")
		}
		if line.Debit.IsZero() && line.Credit.IsZero() {
			return errors.New("line must specify non-zero debit or credit")
		}
		if !line.Debit.IsZero() && !line.Credit.IsZero() {
			return errors.New("line cannot have both debit and credit; split into separate lines")
		}
		currencyID, err := normalizeCurrency(line.CurrencyID, "SAR")
		if err != nil {
			return errors.New("currency_id must be a supported ISO 4217 currency code")
		}
		line.CurrencyID = currencyID
		totalDebit = totalDebit.Add(line.Debit)
		totalCredit = totalCredit.Add(line.Credit)
	}

	if !totalDebit.Equal(totalCredit) {
		return errors.New("total debit must equal total credit")
	}
	return nil
}

// MoveLineResponse is the public representation of a journal entry line.
type MoveLineResponse struct {
	ID             uuid.UUID       `json:"id"`
	AccountID      uuid.UUID       `json:"account_id"`
	Debit          decimal.Decimal `json:"debit"`
	Credit         decimal.Decimal `json:"credit"`
	AmountCurrency decimal.Decimal `json:"amount_currency"`
	CurrencyID     string          `json:"currency_id"`
	CostCenterID   *uuid.UUID      `json:"cost_center_id,omitempty"`
	MatchingNumber string          `json:"matching_number,omitempty"`
}

// MoveResponse is the public representation of a journal entry.
type MoveResponse struct {
	ID             uuid.UUID          `json:"id"`
	Name           string             `json:"name"`
	JournalID      uuid.UUID          `json:"journal_id"`
	Date           string             `json:"date"`
	State          string             `json:"state"`
	Ref            string             `json:"ref,omitempty"`
	ReversedMoveID *uuid.UUID         `json:"reversed_move_id,omitempty"`
	HashSignature  string             `json:"hash_signature,omitempty"`
	PostedAt       *time.Time         `json:"posted_at,omitempty"`
	Lines          []MoveLineResponse `json:"lines"`
}

// ToMoveResponse maps a journal entry entity and its lines to a public response.
func ToMoveResponse(m *AccountMove) MoveResponse {
	lines := make([]MoveLineResponse, len(m.Lines))
	for i, line := range m.Lines {
		lines[i] = MoveLineResponse{
			ID:             line.ID,
			AccountID:      line.AccountID,
			Debit:          line.Debit,
			Credit:         line.Credit,
			AmountCurrency: line.AmountCurrency,
			CurrencyID:     line.CurrencyID,
			CostCenterID:   cloneUUID(line.CostCenterID),
			MatchingNumber: line.MatchingNumber,
		}
	}
	return MoveResponse{
		ID:             m.ID,
		Name:           m.Name,
		JournalID:      m.JournalID,
		Date:           m.Date.Format(dateLayout),
		State:          string(m.State),
		Ref:            m.Ref,
		ReversedMoveID: cloneUUID(m.ReversedMoveID),
		HashSignature:  m.HashSignature,
		PostedAt:       cloneTime(m.PostedAt),
		Lines:          lines,
	}
}

// TrialBalanceRowResponse is the public representation of one account's balances.
type TrialBalanceRowResponse struct {
	AccountID     uuid.UUID       `json:"account_id"`
	AccountCode   string          `json:"account_code"`
	AccountName   string          `json:"account_name"`
	InitialDebit  decimal.Decimal `json:"initial_debit"`
	InitialCredit decimal.Decimal `json:"initial_credit"`
	PeriodDebit   decimal.Decimal `json:"period_debit"`
	PeriodCredit  decimal.Decimal `json:"period_credit"`
	EndingDebit   decimal.Decimal `json:"ending_debit"`
	EndingCredit  decimal.Decimal `json:"ending_credit"`
}

// TrialBalanceResponse is the public representation of a trial balance report.
type TrialBalanceResponse struct {
	FromDate    string                    `json:"from_date"`
	ToDate      string                    `json:"to_date"`
	TotalDebit  decimal.Decimal           `json:"total_debit"`
	TotalCredit decimal.Decimal           `json:"total_credit"`
	IsBalanced  bool                      `json:"is_balanced"`
	Accounts    []TrialBalanceRowResponse `json:"accounts"`
}

// ToTrialBalanceResponse maps trial balance data to its public representation.
func ToTrialBalanceResponse(fromDate, toDate time.Time, totalDebit, totalCredit decimal.Decimal, rows []*TrialBalanceRow) TrialBalanceResponse {
	accounts := make([]TrialBalanceRowResponse, len(rows))
	for i, row := range rows {
		accounts[i] = ToTrialBalanceRowResponse(row)
	}
	return TrialBalanceResponse{
		FromDate:    fromDate.Format(dateLayout),
		ToDate:      toDate.Format(dateLayout),
		TotalDebit:  totalDebit,
		TotalCredit: totalCredit,
		IsBalanced:  totalDebit.Equal(totalCredit),
		Accounts:    accounts,
	}
}

// ToTrialBalanceRowResponse maps a trial balance read model to its public form.
func ToTrialBalanceRowResponse(row *TrialBalanceRow) TrialBalanceRowResponse {
	return TrialBalanceRowResponse{
		AccountID:     row.AccountID,
		AccountCode:   row.AccountCode,
		AccountName:   row.AccountName,
		InitialDebit:  row.InitialDebit,
		InitialCredit: row.InitialCredit,
		PeriodDebit:   row.PeriodDebit,
		PeriodCredit:  row.PeriodCredit,
		EndingDebit:   row.EndingDebit,
		EndingCredit:  row.EndingCredit,
	}
}

// GeneralLedgerLineResponse is the public representation of one ledger entry.
type GeneralLedgerLineResponse struct {
	MoveID         uuid.UUID       `json:"move_id"`
	MoveName       string          `json:"move_name"`
	Date           string          `json:"date"`
	Ref            string          `json:"ref"`
	Debit          decimal.Decimal `json:"debit"`
	Credit         decimal.Decimal `json:"credit"`
	RunningBalance decimal.Decimal `json:"running_balance"`
}

// ToGeneralLedgerLineResponse maps a ledger read model to its public form.
func ToGeneralLedgerLineResponse(line *GeneralLedgerLineRow) GeneralLedgerLineResponse {
	return GeneralLedgerLineResponse{
		MoveID:         line.MoveID,
		MoveName:       line.MoveName,
		Date:           line.Date.Format(dateLayout),
		Ref:            line.Ref,
		Debit:          line.Debit,
		Credit:         line.Credit,
		RunningBalance: line.RunningBalance,
	}
}

// IngestSubledgerEventRequest contains an external economic event for accounting.
type IngestSubledgerEventRequest struct {
	IdempotencyKey string          `json:"idempotency_key"`
	EventType      string          `json:"event_type"`
	Date           string          `json:"date"`
	Ref            string          `json:"ref"`
	Amount         decimal.Decimal `json:"amount"`
	Currency       string          `json:"currency"`
	Payload        map[string]any  `json:"payload"`
}

// Validate normalizes and validates the incoming subledger event.
func (r *IngestSubledgerEventRequest) Validate() error {
	r.IdempotencyKey = strings.TrimSpace(r.IdempotencyKey)
	if r.IdempotencyKey == "" {
		return errors.New("idempotency_key is required")
	}
	r.EventType = strings.TrimSpace(r.EventType)
	if r.EventType == "" {
		return errors.New("event_type is required")
	}
	if utf8.RuneCountInString(r.IdempotencyKey) > 128 {
		return errors.New("idempotency_key must not exceed 128 characters")
	}
	if utf8.RuneCountInString(r.EventType) > 100 {
		return errors.New("event_type must not exceed 100 characters")
	}
	if utf8.RuneCountInString(r.Ref) > 255 {
		return errors.New("ref must not exceed 255 characters")
	}
	if r.Amount.IsZero() || r.Amount.IsNegative() {
		return errors.New("amount must be strictly positive")
	}
	if _, err := parseDate(r.Date, "date"); err != nil {
		return err
	}
	currency, err := normalizeCurrency(r.Currency, "")
	if err != nil {
		return errors.New("currency must be a supported ISO 4217 currency code")
	}
	r.Currency = currency
	return nil
}

func parseDate(value, field string) (time.Time, error) {
	date, err := time.Parse(dateLayout, value)
	if err != nil || date.Format(dateLayout) != value {
		return time.Time{}, fmt.Errorf("%s must be in YYYY-MM-DD format", field)
	}
	return date, nil
}

func normalizeCurrency(value, defaultCode string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(value))
	if code == "" {
		code = defaultCode
	}
	if _, ok := money.SupportedCurrencies[code]; !ok {
		return "", money.ErrInvalidCurrency
	}
	return code, nil
}

func cloneUUID(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
