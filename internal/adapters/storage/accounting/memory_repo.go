package accountingstorage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/accounting"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is a thread-safe, high-fidelity in-memory implementation of accounting.Repository.
type MemoryRepo struct {
	mu           sync.RWMutex
	accounts     map[int64]*accounting.Account
	journals     map[int64]*accounting.Journal
	taxes        map[int64]*accounting.Tax
	paymentTerms map[int64]*accounting.PaymentTerm
	moves        map[int64]*accounting.AccountMove
	moveLines    map[int64]*accounting.AccountMoveLine

	lastAccountID     int64
	lastJournalID     int64
	lastTaxID         int64
	lastPaymentTermID int64
	lastMoveID        int64
	lastMoveLineID    int64
}

// NewMemoryRepo creates an initialized MemoryRepo seeded with standard accounting configuration.
func NewMemoryRepo() *MemoryRepo {
	r := &MemoryRepo{
		accounts:     make(map[int64]*accounting.Account),
		journals:     make(map[int64]*accounting.Journal),
		taxes:        make(map[int64]*accounting.Tax),
		paymentTerms: make(map[int64]*accounting.PaymentTerm),
		moves:        make(map[int64]*accounting.AccountMove),
		moveLines:    make(map[int64]*accounting.AccountMoveLine),
	}

	now := time.Now().UTC()

	// 1. Seed Standard Chart of Accounts
	accounts := []accounting.Account{
		{ID: 1, Code: "101000", Name: "Cash on Hand", Type: accounting.AccountTypeAssetCash, Currency: "USD", Active: true},
		{ID: 2, Code: "102000", Name: "Bank Account", Type: accounting.AccountTypeAssetCash, Currency: "USD", Active: true},
		{ID: 3, Code: "120000", Name: "Accounts Receivable", Type: accounting.AccountTypeAssetReceivable, Reconcile: true, Currency: "USD", Active: true},
		{ID: 4, Code: "130000", Name: "VAT Input (Tax Receivable)", Type: accounting.AccountTypeAssetCurrent, Currency: "USD", Active: true},
		{ID: 5, Code: "140000", Name: "Inventory", Type: accounting.AccountTypeAssetCurrent, Currency: "USD", Active: true},
		{ID: 6, Code: "210000", Name: "Accounts Payable", Type: accounting.AccountTypeLiabilityPayable, Reconcile: true, Currency: "USD", Active: true},
		{ID: 7, Code: "220000", Name: "VAT Output (Tax Payable)", Type: accounting.AccountTypeLiabilityCurrent, Currency: "USD", Active: true},
		{ID: 8, Code: "300000", Name: "Capital / Equity", Type: accounting.AccountTypeEquity, Currency: "USD", Active: true},
		{ID: 9, Code: "320000", Name: "Retained Earnings", Type: accounting.AccountTypeEquity, Currency: "USD", Active: true},
		{ID: 10, Code: "400000", Name: "Product Sales Revenue", Type: accounting.AccountTypeIncome, Currency: "USD", Active: true},
		{ID: 11, Code: "410000", Name: "Service Revenue", Type: accounting.AccountTypeIncome, Currency: "USD", Active: true},
		{ID: 12, Code: "500000", Name: "Cost of Goods Sold", Type: accounting.AccountTypeExpenseDirectCost, Currency: "USD", Active: true},
		{ID: 13, Code: "600000", Name: "Operating Expenses", Type: accounting.AccountTypeExpense, Currency: "USD", Active: true},
		{ID: 14, Code: "610000", Name: "Salaries and Wages", Type: accounting.AccountTypeExpense, Currency: "USD", Active: true},
		{ID: 15, Code: "999999", Name: "Undistributed Profits/Losses", Type: accounting.AccountTypeEquity, Currency: "USD", Active: true},
	}
	for _, a := range accounts {
		clone := a
		clone.Audit.CreatedAt = now
		clone.Audit.UpdatedAt = now
		r.accounts[a.ID] = &clone
	}
	r.lastAccountID = 15

	// 2. Seed Standard Journals
	defAcc10 := int64(10)
	defAcc13 := int64(13)
	defAcc2 := int64(2)
	defAcc1 := int64(1)

	journals := []accounting.Journal{
		{ID: 1, Name: "Customer Invoices", Code: "INV", Type: accounting.JournalTypeSale, DefaultAccountID: &defAcc10, SequencePrefix: "INV/%Y/", NextNumber: 1, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Vendor Bills", Code: "BILL", Type: accounting.JournalTypePurchase, DefaultAccountID: &defAcc13, SequencePrefix: "BILL/%Y/", NextNumber: 1, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "Bank", Code: "BNK1", Type: accounting.JournalTypeBank, DefaultAccountID: &defAcc2, SuspenseAccountID: &defAcc2, SequencePrefix: "BNK1/%Y/", NextNumber: 1, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "Cash", Code: "CSH1", Type: accounting.JournalTypeCash, DefaultAccountID: &defAcc1, SuspenseAccountID: &defAcc1, SequencePrefix: "CSH1/%Y/", NextNumber: 1, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 5, Name: "Miscellaneous Operations", Code: "MISC", Type: accounting.JournalTypeGeneral, SequencePrefix: "MISC/%Y/", NextNumber: 1, Active: true, CreatedAt: now, UpdatedAt: now},
	}
	for _, j := range journals {
		clone := j
		r.journals[j.ID] = &clone
	}
	r.lastJournalID = 5

	// 3. Seed Standard Taxes
	refAcc7 := int64(7)
	refAcc4 := int64(4)
	taxes := []accounting.Tax{
		{ID: 1, Name: "15% Sales VAT", Type: accounting.TaxTypePercent, TypeTaxUse: accounting.TaxScopeSale, Amount: 15.0, AccountID: 7, RefundAccountID: &refAcc7, PriceInclude: false, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "15% Purchase VAT", Type: accounting.TaxTypePercent, TypeTaxUse: accounting.TaxScopePurchase, Amount: 15.0, AccountID: 4, RefundAccountID: &refAcc4, PriceInclude: false, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "15% VAT Included", Type: accounting.TaxTypePercent, TypeTaxUse: accounting.TaxScopeSale, Amount: 15.0, AccountID: 7, RefundAccountID: &refAcc7, PriceInclude: true, Active: true, CreatedAt: now, UpdatedAt: now},
	}
	for _, t := range taxes {
		clone := t
		r.taxes[t.ID] = &clone
	}
	r.lastTaxID = 3

	// 4. Seed Standard Payment Terms
	terms := []accounting.PaymentTerm{
		{ID: 1, Name: "Immediate Payment", Note: "Payment due immediately on issuance", Active: true, CreatedAt: now, UpdatedAt: now, Lines: []accounting.PaymentTermLine{{ID: 1, PaymentTermID: 1, ValueType: accounting.PaymentTermValueBalance, Days: 0}}},
		{ID: 2, Name: "15 Days", Note: "Payment due within 15 calendar days", Active: true, CreatedAt: now, UpdatedAt: now, Lines: []accounting.PaymentTermLine{{ID: 2, PaymentTermID: 2, ValueType: accounting.PaymentTermValueBalance, Days: 15}}},
		{ID: 3, Name: "30 Days", Note: "Payment due within 30 calendar days", Active: true, CreatedAt: now, UpdatedAt: now, Lines: []accounting.PaymentTermLine{{ID: 3, PaymentTermID: 3, ValueType: accounting.PaymentTermValueBalance, Days: 30}}},
	}
	for _, pt := range terms {
		clone := pt
		r.paymentTerms[pt.ID] = &clone
	}
	r.lastPaymentTermID = 3

	return r
}

// ─────────────────────────────────────────────────────────────────────────────
// Chart of Accounts
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateAccount(ctx context.Context, a *accounting.Account) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.accounts {
		if strings.EqualFold(existing.Code, a.Code) {
			return platformerrors.Conflict(fmt.Sprintf("account with code '%s' already exists", a.Code))
		}
	}

	r.lastAccountID++
	a.ID = r.lastAccountID
	now := time.Now().UTC()
	a.Audit.CreatedAt = now
	a.Audit.UpdatedAt = now
	a.Active = true

	clone := *a
	r.accounts[a.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetAccountByID(ctx context.Context, id int64) (*accounting.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, exists := r.accounts[id]
	if !exists || !a.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("account with id %d not found", id))
	}
	clone := *a
	return &clone, nil
}

