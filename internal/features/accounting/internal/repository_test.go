package internal

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"cashflow_backend/internal/features/accounting"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

func TestPostMoveAtomicCommitsSequenceMoveLinesAndAuditTogether(t *testing.T) {
	moveID, journalID, periodID, actorID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	moveDate := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	tx := newFakeTx()
	tx.row = func(query string) pgx.Row {
		switch {
		case strings.Contains(query, "SELECT state FROM fiscal_periods"):
			return fakeRow{values: []any{accounting.PeriodStateOpen}}
		case strings.Contains(query, "SELECT sequence_prefix"):
			return fakeRow{values: []any{"GEN/", int64(7)}}
		case strings.Contains(query, "SELECT state, journal_id"):
			return fakeRow{values: []any{accounting.MoveStateDraft, journalID, periodID, moveDate}}
		case strings.Contains(query, "COALESCE($1::jsonb::text"):
			return fakeRow{values: []any{""}}
		case strings.Contains(query, "SELECT current_log_hash"):
			return fakeRow{err: pgx.ErrNoRows}
		default:
			return fakeRow{err: errors.New("unexpected query: " + query)}
		}
	}

	move := &accounting.AccountMove{
		ID:        moveID,
		JournalID: journalID,
		PeriodID:  periodID,
		Date:      moveDate,
		State:     accounting.MoveStateDraft,
		PostedBy:  &actorID,
		Lines: []accounting.AccountMoveLine{
			{ID: uuid.New(), AccountID: uuid.New(), Debit: decimal.NewFromInt(25)},
			{ID: uuid.New(), AccountID: uuid.New(), Credit: decimal.NewFromInt(25)},
		},
	}
	audit := &accounting.FinancialAuditLog{
		EntityType: "account_move", EntityID: moveID, Action: "POST",
		ActorID: actorID, CurrentLogHash: "audit-hash", Timestamp: moveDate,
	}

	if err := repositoryWithTx(tx).PostMoveAtomic(context.Background(), move, audit); err != nil {
		t.Fatalf("PostMoveAtomic() error = %v", err)
	}
	if !tx.committed || tx.rolledBack {
		t.Fatalf("expected committed transaction, committed=%v rolledBack=%v", tx.committed, tx.rolledBack)
	}
	if move.Name != "GEN/00007" || move.State != accounting.MoveStatePosted || move.HashSignature == "" {
		t.Fatalf("posted move was not updated after commit: %+v", move)
	}
	foundJournalLock := false
	for _, query := range tx.rowQueries {
		if strings.Contains(query, "SELECT sequence_prefix") && strings.Contains(query, "FOR UPDATE") {
			foundJournalLock = true
			break
		}
	}
	if !foundJournalLock {
		t.Fatal("journal sequence was not selected with a row lock")
	}
	if len(tx.execQueries) != 7 {
		t.Fatalf("expected sequence, post, delete, two line inserts, audit lock, and audit insert; got %d statements: %v", len(tx.execQueries), tx.execQueries)
	}
	for _, query := range tx.execQueries {
		if strings.Contains(strings.ToUpper(query), "UPDATE ACCOUNT_MOVE_LINES") {
			t.Fatalf("posted move lines must not be updated: %s", query)
		}
	}
}

