package bankstatementstorage

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"cashflow_backend/internal/domain/bankstatement"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is a thread-safe in-memory implementation of bankstatement.Repository.
type MemoryRepo struct {
	mu          sync.RWMutex
	statements  map[int64]*bankstatement.BankStatement
	lines       map[int64]*bankstatement.BankStatementLine
	partials    map[int64]*bankstatement.PartialReconcile
	fullMatches map[string]*bankstatement.FullReconcile
	models      map[int64]*bankstatement.ReconcileModel
	roundings   map[int64]*bankstatement.CashRounding

	lastStatementID int64
	lastLineID      int64
	lastPartialID   int64
	lastFullID      int64
	lastModelID     int64
	lastRoundingID  int64
}

// NewMemoryRepo creates an empty bank statement repository.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		statements:  make(map[int64]*bankstatement.BankStatement),
		lines:       make(map[int64]*bankstatement.BankStatementLine),
		partials:    make(map[int64]*bankstatement.PartialReconcile),
		fullMatches: make(map[string]*bankstatement.FullReconcile),
		models:      make(map[int64]*bankstatement.ReconcileModel),
		roundings:   make(map[int64]*bankstatement.CashRounding),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Statements
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateStatement(ctx context.Context, st *bankstatement.BankStatement) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastStatementID++
	st.ID = r.lastStatementID
	now := time.Now().UTC()
	st.Audit.CreatedAt = now
	st.Audit.UpdatedAt = now
	st.Active = true
	if st.State == "" {
		st.State = bankstatement.StatementStateOpen
	}
	if st.Currency == "" {
		st.Currency = "USD"
	}
	st.Lines = nil

	clone := *st
	r.statements[st.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetStatementByID(ctx context.Context, id int64) (*bankstatement.BankStatement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	st, exists := r.statements[id]
	if !exists || !st.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("bank statement with id %d not found", id))
	}
	clone := *st
	return &clone, nil
}

func (r *MemoryRepo) GetStatementWithLines(ctx context.Context, id int64) (*bankstatement.BankStatement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	st, exists := r.statements[id]
	if !exists || !st.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("bank statement with id %d not found", id))
	}
	clone := *st
	var lines []bankstatement.BankStatementLine
	for _, l := range r.lines {
		if l.StatementID == id {
			lines = append(lines, *l)
		}
	}
	sort.Slice(lines, func(i, j int) bool {
		if lines[i].Sequence == lines[j].Sequence {
			return lines[i].ID < lines[j].ID
		}
		return lines[i].Sequence < lines[j].Sequence
	})
	clone.Lines = lines
	return &clone, nil
}

func (r *MemoryRepo) UpdateStatement(ctx context.Context, st *bankstatement.BankStatement) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.statements[st.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound(fmt.Sprintf("bank statement with id %d not found", st.ID))
	}
	st.Audit.CreatedAt = existing.Audit.CreatedAt
	st.Audit.UpdatedAt = time.Now().UTC()
	st.Active = true
	st.Lines = nil

	clone := *st
	r.statements[st.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteStatement(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	st, exists := r.statements[id]
	if !exists || !st.Active {
		return platformerrors.NotFound(fmt.Sprintf("bank statement with id %d not found", id))
	}
	st.Active = false

	// Remove lines of the statement (mirrors ON DELETE CASCADE).
	for _, l := range r.lines {
		if l.StatementID == id {
			delete(r.lines, l.ID)
		}
	}
	return nil
}

func (r *MemoryRepo) ListStatements(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[bankstatement.BankStatement], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []bankstatement.BankStatement
	for _, st := range r.statements {
		if !st.Active {
			continue
		}
		clone := *st
		list = append(list, clone)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Date.Equal(list[j].Date) {
			return list[i].ID > list[j].ID
		}
		return list[i].Date.After(list[j].Date)
	})

	total := int64(len(list))
	offset := page.Offset()
	limit := page.LimitClamped()
	var items []bankstatement.BankStatement
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
// Statement Lines
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) AddStatementLines(ctx context.Context, statementID int64, lines []bankstatement.BankStatementLine) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	for i := range lines {
		r.lastLineID++
		lines[i].ID = r.lastLineID
		lines[i].StatementID = statementID
		lines[i].CreatedAt = now
		lines[i].UpdatedAt = now
		lines[i].Amount = roundAmount(lines[i].Amount)
		clone := lines[i]
		r.lines[clone.ID] = &clone
	}
	return nil
}

func (r *MemoryRepo) GetStatementLineByID(ctx context.Context, id int64) (*bankstatement.BankStatementLine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	l, exists := r.lines[id]
	if !exists {
		return nil, platformerrors.NotFound(fmt.Sprintf("bank statement line with id %d not found", id))
	}
	clone := *l
	return &clone, nil
}

func (r *MemoryRepo) UpdateStatementLine(ctx context.Context, line *bankstatement.BankStatementLine) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.lines[line.ID]
	if !exists {
		return platformerrors.NotFound(fmt.Sprintf("bank statement line with id %d not found", line.ID))
	}
	line.CreatedAt = existing.CreatedAt
	line.UpdatedAt = time.Now().UTC()
	clone := *line
	r.lines[line.ID] = &clone
	return nil
}

