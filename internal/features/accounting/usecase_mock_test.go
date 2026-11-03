package accounting

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type mockAccountingRepository struct {
	accountsByID     map[uuid.UUID]*Account
	accountsByCode   map[string]*Account
	accounts         []*Account
	period           *FiscalPeriod
	periodsByID      map[uuid.UUID]*FiscalPeriod
	journal          *Journal
	move             *AccountMove
	trialBalance     []*TrialBalanceRow
	fxBalances       []*FXMonetaryBalance
	processedMoves   map[string]*AccountMove
	createAccountErr error
	saveDraftErr     error
	postMoveErr      error
	postNewMoveErr   error
	reverseMoveErr   error
	postCalls        int
	postNewCalls     int
	draftCalls       int
	reverseCalls     int
}

func newMockAccountingRepository() *mockAccountingRepository {
	return &mockAccountingRepository{
		accountsByID:   make(map[uuid.UUID]*Account),
		accountsByCode: make(map[string]*Account),
		periodsByID:    make(map[uuid.UUID]*FiscalPeriod),
		processedMoves: make(map[string]*AccountMove),
	}
}

func (r *mockAccountingRepository) CreateAccount(_ context.Context, account *Account) error {
	if r.createAccountErr != nil {
		return r.createAccountErr
	}
	r.accountsByID[account.ID] = account
	r.accountsByCode[account.Code] = account
	r.accounts = append(r.accounts, account)
	return nil
}

func (r *mockAccountingRepository) GetAccountByID(_ context.Context, id uuid.UUID) (*Account, error) {
	account, ok := r.accountsByID[id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return account, nil
}

func (r *mockAccountingRepository) GetAccountByCode(_ context.Context, code string) (*Account, error) {
	account, ok := r.accountsByCode[code]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return account, nil
}

func (r *mockAccountingRepository) ListAccounts(context.Context) ([]*Account, error) {
	return r.accounts, nil
}

func (r *mockAccountingRepository) CreateFiscalYear(_ context.Context, _ *FiscalYear, periods []*FiscalPeriod) error {
	for _, period := range periods {
		r.periodsByID[period.ID] = period
	}
	return nil
}

func (r *mockAccountingRepository) GetPeriodByDate(context.Context, time.Time) (*FiscalPeriod, error) {
	if r.period == nil {
		return nil, ErrPeriodNotFound
	}
	return r.period, nil
}

func (r *mockAccountingRepository) GetPeriodByID(_ context.Context, id uuid.UUID) (*FiscalPeriod, error) {
	period, ok := r.periodsByID[id]
	if !ok && r.period != nil && r.period.ID == id {
		return r.period, nil
	}
	if !ok {
		return nil, ErrPeriodNotFound
	}
	return period, nil
}

func (r *mockAccountingRepository) UpdatePeriodState(_ context.Context, id uuid.UUID, state PeriodState, actorID uuid.UUID) error {
	period, ok := r.periodsByID[id]
	if !ok && r.period != nil && r.period.ID == id {
		period = r.period
	}
	if period == nil {
		return ErrPeriodNotFound
	}
	period.State = state
	if state == PeriodStateOpen {
		period.ClosedAt, period.ClosedBy = nil, nil
	} else {
		now := time.Now().UTC()
		period.ClosedAt, period.ClosedBy = &now, &actorID
	}
	return nil
}

func (r *mockAccountingRepository) GetJournalByID(context.Context, uuid.UUID) (*Journal, error) {
	if r.journal == nil {
		return nil, ErrJournalNotFound
	}
	return r.journal, nil
}

func (r *mockAccountingRepository) GetNextSequenceWithLock(context.Context, pgx.Tx, uuid.UUID) (string, error) {
	return "", errors.New("not implemented")
}

func (r *mockAccountingRepository) SaveDraftMove(_ context.Context, move *AccountMove) error {
	r.draftCalls++
	if r.saveDraftErr != nil {
		return r.saveDraftErr
	}
	move.State = MoveStateDraft
	r.move = move
	return nil
}

func (r *mockAccountingRepository) GetMoveByID(context.Context, uuid.UUID) (*AccountMove, error) {
	if r.move == nil {
		return nil, ErrMoveNotFound
	}
	return r.move, nil
}

func (r *mockAccountingRepository) PostMoveAtomic(_ context.Context, move *AccountMove, _ *FinancialAuditLog) error {
	r.postCalls++
	if r.postMoveErr != nil {
		return r.postMoveErr
	}
	move.State = MoveStatePosted
	move.Name = "GEN/00001"
	return nil
}

func (r *mockAccountingRepository) PostNewMoveWithEventAtomic(
	_ context.Context, move *AccountMove, _ *FinancialAuditLog, key, _ string,
) error {
	r.postNewCalls++
	if r.postNewMoveErr != nil {
		return r.postNewMoveErr
	}
	move.State = MoveStatePosted
	move.Name = "GEN/00001"
	r.processedMoves[key] = move
	return nil
}

func (r *mockAccountingRepository) ReverseMoveAtomic(_ context.Context, _ uuid.UUID, move *AccountMove, _ *FinancialAuditLog) error {
	r.reverseCalls++
	if r.reverseMoveErr != nil {
		return r.reverseMoveErr
	}
	move.State = MoveStatePosted
	move.Name = "GEN/00002"
	return nil
}

func (r *mockAccountingRepository) GetTrialBalanceData(context.Context, time.Time, time.Time) ([]*TrialBalanceRow, error) {
	return r.trialBalance, nil
}

func (r *mockAccountingRepository) GetGeneralLedgerLines(context.Context, uuid.UUID, time.Time, time.Time) ([]*GeneralLedgerLineRow, error) {
	return nil, nil
}

func (r *mockAccountingRepository) GetFXMonetaryBalances(context.Context, time.Time) ([]*FXMonetaryBalance, error) {
	return r.fxBalances, nil
}

func (r *mockAccountingRepository) GetProcessedEventMove(_ context.Context, key string) (*AccountMove, error) {
	return r.processedMoves[key], nil
}

func (r *mockAccountingRepository) SaveProcessedEvent(_ context.Context, key, _ string, moveID uuid.UUID) error {
	if r.move == nil || r.move.ID != moveID {
		return ErrMoveNotFound
	}
	r.processedMoves[key] = r.move
	return nil
}
