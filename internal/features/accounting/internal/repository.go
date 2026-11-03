package internal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"cashflow_backend/internal/features/accounting"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type dbtx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type repositoryPool interface {
	dbtx
	Begin(context.Context) (pgx.Tx, error)
}

type PostgresRepository struct {
	pool repositoryPool
}

var _ accounting.AccountingRepository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateAccount(ctx context.Context, acc *accounting.Account) error {
	const query = `
		INSERT INTO accounts (
			id, code, name, account_type, normal_balance, currency_id, parent_id,
			is_reconcilable, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.pool.Exec(ctx, query,
		acc.ID, acc.Code, acc.Name, acc.Type, acc.NormalBalance, acc.CurrencyID,
		acc.ParentID, acc.IsReconcilable, acc.IsActive, acc.CreatedAt, acc.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "uq_accounts_code" {
			return fmt.Errorf("%w: %s", accounting.ErrDuplicateAccountCode, acc.Code)
		}
		return fmt.Errorf("create account: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (*accounting.Account, error) {
	const query = `
		SELECT id, code, name, account_type, normal_balance, currency_id, parent_id,
			is_reconcilable, is_active, created_at, updated_at
		FROM accounts WHERE id = $1`
	account, err := scanAccount(r.pool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, accounting.ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get account by id: %w", err)
	}
	return account, nil
}

func (r *PostgresRepository) GetAccountByCode(ctx context.Context, code string) (*accounting.Account, error) {
	const query = `
		SELECT id, code, name, account_type, normal_balance, currency_id, parent_id,
			is_reconcilable, is_active, created_at, updated_at
		FROM accounts WHERE code = $1`
	account, err := scanAccount(r.pool.QueryRow(ctx, query, code))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, accounting.ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get account by code: %w", err)
	}
	return account, nil
}

func (r *PostgresRepository) ListAccounts(ctx context.Context) ([]*accounting.Account, error) {
	const query = `
		SELECT id, code, name, account_type, normal_balance, currency_id, parent_id,
			is_reconcilable, is_active, created_at, updated_at
		FROM accounts ORDER BY code ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]*accounting.Account, 0)
	for rows.Next() {
		account, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accounts: %w", err)
	}
	return accounts, nil
}