func TestPostNewMoveWithEventAtomicCommitsMoveAndIdempotencyRecordTogether(t *testing.T) {
	moveID, journalID, periodID := uuid.New(), uuid.New(), uuid.New()
	moveDate := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	tx := newFakeTx()
	tx.row = func(query string) pgx.Row {
		switch {
		case strings.Contains(query, "SELECT state FROM fiscal_periods"):
			return fakeRow{values: []any{accounting.PeriodStateOpen}}
		case strings.Contains(query, "SELECT sequence_prefix"):
			return fakeRow{values: []any{"GEN/", int64(9)}}
		case strings.Contains(query, "SELECT state, journal_id"):
			return fakeRow{values: []any{accounting.MoveStateDraft, journalID, periodID, moveDate}}
		default:
			return fakeRow{err: errors.New("unexpected query: " + query)}
		}
	}
	move := &accounting.AccountMove{
		ID: moveID, JournalID: journalID, PeriodID: periodID, Date: moveDate,
		State: accounting.MoveStateDraft,
		Lines: []accounting.AccountMoveLine{
			{ID: uuid.New(), AccountID: uuid.New(), Debit: decimal.NewFromInt(12)},
			{ID: uuid.New(), AccountID: uuid.New(), Credit: decimal.NewFromInt(12)},
		},
	}
	const key, eventType = "event-key", "goods_delivered"
	err := repositoryWithTx(tx).PostNewMoveWithEventAtomic(context.Background(), move, nil, key, eventType)
	if err != nil {
		t.Fatalf("PostNewMoveWithEventAtomic() error = %v", err)
	}
	if !tx.committed || tx.rolledBack {
		t.Fatalf("expected committed transaction, committed=%v rolledBack=%v", tx.committed, tx.rolledBack)
	}
	if move.State != accounting.MoveStatePosted || move.Name != "GEN/00009" {
		t.Fatalf("new move was not posted after commit: %+v", move)
	}
	eventInsert := -1
	for i, query := range tx.execQueries {
		if strings.Contains(query, "INSERT INTO subledger_processed_events") {
			eventInsert = i
			break
		}
	}
	if eventInsert == -1 {
		t.Fatalf("processed event was not inserted: %v", tx.execQueries)
	}
	args := tx.execArgs[eventInsert]
	if len(args) != 3 || args[0] != key || args[1] != eventType || args[2] != moveID {
		t.Fatalf("processed event arguments = %v", args)
	}
}