func (r *MemoryRepo) GetAccountByCode(ctx context.Context, code string) (*accounting.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, a := range r.accounts {
		if a.Active && strings.EqualFold(a.Code, code) {
			clone := *a
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound(fmt.Sprintf("account with code '%s' not found", code))
}

func (r *MemoryRepo) UpdateAccount(ctx context.Context, a *accounting.Account) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.accounts[a.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound(fmt.Sprintf("account with id %d not found", a.ID))
	}

	for _, other := range r.accounts {
		if other.ID != a.ID && strings.EqualFold(other.Code, a.Code) {
			return platformerrors.Conflict(fmt.Sprintf("account with code '%s' already exists", a.Code))
		}
	}

	a.Audit.CreatedAt = existing.Audit.CreatedAt
	a.Audit.UpdatedAt = time.Now().UTC()
	clone := *a
	r.accounts[a.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteAccount(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	a, exists := r.accounts[id]
	if !exists || !a.Active {
		return platformerrors.NotFound(fmt.Sprintf("account with id %d not found", id))
	}

	// Check if account is used in any move line
	for _, l := range r.moveLines {
		if l.AccountID == id {
			return platformerrors.Conflict("cannot delete account referenced by journal entries")
		}
	}

	a.Active = false
	a.Audit.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListAccounts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[accounting.Account], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []accounting.Account
	for _, a := range r.accounts {
		if !a.Active {
			continue
		}
		list = append(list, *a)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Code < list[j].Code
	})

	total := int64(len(list))
	offset := page.Offset()
	limit := page.LimitClamped()

	var items []accounting.Account
	if offset < len(list) {
		end := offset + limit
		if end > len(list) {
			end = len(list)
		}
		items = list[offset:end]
	}

	return pagination.NewPageResult(items, total, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Journals
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateJournal(ctx context.Context, j *accounting.Journal) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.journals {
		if strings.EqualFold(existing.Code, j.Code) {
			return platformerrors.Conflict(fmt.Sprintf("journal with code '%s' already exists", j.Code))
		}
	}

	r.lastJournalID++
	j.ID = r.lastJournalID
	now := time.Now().UTC()
	j.CreatedAt = now
	j.UpdatedAt = now
	j.Active = true

	clone := *j
	r.journals[j.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetJournalByID(ctx context.Context, id int64) (*accounting.Journal, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	j, exists := r.journals[id]
	if !exists || !j.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("journal with id %d not found", id))
	}
	clone := *j
	return &clone, nil
}

func (r *MemoryRepo) GetJournalByCode(ctx context.Context, code string) (*accounting.Journal, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, j := range r.journals {
		if j.Active && strings.EqualFold(j.Code, code) {
			clone := *j
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound(fmt.Sprintf("journal with code '%s' not found", code))
}

func (r *MemoryRepo) UpdateJournal(ctx context.Context, j *accounting.Journal) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.journals[j.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound(fmt.Sprintf("journal with id %d not found", j.ID))
	}

	for _, other := range r.journals {
		if other.ID != j.ID && strings.EqualFold(other.Code, j.Code) {
			return platformerrors.Conflict(fmt.Sprintf("journal with code '%s' already exists", j.Code))
		}
	}

	j.CreatedAt = existing.CreatedAt
	j.UpdatedAt = time.Now().UTC()
	clone := *j
	r.journals[j.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteJournal(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, exists := r.journals[id]
	if !exists || !j.Active {
		return platformerrors.NotFound(fmt.Sprintf("journal with id %d not found", id))
	}

	for _, m := range r.moves {
		if m.JournalID == id {
			return platformerrors.Conflict("cannot delete journal with associated account moves")
		}
	}

	j.Active = false
	j.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListJournals(ctx context.Context) ([]accounting.Journal, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []accounting.Journal
	for _, j := range r.journals {
		if j.Active {
			list = append(list, *j)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})
	return list, nil
}

func (r *MemoryRepo) GetNextSequence(ctx context.Context, journalID int64, year int) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, exists := r.journals[journalID]
	if !exists || !j.Active {
		return "", platformerrors.NotFound(fmt.Sprintf("journal with id %d not found", journalID))
	}

	seqNumber := j.NextNumber
	j.NextNumber++
	j.UpdatedAt = time.Now().UTC()

	return j.FormatSequence(year, seqNumber), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Taxes
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateTax(ctx context.Context, t *accounting.Tax) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastTaxID++
	t.ID = r.lastTaxID
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	t.Active = true

	clone := *t
	r.taxes[t.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetTaxByID(ctx context.Context, id int64) (*accounting.Tax, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, exists := r.taxes[id]
	if !exists || !t.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("tax with id %d not found", id))
	}
	clone := *t
	return &clone, nil
}

func (r *MemoryRepo) UpdateTax(ctx context.Context, t *accounting.Tax) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.taxes[t.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound(fmt.Sprintf("tax with id %d not found", t.ID))
	}

	t.CreatedAt = existing.CreatedAt
	t.UpdatedAt = time.Now().UTC()
	clone := *t
	r.taxes[t.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteTax(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, exists := r.taxes[id]
	if !exists || !t.Active {
		return platformerrors.NotFound(fmt.Sprintf("tax with id %d not found", id))
	}

	t.Active = false
	t.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListTaxes(ctx context.Context, scope *accounting.TaxScope) ([]accounting.Tax, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []accounting.Tax
	for _, t := range r.taxes {
		if !t.Active {
			continue
		}
		if scope != nil && *scope != "" && t.TypeTaxUse != *scope {
			continue
		}
		list = append(list, *t)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})
	return list, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Payment Terms
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreatePaymentTerm(ctx context.Context, pt *accounting.PaymentTerm) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastPaymentTermID++
	pt.ID = r.lastPaymentTermID
	now := time.Now().UTC()
	pt.CreatedAt = now
	pt.UpdatedAt = now
	pt.Active = true

	for i := range pt.Lines {
		pt.Lines[i].ID = int64(i + 1)
		pt.Lines[i].PaymentTermID = pt.ID
	}

	clone := *pt
	r.paymentTerms[pt.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetPaymentTermByID(ctx context.Context, id int64) (*accounting.PaymentTerm, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pt, exists := r.paymentTerms[id]
	if !exists || !pt.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("payment term with id %d not found", id))
	}
	clone := *pt
	return &clone, nil
}

func (r *MemoryRepo) UpdatePaymentTerm(ctx context.Context, pt *accounting.PaymentTerm) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.paymentTerms[pt.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound(fmt.Sprintf("payment term with id %d not found", pt.ID))
	}

	pt.CreatedAt = existing.CreatedAt
	pt.UpdatedAt = time.Now().UTC()
	clone := *pt
	r.paymentTerms[pt.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeletePaymentTerm(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pt, exists := r.paymentTerms[id]
	if !exists || !pt.Active {
		return platformerrors.NotFound(fmt.Sprintf("payment term with id %d not found", id))
	}

	pt.Active = false
	pt.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListPaymentTerms(ctx context.Context) ([]accounting.PaymentTerm, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []accounting.PaymentTerm
	for _, pt := range r.paymentTerms {
		if pt.Active {
			list = append(list, *pt)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})
	return list, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Account Moves & Invoices
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateMove(ctx context.Context, m *accounting.AccountMove) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastMoveID++
	m.ID = r.lastMoveID
	now := time.Now().UTC()
	m.Audit.CreatedAt = now
	m.Audit.UpdatedAt = now
	m.Active = true

	lines := make([]accounting.AccountMoveLine, len(m.Lines))
	for i, l := range m.Lines {
		r.lastMoveLineID++
		l.ID = r.lastMoveLineID
		l.MoveID = m.ID
		l.Balance = l.Debit - l.Credit
		l.CreatedAt = now
		l.UpdatedAt = now
		lines[i] = l
		r.moveLines[l.ID] = &l
	}
	m.Lines = lines

	clone := *m
	r.moves[m.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetMoveByID(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, exists := r.moves[id]
	if !exists || !m.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("account move with id %d not found", id))
	}
	clone := *m
	return &clone, nil
}

func (r *MemoryRepo) GetMoveWithLines(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, exists := r.moves[id]
	if !exists || !m.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("account move with id %d not found", id))
	}

	clone := *m
	var lines []accounting.AccountMoveLine
	for _, l := range r.moveLines {
		if l.MoveID == id {
			lines = append(lines, *l)
		}
	}
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].ID < lines[j].ID
	})
	clone.Lines = lines
	return &clone, nil
}

func (r *MemoryRepo) UpdateMove(ctx context.Context, m *accounting.AccountMove) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.moves[m.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound(fmt.Sprintf("account move with id %d not found", m.ID))
	}

	now := time.Now().UTC()
	m.Audit.CreatedAt = existing.Audit.CreatedAt
	m.Audit.UpdatedAt = now

	// If lines were modified, update them
	if len(m.Lines) > 0 {
		// Remove existing lines for this move
		for lid, l := range r.moveLines {
			if l.MoveID == m.ID {
				delete(r.moveLines, lid)
			}
		}
		// Insert new lines
		for i := range m.Lines {
			r.lastMoveLineID++
			m.Lines[i].ID = r.lastMoveLineID
			m.Lines[i].MoveID = m.ID
			m.Lines[i].Balance = m.Lines[i].Debit - m.Lines[i].Credit
			m.Lines[i].CreatedAt = now
			m.Lines[i].UpdatedAt = now
			r.moveLines[m.Lines[i].ID] = &m.Lines[i]
		}
	}

	clone := *m
	r.moves[m.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteMove(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	m, exists := r.moves[id]
	if !exists || !m.Active {
		return platformerrors.NotFound(fmt.Sprintf("account move with id %d not found", id))
	}

	if m.State == accounting.MoveStatePosted {
		return platformerrors.Conflict("cannot delete a posted move; cancel or reverse it instead")
	}

	m.Active = false
	m.Audit.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListMoves(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[accounting.AccountMove], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []accounting.AccountMove
	for _, m := range r.moves {
		if !m.Active {
			continue
		}
		clone := *m
		list = append(list, clone)
	}

	// Sort by date descending, then ID descending
	sort.Slice(list, func(i, j int) bool {
		if list[i].Date.Equal(list[j].Date) {
			return list[i].ID > list[j].ID
		}
		return list[i].Date.After(list[j].Date)
	})

	total := int64(len(list))
	offset := page.Offset()
	limit := page.LimitClamped()

	var items []accounting.AccountMove
	if offset < len(list) {
		end := offset + limit
		if end > len(list) {
			end = len(list)
		}
		items = list[offset:end]
	}

	return pagination.NewPageResult(items, total, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Financial Reports
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) GetTrialBalance(ctx context.Context, fromDate, toDate time.Time, onlyPosted bool) (*accounting.TrialBalanceReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	report := &accounting.TrialBalanceReport{
		FromDate: fromDate,
		ToDate:   toDate,
		Lines:    make([]accounting.TrialBalanceLine, 0),
	}

	// Group balances by Account ID
	type accBal struct {
		initial float64
		debit   float64
		credit  float64
	}
	accMap := make(map[int64]*accBal)

	for _, m := range r.moves {
		if !m.Active {
			continue
		}
		if onlyPosted && m.State != accounting.MoveStatePosted {
			continue
		}

		for _, l := range r.moveLines {
			if l.MoveID != m.ID {
				continue
			}
			if _, ok := accMap[l.AccountID]; !ok {
				accMap[l.AccountID] = &accBal{}
			}

			if !fromDate.IsZero() && m.Date.Before(fromDate) {
				accMap[l.AccountID].initial += (l.Debit - l.Credit)
			} else if (fromDate.IsZero() || !m.Date.Before(fromDate)) && (toDate.IsZero() || !m.Date.After(toDate)) {
				accMap[l.AccountID].debit += l.Debit
				accMap[l.AccountID].credit += l.Credit
			}
		}
	}

	// Construct report lines
	for _, a := range r.accounts {
		if !a.Active {
			continue
		}
		bal := accMap[a.ID]
		if bal == nil {
			bal = &accBal{}
		}

		endBal := bal.initial + bal.debit - bal.credit
		report.Lines = append(report.Lines, accounting.TrialBalanceLine{
			AccountID:      a.ID,
			AccountCode:    a.Code,
			AccountName:    a.Name,
			AccountType:    a.Type,
			InitialBalance: bal.initial,
			Debit:          bal.debit,
			Credit:         bal.credit,
			EndingBalance:  endBal,
		})
	}

	sort.Slice(report.Lines, func(i, j int) bool {
		return report.Lines[i].AccountCode < report.Lines[j].AccountCode
	})

	report.ComputeTotals()
	return report, nil
}

func (r *MemoryRepo) GetProfitAndLoss(ctx context.Context, fromDate, toDate time.Time) (*accounting.ProfitAndLossReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	report := &accounting.ProfitAndLossReport{
		FromDate:     fromDate,
		ToDate:       toDate,
		IncomeLines:  make([]accounting.ReportLine, 0),
		ExpenseLines: make([]accounting.ReportLine, 0),
	}

	incomeMap := make(map[int64]float64)
	expenseMap := make(map[int64]float64)

	for _, m := range r.moves {
		if !m.Active || m.State != accounting.MoveStatePosted {
			continue
		}
		if (!fromDate.IsZero() && m.Date.Before(fromDate)) || (!toDate.IsZero() && m.Date.After(toDate)) {
			continue
		}

		for _, l := range r.moveLines {
			if l.MoveID != m.ID {
				continue
			}
			acc := r.accounts[l.AccountID]
			if acc == nil || !acc.Active {
				continue
			}

			if acc.Type.IsIncome() {
				// Income normal balance is Credit: Credit - Debit
				incomeMap[acc.ID] += (l.Credit - l.Debit)
			} else if acc.Type.IsExpense() {
				// Expense normal balance is Debit: Debit - Credit
				expenseMap[acc.ID] += (l.Debit - l.Credit)
			}
		}
	}

	for accID, amt := range incomeMap {
		acc := r.accounts[accID]
		report.IncomeLines = append(report.IncomeLines, accounting.ReportLine{
			AccountID:   acc.ID,
			AccountCode: acc.Code,
			AccountName: acc.Name,
			AccountType: acc.Type,
			Amount:      amt,
		})
	}
	sort.Slice(report.IncomeLines, func(i, j int) bool {
		return report.IncomeLines[i].AccountCode < report.IncomeLines[j].AccountCode
	})

	for accID, amt := range expenseMap {
		acc := r.accounts[accID]
		report.ExpenseLines = append(report.ExpenseLines, accounting.ReportLine{
			AccountID:   acc.ID,
			AccountCode: acc.Code,
			AccountName: acc.Name,
			AccountType: acc.Type,
			Amount:      amt,
		})
	}
	sort.Slice(report.ExpenseLines, func(i, j int) bool {
		return report.ExpenseLines[i].AccountCode < report.ExpenseLines[j].AccountCode
	})

	report.ComputeTotals()
	return report, nil
}

func (r *MemoryRepo) GetBalanceSheet(ctx context.Context, asOfDate time.Time) (*accounting.BalanceSheetReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if asOfDate.IsZero() {
		asOfDate = time.Now().UTC()
	}

	report := &accounting.BalanceSheetReport{
		AsOfDate:       asOfDate,
		AssetLines:     make([]accounting.ReportLine, 0),
		LiabilityLines: make([]accounting.ReportLine, 0),
		EquityLines:    make([]accounting.ReportLine, 0),
	}

	assetMap := make(map[int64]float64)
	liabilityMap := make(map[int64]float64)
	equityMap := make(map[int64]float64)
	var retainedEarnings float64

	for _, m := range r.moves {
		if !m.Active || m.State != accounting.MoveStatePosted {
			continue
		}
		if m.Date.After(asOfDate) {
			continue
		}

		for _, l := range r.moveLines {
			if l.MoveID != m.ID {
				continue
			}
			acc := r.accounts[l.AccountID]
			if acc == nil || !acc.Active {
				continue
			}

			if acc.Type.IsAsset() {
				// Asset normal balance is Debit: Debit - Credit
				assetMap[acc.ID] += (l.Debit - l.Credit)
			} else if acc.Type.IsLiability() {
				// Liability normal balance is Credit: Credit - Debit
				liabilityMap[acc.ID] += (l.Credit - l.Debit)
			} else if acc.Type.IsEquity() {
				// Equity normal balance is Credit: Credit - Debit
				equityMap[acc.ID] += (l.Credit - l.Debit)
			} else if acc.Type.IsIncome() {
				retainedEarnings += (l.Credit - l.Debit)
			} else if acc.Type.IsExpense() {
				retainedEarnings -= (l.Debit - l.Credit)
			}
		}
	}

	for accID, amt := range assetMap {
		acc := r.accounts[accID]
		report.AssetLines = append(report.AssetLines, accounting.ReportLine{
			AccountID:   acc.ID,
			AccountCode: acc.Code,
			AccountName: acc.Name,
			AccountType: acc.Type,
			Amount:      amt,
		})
	}
	sort.Slice(report.AssetLines, func(i, j int) bool {
		return report.AssetLines[i].AccountCode < report.AssetLines[j].AccountCode
	})

	for accID, amt := range liabilityMap {
		acc := r.accounts[accID]
		report.LiabilityLines = append(report.LiabilityLines, accounting.ReportLine{
			AccountID:   acc.ID,
			AccountCode: acc.Code,
			AccountName: acc.Name,
			AccountType: acc.Type,
			Amount:      amt,
		})
	}
	sort.Slice(report.LiabilityLines, func(i, j int) bool {
		return report.LiabilityLines[i].AccountCode < report.LiabilityLines[j].AccountCode
	})

	for accID, amt := range equityMap {
		acc := r.accounts[accID]
		report.EquityLines = append(report.EquityLines, accounting.ReportLine{
			AccountID:   acc.ID,
			AccountCode: acc.Code,
			AccountName: acc.Name,
			AccountType: acc.Type,
			Amount:      amt,
		})
	}
	sort.Slice(report.EquityLines, func(i, j int) bool {
		return report.EquityLines[i].AccountCode < report.EquityLines[j].AccountCode
	})

	report.RetainedEarnings = retainedEarnings
	report.ComputeTotals()
	return report, nil
}

func (r *MemoryRepo) GetGeneralLedger(ctx context.Context, accountID *int64, partnerID *int64, fromDate, toDate *time.Time) ([]accounting.GeneralLedgerItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var items []accounting.GeneralLedgerItem

	for _, m := range r.moves {
		if !m.Active || m.State != accounting.MoveStatePosted {
			continue
		}
		if fromDate != nil && !fromDate.IsZero() && m.Date.Before(*fromDate) {
			continue
		}
		if toDate != nil && !toDate.IsZero() && m.Date.After(*toDate) {
			continue
		}

		for _, l := range r.moveLines {
			if l.MoveID != m.ID {
				continue
			}
			if accountID != nil && l.AccountID != *accountID {
				continue
			}
			if partnerID != nil && (l.PartnerID == nil || *l.PartnerID != *partnerID) {
				continue
			}

			acc := r.accounts[l.AccountID]
			accCode := ""
			if acc != nil {
				accCode = acc.Code
			}

			items = append(items, accounting.GeneralLedgerItem{
				Date:        m.Date,
				MoveID:      m.ID,
				MoveName:    m.Name,
				LineID:      l.ID,
				AccountID:   l.AccountID,
				AccountCode: accCode,
				PartnerID:   l.PartnerID,
				Label:       l.Name,
				Debit:       l.Debit,
				Credit:      l.Credit,
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Date.Equal(items[j].Date) {
			return items[i].LineID < items[j].LineID
		}
		return items[i].Date.Before(items[j].Date)
	})

	var runningBal float64
	for i := range items {
		runningBal += (items[i].Debit - items[i].Credit)
		items[i].RunningBalance = runningBal
	}

	return items, nil
}