func (r *PostgresRepository) CreateFiscalYear(ctx context.Context, year *accounting.FiscalYear, periods []*accounting.FiscalPeriod) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create fiscal year: %w", err)
	}
	defer tx.Rollback(ctx)

	if year.ID == uuid.Nil {
		year.ID = uuid.New()
	}
	const yearQuery = `
		INSERT INTO fiscal_years (id, year, start_date, end_date, state, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	if _, err := tx.Exec(ctx, yearQuery, year.ID, year.Year, year.StartDate, year.EndDate, year.State, year.CreatedAt); err != nil {
		return fmt.Errorf("insert fiscal year: %w", err)
	}

	const periodQuery = `
		INSERT INTO fiscal_periods (
			id, fiscal_year_id, period_number, name, start_date, end_date, state,
			closed_at, closed_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	for _, period := range periods {
		if period.ID == uuid.Nil {
			period.ID = uuid.New()
		}
		period.FiscalYearID = year.ID
		if _, err := tx.Exec(ctx, periodQuery,
			period.ID, year.ID, period.PeriodNumber, period.Name, period.StartDate, period.EndDate,
			period.State, period.ClosedAt, period.ClosedBy, period.CreatedAt,
		); err != nil {
			return fmt.Errorf("insert fiscal period %d: %w", period.PeriodNumber, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit fiscal year: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetPeriodByDate(ctx context.Context, date time.Time) (*accounting.FiscalPeriod, error) {
	const query = `
		SELECT id, fiscal_year_id, period_number, name, start_date, end_date, state,
			closed_at, closed_by, created_at
		FROM fiscal_periods
		WHERE start_date <= $1 AND end_date >= $1
		ORDER BY start_date DESC
		LIMIT 1`
	period, err := scanPeriod(r.pool.QueryRow(ctx, query, date))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, accounting.ErrPeriodNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get fiscal period by date: %w", err)
	}
	return period, nil
}

func (r *PostgresRepository) GetPeriodByID(ctx context.Context, id uuid.UUID) (*accounting.FiscalPeriod, error) {
	const query = `
		SELECT id, fiscal_year_id, period_number, name, start_date, end_date, state,
			closed_at, closed_by, created_at
		FROM fiscal_periods WHERE id = $1`
	period, err := scanPeriod(r.pool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, accounting.ErrPeriodNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get fiscal period by id: %w", err)
	}
	return period, nil
}

func (r *PostgresRepository) UpdatePeriodState(ctx context.Context, id uuid.UUID, state accounting.PeriodState, actorID uuid.UUID) error {
	const query = `
		UPDATE fiscal_periods
		SET state = $2,
			closed_at = CASE WHEN $2 = 'open' THEN NULL ELSE NOW() END,
			closed_by = CASE WHEN $2 = 'open' THEN NULL ELSE $3 END
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id, state, actorID)
	if err != nil {
		return fmt.Errorf("update fiscal period state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return accounting.ErrPeriodNotFound
	}
	return nil
}

func (r *PostgresRepository) GetJournalByID(ctx context.Context, id uuid.UUID) (*accounting.Journal, error) {
	const query = `
		SELECT id, code, name, type, sequence_prefix, next_number, is_active, created_at
		FROM journals WHERE id = $1`
	var journal accounting.Journal
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&journal.ID, &journal.Code, &journal.Name, &journal.Type, &journal.SequencePrefix,
		&journal.NextNumber, &journal.IsActive, &journal.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, accounting.ErrJournalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get journal by id: %w", err)
	}
	return &journal, nil
}

func (r *PostgresRepository) GetNextSequenceWithLock(ctx context.Context, tx pgx.Tx, journalID uuid.UUID) (string, error) {
	const query = `
		SELECT sequence_prefix, next_number
		FROM journals WHERE id = $1
		FOR UPDATE`
	var prefix string
	var nextNumber int64
	if err := tx.QueryRow(ctx, query, journalID).Scan(&prefix, &nextNumber); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", accounting.ErrJournalNotFound
		}
		return "", fmt.Errorf("lock journal sequence: %w", err)
	}
	if nextNumber < 1 {
		return "", fmt.Errorf("journal %s has invalid next sequence number %d", journalID, nextNumber)
	}

	const update = `UPDATE journals SET next_number = next_number + 1 WHERE id = $1`
	tag, err := tx.Exec(ctx, update, journalID)
	if err != nil {
		return "", fmt.Errorf("increment journal sequence: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return "", accounting.ErrJournalNotFound
	}
	return fmt.Sprintf("%s%05d", prefix, nextNumber), nil
}

func (r *PostgresRepository) SaveDraftMove(ctx context.Context, move *accounting.AccountMove) error {
	if move.State != accounting.MoveStateDraft {
		return accounting.ErrMoveAlreadyPosted
	}
	if err := move.ValidateBalance(); err != nil {
		return err
	}
	if move.ID == uuid.Nil {
		move.ID = uuid.New()
	}
	if move.CreatedAt.IsZero() {
		move.CreatedAt = time.Now().UTC()
	}
	move.UpdatedAt = time.Now().UTC()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save draft move: %w", err)
	}
	defer tx.Rollback(ctx)

	const query = `
		INSERT INTO account_moves (
			id, name, journal_id, period_id, date, state, ref, reversed_move_id,
			hash_signature, posted_at, posted_by, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, 'draft', $6, NULL, NULL, NULL, NULL, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			journal_id = EXCLUDED.journal_id,
			period_id = EXCLUDED.period_id,
			date = EXCLUDED.date,
			ref = EXCLUDED.ref,
			created_by = COALESCE(account_moves.created_by, EXCLUDED.created_by),
			updated_at = EXCLUDED.updated_at
		WHERE account_moves.state = 'draft'`
	tag, err := tx.Exec(ctx, query,
		move.ID, move.Name, move.JournalID, move.PeriodID, move.Date, move.Ref,
		move.CreatedBy, move.CreatedAt, move.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("save draft move header: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return accounting.ErrMoveAlreadyPosted
	}
	if _, err := tx.Exec(ctx, `DELETE FROM account_move_lines WHERE move_id = $1`, move.ID); err != nil {
		return fmt.Errorf("replace draft move lines: %w", err)
	}
	if err := insertMoveLines(ctx, tx, move.ID, move.Date, move.Lines); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit draft move: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetMoveByID(ctx context.Context, id uuid.UUID) (*accounting.AccountMove, error) {
	move, err := r.getMoveByID(ctx, r.pool, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, accounting.ErrMoveNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get move by id: %w", err)
	}
	return move, nil
}

func (r *PostgresRepository) getMoveByID(ctx context.Context, db dbtx, id uuid.UUID) (*accounting.AccountMove, error) {
	const query = `
		SELECT id, name, journal_id, period_id, date, state, ref, reversed_move_id,
			hash_signature, posted_at, posted_by, created_by, created_at, updated_at
		FROM account_moves WHERE id = $1`
	move, err := scanMove(db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, err
	}
	const linesQuery = `
		SELECT id, move_id, account_id, partner_id, debit, credit, amount_currency,
			currency_id, cost_center_id, matching_number, date, created_at
		FROM account_move_lines WHERE move_id = $1
		ORDER BY date, created_at, id`
	rows, err := db.Query(ctx, linesQuery, id)
	if err != nil {
		return nil, fmt.Errorf("query move lines: %w", err)
	}
	defer rows.Close()
	move.Lines = make([]accounting.AccountMoveLine, 0)
	for rows.Next() {
		line, err := scanMoveLine(rows)
		if err != nil {
			return nil, fmt.Errorf("scan move line: %w", err)
		}
		move.Lines = append(move.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate move lines: %w", err)
	}
	return move, nil
}

func (r *PostgresRepository) PostMoveAtomic(ctx context.Context, move *accounting.AccountMove, audit *accounting.FinancialAuditLog) error {
	return r.postMoveAtomic(ctx, move, audit, nil, false)
}

func (r *PostgresRepository) PostNewMoveWithEventAtomic(
	ctx context.Context,
	move *accounting.AccountMove,
	audit *accounting.FinancialAuditLog,
	idempotencyKey, eventType string,
) error {
	if idempotencyKey == "" || eventType == "" {
		return errors.New("idempotency key and event type are required")
	}
	return r.postMoveAtomic(ctx, move, audit, &processedEvent{
		idempotencyKey: idempotencyKey,
		eventType:      eventType,
	}, true)
}

type processedEvent struct {
	idempotencyKey string
	eventType      string
}

func (r *PostgresRepository) postMoveAtomic(
	ctx context.Context,
	move *accounting.AccountMove,
	audit *accounting.FinancialAuditLog,
	event *processedEvent,
	createMove bool,
) error {
	if move.State != accounting.MoveStateDraft {
		return accounting.ErrMoveAlreadyPosted
	}
	if err := move.ValidateBalance(); err != nil {
		return err
	}
	if createMove {
		if move.ID == uuid.Nil {
			move.ID = uuid.New()
		}
		if move.CreatedAt.IsZero() {
			move.CreatedAt = time.Now().UTC()
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin post move: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := lockOpenPeriod(ctx, tx, move.PeriodID, move.Date); err != nil {
		return err
	}
	if createMove {
		const insertMoveQuery = `
			INSERT INTO account_moves (
				id, name, journal_id, period_id, date, state, ref, reversed_move_id,
				hash_signature, posted_at, posted_by, created_at, updated_at
			) VALUES ($1, '', $2, $3, $4, 'draft', $5, NULL, NULL, NULL, NULL, $6, $7)`
		if _, err := tx.Exec(ctx, insertMoveQuery,
			move.ID, move.JournalID, move.PeriodID, move.Date, move.Ref, move.CreatedAt, move.CreatedAt,
		); err != nil {
			return fmt.Errorf("insert new draft move: %w", err)
		}
	}
	sequence, err := r.GetNextSequenceWithLock(ctx, tx, move.JournalID)
	if err != nil {
		return err
	}

	const lockMoveQuery = `
		SELECT state, journal_id, period_id, date
		FROM account_moves WHERE id = $1 FOR UPDATE`
	var state accounting.MoveState
	var journalID, periodID uuid.UUID
	var moveDate time.Time
	if err := tx.QueryRow(ctx, lockMoveQuery, move.ID).Scan(&state, &journalID, &periodID, &moveDate); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accounting.ErrMoveNotFound
		}
		return fmt.Errorf("lock draft move: %w", err)
	}
	if state != accounting.MoveStateDraft {
		return accounting.ErrMoveAlreadyPosted
	}
	if journalID != move.JournalID || periodID != move.PeriodID || !sameDate(moveDate, move.Date) {
		return errors.New("move fields changed since the draft was loaded")
	}

	postedAt := time.Now().UTC()
	hash := hashMove(move.ID, sequence, move.Date, move.JournalID)
	const updateQuery = `
		UPDATE account_moves
		SET name = $1, state = 'posted', hash_signature = $2,
			posted_at = $3, posted_by = $4, updated_at = $3
		WHERE id = $5 AND state = 'draft'`
	tag, err := tx.Exec(ctx, updateQuery, sequence, hash, postedAt, move.PostedBy, move.ID)
	if err != nil {
		return fmt.Errorf("mark move posted: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return accounting.ErrMoveAlreadyPosted
	}
	if _, err := tx.Exec(ctx, `DELETE FROM account_move_lines WHERE move_id = $1`, move.ID); err != nil {
		return fmt.Errorf("replace posted move lines: %w", err)
	}
	if err := insertMoveLines(ctx, tx, move.ID, move.Date, move.Lines); err != nil {
		return err
	}
	if err := insertAuditLog(ctx, tx, audit); err != nil {
		return err
	}
	if event != nil {
		const insertEventQuery = `
			INSERT INTO subledger_processed_events (idempotency_key, event_type, move_id)
			VALUES ($1, $2, $3)`
		if _, err := tx.Exec(ctx, insertEventQuery, event.idempotencyKey, event.eventType, move.ID); err != nil {
			return fmt.Errorf("save processed subledger event: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit posted move: %w", err)
	}

	move.Name = sequence
	move.State = accounting.MoveStatePosted
	move.HashSignature = hash
	move.PostedAt = &postedAt
	move.UpdatedAt = postedAt
	for i := range move.Lines {
		move.Lines[i].MoveID = move.ID
		if move.Lines[i].Date.IsZero() {
			move.Lines[i].Date = move.Date
		}
		if move.Lines[i].CreatedAt.IsZero() {
			move.Lines[i].CreatedAt = postedAt
		}
	}
	return nil
}

func (r *PostgresRepository) ReverseMoveAtomic(ctx context.Context, originalMoveID uuid.UUID, reversalMove *accounting.AccountMove, audit *accounting.FinancialAuditLog) error {
	if reversalMove.State != accounting.MoveStateDraft {
		return accounting.ErrMoveAlreadyPosted
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reverse move: %w", err)
	}
	defer tx.Rollback(ctx)

	const lockOriginalQuery = `SELECT state FROM account_moves WHERE id = $1 FOR UPDATE`
	var originalState accounting.MoveState
	if err := tx.QueryRow(ctx, lockOriginalQuery, originalMoveID).Scan(&originalState); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accounting.ErrMoveNotFound
		}
		return fmt.Errorf("lock original move: %w", err)
	}
	if originalState != accounting.MoveStatePosted {
		return accounting.ErrCannotReverseUnpostedMove
	}
	var reversalExists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM account_moves WHERE reversed_move_id = $1)`, originalMoveID,
	).Scan(&reversalExists); err != nil {
		return fmt.Errorf("check existing reversal: %w", err)
	}
	if reversalExists {
		return accounting.ErrMoveAlreadyReversed
	}
	if err := lockOpenPeriod(ctx, tx, reversalMove.PeriodID, reversalMove.Date); err != nil {
		return err
	}

	sequence, err := r.GetNextSequenceWithLock(ctx, tx, reversalMove.JournalID)
	if err != nil {
		return err
	}
	originalLines, err := loadMoveLines(ctx, tx, originalMoveID)
	if err != nil {
		return err
	}
	if len(originalLines) < 2 {
		return fmt.Errorf("original move %s has fewer than two lines", originalMoveID)
	}

	now := time.Now().UTC()
	reversalID := ensureUUID(reversalMove.ID)
	reversalHash := hashMove(reversalID, sequence, reversalMove.Date, reversalMove.JournalID)
	postedBy := auditActor(audit)
	createdAt := reversalMove.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}

	lineIDs := make([]uuid.UUID, len(originalLines))
	for i := range lineIDs {
		if i < len(reversalMove.Lines) && reversalMove.Lines[i].ID != uuid.Nil {
			lineIDs[i] = reversalMove.Lines[i].ID
		} else {
			lineIDs[i] = uuid.New()
		}
	}

	const insertMoveQuery = `
		INSERT INTO account_moves (
			id, name, journal_id, period_id, date, state, ref, reversed_move_id,
			hash_signature, posted_at, posted_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, 'posted', $6, $7, $8, $9, $10, $11, $12)`
	if _, err := tx.Exec(ctx, insertMoveQuery,
		reversalID, sequence, reversalMove.JournalID, reversalMove.PeriodID, reversalMove.Date,
		reversalMove.Ref, originalMoveID, reversalHash, now, postedBy,
		createdAt, now,
	); err != nil {
		return fmt.Errorf("insert reversal move: %w", err)
	}

	reversalLines := make([]accounting.AccountMoveLine, len(originalLines))
	for i, originalLine := range originalLines {
		reversalLines[i] = accounting.AccountMoveLine{
			ID:             lineIDs[i],
			MoveID:         reversalID,
			AccountID:      originalLine.AccountID,
			PartnerID:      originalLine.PartnerID,
			Debit:          originalLine.Credit,
			Credit:         originalLine.Debit,
			AmountCurrency: originalLine.AmountCurrency.Neg(),
			CurrencyID:     originalLine.CurrencyID,
			CostCenterID:   originalLine.CostCenterID,
			MatchingNumber: originalLine.MatchingNumber,
			Date:           reversalMove.Date,
			CreatedAt:      now,
		}
	}
	if err := insertMoveLines(ctx, tx, reversalID, reversalMove.Date, reversalLines); err != nil {
		return err
	}
	if err := insertAuditLog(ctx, tx, audit); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit reversal: %w", err)
	}
	reversalMove.ID = reversalID
	reversalMove.ReversedMoveID = &originalMoveID
	reversalMove.State = accounting.MoveStatePosted
	reversalMove.Name = sequence
	reversalMove.PostedAt = &now
	reversalMove.PostedBy = postedBy
	reversalMove.HashSignature = reversalHash
	reversalMove.CreatedAt = createdAt
	reversalMove.UpdatedAt = now
	reversalMove.Lines = reversalLines
	return nil
}

func (r *PostgresRepository) GetProcessedEventMove(ctx context.Context, idempotencyKey string) (*accounting.AccountMove, error) {
	const query = `SELECT move_id FROM subledger_processed_events WHERE idempotency_key = $1`
	var moveID uuid.UUID
	if err := r.pool.QueryRow(ctx, query, idempotencyKey).Scan(&moveID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get processed event: %w", err)
	}
	move, err := r.GetMoveByID(ctx, moveID)
	if err != nil {
		return nil, fmt.Errorf("get processed event move: %w", err)
	}
	return move, nil
}

func (r *PostgresRepository) SaveProcessedEvent(ctx context.Context, key, eventType string, moveID uuid.UUID) error {
	const query = `
		INSERT INTO subledger_processed_events (idempotency_key, event_type, move_id)
		VALUES ($1, $2, $3)`
	if _, err := r.pool.Exec(ctx, query, key, eventType, moveID); err != nil {
		return fmt.Errorf("save processed event: %w", err)
	}
	return nil
}

func scanAccount(row pgx.Row) (*accounting.Account, error) {
	var account accounting.Account
	var parentID pgtype.UUID
	err := row.Scan(
		&account.ID, &account.Code, &account.Name, &account.Type, &account.NormalBalance,
		&account.CurrencyID, &parentID, &account.IsReconcilable, &account.IsActive,
		&account.CreatedAt, &account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	account.ParentID = nullableUUID(parentID)
	return &account, nil
}

func scanPeriod(row pgx.Row) (*accounting.FiscalPeriod, error) {
	var period accounting.FiscalPeriod
	var closedAt pgtype.Timestamptz
	var closedBy pgtype.UUID
	err := row.Scan(
		&period.ID, &period.FiscalYearID, &period.PeriodNumber, &period.Name, &period.StartDate,
		&period.EndDate, &period.State, &closedAt, &closedBy, &period.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if closedAt.Valid {
		period.ClosedAt = &closedAt.Time
	}
	period.ClosedBy = nullableUUID(closedBy)
	return &period, nil
}

func scanMove(row pgx.Row) (*accounting.AccountMove, error) {
	var move accounting.AccountMove
	var reversedMoveID, postedBy, createdBy pgtype.UUID
	var postedAt pgtype.Timestamptz
	err := row.Scan(
		&move.ID, &move.Name, &move.JournalID, &move.PeriodID, &move.Date, &move.State, &move.Ref,
		&reversedMoveID, &move.HashSignature, &postedAt, &postedBy, &createdBy, &move.CreatedAt, &move.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	move.ReversedMoveID = nullableUUID(reversedMoveID)
	move.PostedBy = nullableUUID(postedBy)
	move.CreatedBy = nullableUUID(createdBy)
	if postedAt.Valid {
		move.PostedAt = &postedAt.Time
	}
	return &move, nil
}

func scanMoveLine(row pgx.Row) (accounting.AccountMoveLine, error) {
	var line accounting.AccountMoveLine
	var partnerID, costCenterID pgtype.UUID
	err := row.Scan(
		&line.ID, &line.MoveID, &line.AccountID, &partnerID, &line.Debit, &line.Credit,
		&line.AmountCurrency, &line.CurrencyID, &costCenterID, &line.MatchingNumber,
		&line.Date, &line.CreatedAt,
	)
	if err != nil {
		return accounting.AccountMoveLine{}, err
	}
	line.PartnerID = nullableUUID(partnerID)
	line.CostCenterID = nullableUUID(costCenterID)
	return line, nil
}

func nullableUUID(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	result := uuid.UUID(value.Bytes)
	return &result
}

func insertMoveLines(ctx context.Context, tx dbtx, moveID uuid.UUID, moveDate time.Time, lines []accounting.AccountMoveLine) error {
	const query = `
		INSERT INTO account_move_lines (
			id, move_id, account_id, partner_id, debit, credit, amount_currency, currency_id,
			cost_center_id, matching_number, date, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	for i := range lines {
		line := &lines[i]
		if line.ID == uuid.Nil {
			line.ID = uuid.New()
		}
		if line.Date.IsZero() {
			line.Date = moveDate
		}
		if line.CreatedAt.IsZero() {
			line.CreatedAt = time.Now().UTC()
		}
		if _, err := tx.Exec(ctx, query,
			line.ID, moveID, line.AccountID, line.PartnerID, line.Debit, line.Credit,
			line.AmountCurrency, line.CurrencyID, line.CostCenterID, line.MatchingNumber, line.Date, line.CreatedAt,
		); err != nil {
			return fmt.Errorf("insert move line %d: %w", i, err)
		}
	}
	return nil
}

func insertAuditLog(ctx context.Context, tx dbtx, audit *accounting.FinancialAuditLog) error {
	if audit == nil {
		return nil
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, $2)`, int32(0x41554454), int32(1)); err != nil {
		return fmt.Errorf("lock financial audit chain: %w", err)
	}
	if audit.Timestamp.IsZero() {
		audit.Timestamp = time.Now().UTC()
	}
	audit.Timestamp = audit.Timestamp.UTC().Truncate(time.Microsecond)

	var payloadAfter string
	if err := tx.QueryRow(ctx, `SELECT COALESCE($1::jsonb::text, '')`, audit.PayloadAfter).Scan(&payloadAfter); err != nil {
		return fmt.Errorf("normalize financial audit payload: %w", err)
	}
	var previousHash string
	err := tx.QueryRow(ctx, `
		SELECT current_log_hash
		FROM financial_audit_logs
		ORDER BY id DESC
		LIMIT 1`).Scan(&previousHash)
	if errors.Is(err, pgx.ErrNoRows) {
		previousHash = ""
	} else if err != nil {
		return fmt.Errorf("get previous financial audit hash: %w", err)
	}
	audit.PreviousLogHash = previousHash
	audit.CurrentLogHash = (AuditHasher{}).ComputeHash(
		previousHash, audit.EntityID, audit.Action, []byte(payloadAfter), audit.Timestamp,
	)
	const query = `
		INSERT INTO financial_audit_logs (
			entity_type, entity_id, action, actor_id, payload_before, payload_after,
			ip_address, previous_log_hash, current_log_hash, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::inet, $8, $9, $10)`
	var ipAddress string
	if audit.IPAddress != "" {
		ipAddress = audit.IPAddress
	}
	if _, err := tx.Exec(ctx, query,
		audit.EntityType, audit.EntityID, audit.Action, audit.ActorID, audit.PayloadBefore,
		audit.PayloadAfter, ipAddress, audit.PreviousLogHash, audit.CurrentLogHash, audit.Timestamp,
	); err != nil {
		return fmt.Errorf("insert financial audit log: %w", err)
	}
	return nil
}

func lockOpenPeriod(ctx context.Context, tx pgx.Tx, periodID uuid.UUID, date time.Time) error {
	const query = `
		SELECT state FROM fiscal_periods
		WHERE id = $1 AND start_date <= $2 AND end_date >= $2
		FOR SHARE`
	var state accounting.PeriodState
	if err := tx.QueryRow(ctx, query, periodID, date).Scan(&state); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accounting.ErrPeriodNotFound
		}
		return fmt.Errorf("lock fiscal period: %w", err)
	}
	if state != accounting.PeriodStateOpen {
		return accounting.ErrPeriodClosed
	}
	return nil
}

func loadMoveLines(ctx context.Context, tx pgx.Tx, moveID uuid.UUID) ([]accounting.AccountMoveLine, error) {
	const query = `
		SELECT id, move_id, account_id, partner_id, debit, credit, amount_currency,
			currency_id, cost_center_id, matching_number, date, created_at
		FROM account_move_lines WHERE move_id = $1
		ORDER BY created_at, id`
	rows, err := tx.Query(ctx, query, moveID)
	if err != nil {
		return nil, fmt.Errorf("query original move lines: %w", err)
	}
	defer rows.Close()
	lines := make([]accounting.AccountMoveLine, 0)
	for rows.Next() {
		line, err := scanMoveLine(rows)
		if err != nil {
			return nil, fmt.Errorf("scan original move line: %w", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate original move lines: %w", err)
	}
	return lines, nil
}

func hashMove(id uuid.UUID, name string, date time.Time, journalID uuid.UUID) string {
	payload := fmt.Sprintf("%s:%s:%s:%s", id, name, date.Format("2006-01-02"), journalID)
	hash := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(hash[:])
}

func auditActor(audit *accounting.FinancialAuditLog) *uuid.UUID {
	if audit == nil {
		return nil
	}
	actorID := audit.ActorID
	return &actorID
}

func ensureUUID(id uuid.UUID) uuid.UUID {
	if id == uuid.Nil {
		return uuid.New()
	}
	return id
}

func sameDate(left, right time.Time) bool {
	y1, m1, d1 := left.Date()
	y2, m2, d2 := right.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