func TestPostNewMoveWithEventAtomicRollsBackWhenEventInsertFails(t *testing.T) {
	moveID, journalID, periodID := uuid.New(), uuid.New(), uuid.New()
	moveDate := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	tx := newFakeTx()
	tx.row = func(query string) pgx.Row {
		switch {
		case strings.Contains(query, "SELECT state FROM fiscal_periods"):
			return fakeRow{values: []any{accounting.PeriodStateOpen}}
		case strings.Contains(query, "SELECT sequence_prefix"):
			return fakeRow{values: []any{"GEN/", int64(1)}}
		case strings.Contains(query, "SELECT state, journal_id"):
			return fakeRow{values: []any{accounting.MoveStateDraft, journalID, periodID, moveDate}}
		case strings.Contains(query, "COALESCE($1::jsonb::text"):
			return fakeRow{values: []any{""}}
		case strings.Contains(query, "SELECT current_log_hash"):
			return fakeRow{err: pgx.ErrNoRows}
		default:
			return fakeRow{err: errors.New("unexpected query: " + query)}
		}
	}
	tx.exec = func(query string) (pgconn.CommandTag, error) {
		if strings.Contains(query, "INSERT INTO subledger_processed_events") {
			return pgconn.CommandTag{}, errors.New("idempotency key already exists")
		}
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	move := &accounting.AccountMove{
		ID: moveID, JournalID: journalID, PeriodID: periodID, Date: moveDate,
		State: accounting.MoveStateDraft,
		Lines: []accounting.AccountMoveLine{
			{ID: uuid.New(), AccountID: uuid.New(), Debit: decimal.NewFromInt(12)},
			{ID: uuid.New(), AccountID: uuid.New(), Credit: decimal.NewFromInt(12)},
		},
	}
	err := repositoryWithTx(tx).PostNewMoveWithEventAtomic(context.Background(), move, nil, "duplicate-key", "payment_received")
	if err == nil {
		t.Fatal("PostNewMoveWithEventAtomic() expected an event insertion error")
	}
	if tx.committed || !tx.rolledBack {
		t.Fatalf("expected transaction rollback, committed=%v rolledBack=%v", tx.committed, tx.rolledBack)
	}
	if move.State != accounting.MoveStateDraft || move.Name != "" {
		t.Fatalf("failed transaction mutated move: %+v", move)
	}
}

func TestPostMoveAtomicRollsBackOnAuditInsertFailure(t *testing.T) {
	moveID, journalID, periodID := uuid.New(), uuid.New(), uuid.New()
	moveDate := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	tx := newFakeTx()
	tx.row = func(query string) pgx.Row {
		switch {
		case strings.Contains(query, "SELECT state FROM fiscal_periods"):
			return fakeRow{values: []any{accounting.PeriodStateOpen}}
		case strings.Contains(query, "SELECT sequence_prefix"):
			return fakeRow{values: []any{"GEN/", int64(1)}}
		case strings.Contains(query, "SELECT state, journal_id"):
			return fakeRow{values: []any{accounting.MoveStateDraft, journalID, periodID, moveDate}}
		default:
			return fakeRow{err: errors.New("unexpected query: " + query)}
		}
	}
	tx.exec = func(query string) (pgconn.CommandTag, error) {
		if strings.Contains(query, "INSERT INTO financial_audit_logs") {
			return pgconn.CommandTag{}, errors.New("audit insert failed")
		}
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	move := &accounting.AccountMove{
		ID: moveID, JournalID: journalID, PeriodID: periodID, Date: moveDate,
		State: accounting.MoveStateDraft,
		Lines: []accounting.AccountMoveLine{
			{ID: uuid.New(), AccountID: uuid.New(), Debit: decimal.NewFromInt(1)},
			{ID: uuid.New(), AccountID: uuid.New(), Credit: decimal.NewFromInt(1)},
		},
	}
	audit := &accounting.FinancialAuditLog{
		EntityType: "account_move", EntityID: moveID, Action: "POST",
		ActorID: uuid.New(), CurrentLogHash: "audit-hash", Timestamp: moveDate,
	}

	err := repositoryWithTx(tx).PostMoveAtomic(context.Background(), move, audit)
	if err == nil {
		t.Fatal("PostMoveAtomic() expected an audit insertion error")
	}
	if tx.committed || !tx.rolledBack {
		t.Fatalf("expected transaction rollback, committed=%v rolledBack=%v", tx.committed, tx.rolledBack)
	}
	if move.State != accounting.MoveStateDraft || move.Name != "" {
		t.Fatalf("failed post mutated move: %+v", move)
	}
}

func TestPostMoveAtomicRollsBackOnLineInsertFailure(t *testing.T) {
	moveID, journalID, periodID := uuid.New(), uuid.New(), uuid.New()
	moveDate := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	tx := newFakeTx()
	tx.row = func(query string) pgx.Row {
		switch {
		case strings.Contains(query, "SELECT state FROM fiscal_periods"):
			return fakeRow{values: []any{accounting.PeriodStateOpen}}
		case strings.Contains(query, "SELECT sequence_prefix"):
			return fakeRow{values: []any{"GEN/", int64(1)}}
		case strings.Contains(query, "SELECT state, journal_id"):
			return fakeRow{values: []any{accounting.MoveStateDraft, journalID, periodID, moveDate}}
		default:
			return fakeRow{err: errors.New("unexpected query: " + query)}
		}
	}
	tx.exec = func(query string) (pgconn.CommandTag, error) {
		if strings.Contains(query, "INSERT INTO account_move_lines") {
			return pgconn.CommandTag{}, errors.New("line insert failed")
		}
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	move := &accounting.AccountMove{
		ID: moveID, JournalID: journalID, PeriodID: periodID, Date: moveDate,
		State: accounting.MoveStateDraft,
		Lines: []accounting.AccountMoveLine{
			{ID: uuid.New(), AccountID: uuid.New(), Debit: decimal.NewFromInt(1)},
			{ID: uuid.New(), AccountID: uuid.New(), Credit: decimal.NewFromInt(1)},
		},
	}

	err := repositoryWithTx(tx).PostMoveAtomic(context.Background(), move, nil)
	if err == nil {
		t.Fatal("PostMoveAtomic() expected an insertion error")
	}
	if tx.committed || !tx.rolledBack {
		t.Fatalf("expected transaction rollback, committed=%v rolledBack=%v", tx.committed, tx.rolledBack)
	}
	if move.State != accounting.MoveStateDraft || move.Name != "" || move.HashSignature != "" {
		t.Fatalf("failed post mutated move: %+v", move)
	}
}

func TestReverseMoveAtomicPersistsContraEntryAtomically(t *testing.T) {
	originalID, journalID, periodID := uuid.New(), uuid.New(), uuid.New()
	firstAccount, secondAccount := uuid.New(), uuid.New()
	moveDate := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	tx := newFakeTx()
	tx.row = func(query string) pgx.Row {
		switch {
		case strings.Contains(query, "SELECT state FROM account_moves"):
			return fakeRow{values: []any{accounting.MoveStatePosted}}
		case strings.Contains(query, "SELECT EXISTS"):
			return fakeRow{values: []any{false}}
		case strings.Contains(query, "SELECT state FROM fiscal_periods"):
			return fakeRow{values: []any{accounting.PeriodStateOpen}}
		case strings.Contains(query, "SELECT sequence_prefix"):
			return fakeRow{values: []any{"GEN/", int64(8)}}
		default:
			return fakeRow{err: errors.New("unexpected query: " + query)}
		}
	}
	tx.rows = func(query string) pgx.Rows {
		return &fakeRows{values: [][]any{
			{uuid.New(), originalID, firstAccount, pgtype.UUID{}, decimal.NewFromInt(30), decimal.Zero, decimal.NewFromInt(30), "SAR", pgtype.UUID{}, "", moveDate, moveDate},
			{uuid.New(), originalID, secondAccount, pgtype.UUID{}, decimal.Zero, decimal.NewFromInt(30), decimal.NewFromInt(-30), "SAR", pgtype.UUID{}, "", moveDate, moveDate},
		}}
	}

	reversal := &accounting.AccountMove{
		JournalID: journalID, PeriodID: periodID, Date: moveDate,
		State: accounting.MoveStateDraft,
	}
	err := repositoryWithTx(tx).ReverseMoveAtomic(context.Background(), originalID, reversal, nil)
	if err != nil {
		t.Fatalf("ReverseMoveAtomic() error = %v", err)
	}
	if !tx.committed || tx.rolledBack {
		t.Fatalf("expected committed transaction, committed=%v rolledBack=%v", tx.committed, tx.rolledBack)
	}
	if reversal.State != accounting.MoveStatePosted || reversal.Name != "GEN/00008" || len(reversal.Lines) != 2 {
		t.Fatalf("reversal aggregate not updated after commit: %+v", reversal)
	}
	if reversal.Lines[0].Debit.String() != "0" || reversal.Lines[0].Credit.String() != "30" ||
		reversal.Lines[1].Debit.String() != "30" || reversal.Lines[1].Credit.String() != "0" {
		t.Fatalf("reversal lines do not swap debit and credit: %+v", reversal.Lines)
	}
	lineInsertCount := 0
	for i, query := range tx.execQueries {
		if !strings.Contains(query, "INSERT INTO account_move_lines") {
			continue
		}
		lineInsertCount++
		if got := tx.execArgs[i][1]; got != reversal.ID {
			t.Fatalf("reversal line %d was attached to move %v, want %v", lineInsertCount, got, reversal.ID)
		}
	}
	if lineInsertCount != 2 {
		t.Fatalf("expected two persisted reversal lines, got %d", lineInsertCount)
	}
}

func TestReportQueriesRejectReversedDateRange(t *testing.T) {
	repository := &PostgresRepository{}
	fromDate := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	toDate := fromDate.AddDate(0, 0, -1)
	if _, err := repository.GetTrialBalanceData(context.Background(), fromDate, toDate); err == nil {
		t.Fatal("GetTrialBalanceData() expected invalid date range error")
	}
	if _, err := repository.GetGeneralLedgerLines(context.Background(), uuid.New(), fromDate, toDate); err == nil {
		t.Fatal("GetGeneralLedgerLines() expected invalid date range error")
	}
}

func TestGetFXMonetaryBalancesReturnsSignedCurrencyBalances(t *testing.T) {
	accountID, partnerID, costCenterID := uuid.New(), uuid.New(), uuid.New()
	rows := &fakeRows{values: [][]any{{
		accountID, accounting.AccountTypeLiability,
		pgtype.UUID{Bytes: partnerID, Valid: true},
		pgtype.UUID{Bytes: costCenterID, Valid: true},
		"USD", decimal.NewFromInt(-100), decimal.NewFromInt(-350),
	}}}
	pool := &fakeQueryPool{rows: rows}
	repository := &PostgresRepository{pool: pool}
	asOf := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)

	balances, err := repository.GetFXMonetaryBalances(context.Background(), asOf)
	if err != nil {
		t.Fatalf("GetFXMonetaryBalances() error = %v", err)
	}
	if len(balances) != 1 {
		t.Fatalf("got %d balances, want 1", len(balances))
	}
	balance := balances[0]
	if balance.AccountID != accountID || balance.AccountType != accounting.AccountTypeLiability ||
		balance.CurrencyID != "USD" || balance.ForeignAmount.String() != "-100" ||
		balance.FunctionalCarryingAmount.String() != "-350" {
		t.Fatalf("balance = %+v", balance)
	}
	if balance.PartnerID == nil || *balance.PartnerID != partnerID ||
		balance.CostCenterID == nil || *balance.CostCenterID != costCenterID {
		t.Fatalf("balance dimensions = %+v", balance)
	}
	if !pool.queryAsOf.Equal(asOf) || !strings.Contains(pool.query, "m.state = 'posted'") ||
		!strings.Contains(pool.query, "l.date <= $1") {
		t.Fatalf("FX balance query or date parameter is incorrect: %s (%v)", pool.query, pool.queryAsOf)
	}
}

func repositoryWithTx(tx *fakeTx) *PostgresRepository {
	return &PostgresRepository{pool: &fakePool{tx: tx}}
}

type fakePool struct {
	repositoryPool
	tx *fakeTx
}

type fakeQueryPool struct {
	repositoryPool
	rows      pgx.Rows
	query     string
	queryAsOf time.Time
}

func (p *fakeQueryPool) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	p.query = query
	if len(args) > 0 {
		p.queryAsOf, _ = args[0].(time.Time)
	}
	return p.rows, nil
}

func (p *fakePool) Begin(context.Context) (pgx.Tx, error) {
	return p.tx, nil
}

type fakeTx struct {
	pgx.Tx
	row         func(string) pgx.Row
	rows        func(string) pgx.Rows
	exec        func(string) (pgconn.CommandTag, error)
	rowQueries  []string
	execQueries []string
	execArgs    [][]any
	committed   bool
	rolledBack  bool
}

func newFakeTx() *fakeTx {
	return &fakeTx{}
}

func (tx *fakeTx) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	tx.rowQueries = append(tx.rowQueries, query)
	return tx.row(query)
}