func (r *MemoryRepo) UpdateStatementLineReconcileState(ctx context.Context, lineID int64, reconciled bool, residual float64, matchingNumber *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	l, exists := r.lines[lineID]
	if !exists {
		return platformerrors.NotFound(fmt.Sprintf("bank statement line with id %d not found", lineID))
	}
	l.Reconciled = reconciled
	l.AmountResidual = residual
	l.MatchingNumber = matchingNumber
	l.UpdatedAt = time.Now().UTC()
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Reconciles
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreatePartialReconcile(ctx context.Context, pr *bankstatement.PartialReconcile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastPartialID++
	pr.ID = r.lastPartialID
	now := time.Now().UTC()
	pr.CreatedAt = now
	pr.UpdatedAt = now
	if pr.Currency == "" {
		pr.Currency = "USD"
	}
	clone := *pr
	r.partials[pr.ID] = &clone
	return nil
}

func (r *MemoryRepo) ListPartialReconcilesByLine(ctx context.Context, lineID int64) ([]bankstatement.PartialReconcile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []bankstatement.PartialReconcile
	for _, pr := range r.partials {
		if pr.DebitLineID == lineID || pr.CreditLineID == lineID {
			result = append(result, *pr)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (r *MemoryRepo) DeletePartialReconcile(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.partials[id]; !exists {
		return platformerrors.NotFound(fmt.Sprintf("partial reconcile with id %d not found", id))
	}
	delete(r.partials, id)
	return nil
}

func (r *MemoryRepo) CreateFullReconcile(ctx context.Context, fr *bankstatement.FullReconcile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.fullMatches[fr.MatchingNumber]; exists {
		return platformerrors.Conflict(fmt.Sprintf("full reconcile with matching number '%s' already exists", fr.MatchingNumber))
	}
	r.lastFullID++
	fr.ID = r.lastFullID
	now := time.Now().UTC()
	fr.CreatedAt = now
	fr.UpdatedAt = now
	clone := *fr
	r.fullMatches[fr.MatchingNumber] = &clone
	return nil
}

func (r *MemoryRepo) GetFullReconcileByMatchingNumber(ctx context.Context, matchingNumber string) (*bankstatement.FullReconcile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fr, exists := r.fullMatches[matchingNumber]
	if !exists {
		return nil, platformerrors.NotFound(fmt.Sprintf("full reconcile '%s' not found", matchingNumber))
	}
	clone := *fr
	return &clone, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Reconcile Models
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateReconcileModel(ctx context.Context, m *bankstatement.ReconcileModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastModelID++
	m.ID = r.lastModelID
	now := time.Now().UTC()
	m.CreatedAt = now
	m.UpdatedAt = now
	if m.MatchNature == "" {
		m.MatchNature = bankstatement.MatchNatureBoth
	}
	clone := *m
	r.models[m.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetReconcileModelByID(ctx context.Context, id int64) (*bankstatement.ReconcileModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, exists := r.models[id]
	if !exists || !m.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("reconcile model with id %d not found", id))
	}
	clone := *m
	return &clone, nil
}

func (r *MemoryRepo) UpdateReconcileModel(ctx context.Context, m *bankstatement.ReconcileModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.models[m.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound(fmt.Sprintf("reconcile model with id %d not found", m.ID))
	}
	m.CreatedAt = existing.CreatedAt
	m.UpdatedAt = time.Now().UTC()
	clone := *m
	r.models[m.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteReconcileModel(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	m, exists := r.models[id]
	if !exists || !m.Active {
		return platformerrors.NotFound(fmt.Sprintf("reconcile model with id %d not found", id))
	}
	m.Active = false
	m.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListReconcileModels(ctx context.Context) ([]bankstatement.ReconcileModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []bankstatement.ReconcileModel
	for _, m := range r.models {
		if !m.Active {
			continue
		}
		list = append(list, *m)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Sequence < list[j].Sequence
	})
	return list, nil
}

func (r *MemoryRepo) ListAutoReconcileModels(ctx context.Context) ([]bankstatement.ReconcileModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []bankstatement.ReconcileModel
	for _, m := range r.models {
		if !m.Active || !m.IsAutoReconcile {
			continue
		}
		list = append(list, *m)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Sequence < list[j].Sequence
	})
	return list, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Cash Rounding
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateCashRounding(ctx context.Context, cr *bankstatement.CashRounding) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastRoundingID++
	cr.ID = r.lastRoundingID
	now := time.Now().UTC()
	cr.CreatedAt = now
	cr.UpdatedAt = now
	cr.Active = true
	clone := *cr
	r.roundings[cr.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetCashRoundingByID(ctx context.Context, id int64) (*bankstatement.CashRounding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cr, exists := r.roundings[id]
	if !exists || !cr.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("cash rounding with id %d not found", id))
	}
	clone := *cr
	return &clone, nil
}

func (r *MemoryRepo) UpdateCashRounding(ctx context.Context, cr *bankstatement.CashRounding) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.roundings[cr.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound(fmt.Sprintf("cash rounding with id %d not found", cr.ID))
	}
	cr.CreatedAt = existing.CreatedAt
	cr.UpdatedAt = time.Now().UTC()
	clone := *cr
	r.roundings[cr.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteCashRounding(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cr, exists := r.roundings[id]
	if !exists || !cr.Active {
		return platformerrors.NotFound(fmt.Sprintf("cash rounding with id %d not found", id))
	}
	cr.Active = false
	cr.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListCashRoundings(ctx context.Context) ([]bankstatement.CashRounding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []bankstatement.CashRounding
	for _, cr := range r.roundings {
		if !cr.Active {
			continue
		}
		list = append(list, *cr)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})
	return list, nil
}