func (tx *fakeTx) Query(_ context.Context, query string, _ ...any) (pgx.Rows, error) {
	if tx.rows == nil {
		return nil, errors.New("unexpected rows query: " + query)
	}
	return tx.rows(query), nil
}

func (tx *fakeTx) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	tx.execQueries = append(tx.execQueries, query)
	tx.execArgs = append(tx.execArgs, args)
	if tx.exec != nil {
		return tx.exec(query)
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (tx *fakeTx) Commit(context.Context) error {
	tx.committed = true
	return nil
}

func (tx *fakeTx) Rollback(context.Context) error {
	if tx.committed {
		return nil
	}
	tx.rolledBack = true
	return nil
}

type fakeRow struct {
	values []any
	err    error
}

func (row fakeRow) Scan(destinations ...any) error {
	if row.err != nil {
		return row.err
	}
	if len(destinations) != len(row.values) {
		return errors.New("unexpected number of scan destinations")
	}
	for i, destination := range destinations {
		target := reflect.ValueOf(destination)
		if target.Kind() != reflect.Ptr || target.IsNil() {
			return errors.New("scan destination must be a non-nil pointer")
		}
		value := reflect.ValueOf(row.values[i])
		if !value.IsValid() {
			target.Elem().Set(reflect.Zero(target.Elem().Type()))
			continue
		}
		if value.Type().AssignableTo(target.Elem().Type()) {
			target.Elem().Set(value)
		} else if value.Type().ConvertibleTo(target.Elem().Type()) {
			target.Elem().Set(value.Convert(target.Elem().Type()))
		} else {
			return errors.New("incompatible fake scan value")
		}
	}
	return nil
}

type fakeRows struct {
	pgx.Rows
	values [][]any
	index  int
}

func (rows *fakeRows) Next() bool {
	if rows.index >= len(rows.values) {
		return false
	}
	rows.index++
	return true
}

func (rows *fakeRows) Scan(destinations ...any) error {
	return fakeRow{values: rows.values[rows.index-1]}.Scan(destinations...)
}

func (*fakeRows) Err() error { return nil }
func (*fakeRows) Close()     {}
