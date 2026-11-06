package internal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
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
			is_reconcilable, is_active, is_cash_equivalent, is_non_cash_adjustment,
			cash_flow_activity, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := r.pool.Exec(ctx, query,
		acc.ID, acc.Code, acc.Name, acc.Type, acc.NormalBalance, acc.CurrencyID,
		acc.ParentID, acc.IsReconcilable, acc.IsActive, acc.IsCashEquivalent,
		acc.IsNonCashAdjustment, acc.CashFlowActivity, acc.CreatedAt, acc.UpdatedAt,
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

func (r *PostgresRepository) UpdateAccount(ctx context.Context, account *accounting.Account) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin account update: %w", err)
	}
	defer tx.Rollback(ctx)

	if account.ParentID != nil {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, $2)`, int32(0x41434354), int32(1)); err != nil {
			return fmt.Errorf("lock account hierarchy: %w", err)
		}
		var parentIsDescendant bool
		if err := tx.QueryRow(ctx, `
			WITH RECURSIVE descendants(id) AS (
				SELECT $1::uuid
				UNION
				SELECT child.id
				FROM accounts child
				JOIN descendants parent ON child.parent_id = parent.id
			)
			SELECT EXISTS (SELECT 1 FROM descendants WHERE id = $2)`,
			account.ID, *account.ParentID,
		).Scan(&parentIsDescendant); err != nil {
			return fmt.Errorf("check account hierarchy: %w", err)
		}
		if parentIsDescendant {
			return errors.New("an account cannot be moved under one of its descendants")
		}
	}

	var oldCode string
	var oldType accounting.AccountType
	const lockQuery = `
		SELECT code, account_type
		FROM accounts
		WHERE id = $1
		FOR UPDATE`
	if err := tx.QueryRow(ctx, lockQuery, account.ID).Scan(&oldCode, &oldType); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accounting.ErrAccountNotFound
		}
		return fmt.Errorf("lock account for update: %w", err)
	}
	if oldCode != account.Code || oldType != account.Type {
		var hasPostedMoves bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM account_move_lines l
				JOIN account_moves m ON m.id = l.move_id
				WHERE l.account_id = $1 AND m.state IN ('posted', 'reversed')
			)`, account.ID).Scan(&hasPostedMoves); err != nil {
			return fmt.Errorf("check account posting history: %w", err)
		}
		if hasPostedMoves {
			return accounting.ErrAccountHasPostedMoves
		}
	}
	const updateQuery = `
		UPDATE accounts SET
			code = $2, name = $3, account_type = $4, normal_balance = $5,
			currency_id = $6, parent_id = $7, is_reconcilable = $8, is_active = $9,
			is_cash_equivalent = $10, is_non_cash_adjustment = $11,
			cash_flow_activity = $12, updated_at = $13
		WHERE id = $1`
	tag, err := tx.Exec(ctx, updateQuery,
		account.ID, account.Code, account.Name, account.Type, account.NormalBalance,
		account.CurrencyID, account.ParentID, account.IsReconcilable, account.IsActive,
		account.IsCashEquivalent, account.IsNonCashAdjustment, account.CashFlowActivity, account.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "uq_accounts_code" {
			return fmt.Errorf("%w: %s", accounting.ErrDuplicateAccountCode, account.Code)
		}
		return fmt.Errorf("update account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return accounting.ErrAccountNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit account update: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (*accounting.Account, error) {
	const query = `
		SELECT id, code, name, account_type, normal_balance, currency_id, parent_id,
			is_reconcilable, is_active, is_cash_equivalent, is_non_cash_adjustment,
			cash_flow_activity, created_at, updated_at
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
			is_reconcilable, is_active, is_cash_equivalent, is_non_cash_adjustment,
			cash_flow_activity, created_at, updated_at
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
			is_reconcilable, is_active, is_cash_equivalent, is_non_cash_adjustment,
			cash_flow_activity, created_at, updated_at
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

func (r *PostgresRepository) GetSubledgerEventMapping(ctx context.Context, eventType string) (*accounting.SubledgerEventMapping, error) {
	const query = `
		SELECT event_type, journal_id, debit_account_id, credit_account_id,
			COALESCE(description, ''), is_active
		FROM subledger_event_mappings
		WHERE event_type = $1 AND is_active = TRUE`
	var mapping accounting.SubledgerEventMapping
	if err := r.pool.QueryRow(ctx, query, eventType).Scan(
		&mapping.EventType, &mapping.JournalID, &mapping.DebitAccountID,
		&mapping.CreditAccountID, &mapping.Description, &mapping.IsActive,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, accounting.ErrSubledgerMappingNotFound
		}
		return nil, fmt.Errorf("get subledger event mapping: %w", err)
	}
	return &mapping, nil
}

func (r *PostgresRepository) ListSubledgerEventMappings(ctx context.Context) ([]*accounting.SubledgerEventMapping, error) {
	const query = `
		SELECT event_type, journal_id, debit_account_id, credit_account_id,
			COALESCE(description, ''), is_active
		FROM subledger_event_mappings
		ORDER BY event_type`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list subledger event mappings: %w", err)
	}
	defer rows.Close()
	mappings := make([]*accounting.SubledgerEventMapping, 0)
	for rows.Next() {
		var m accounting.SubledgerEventMapping
		if err := rows.Scan(
			&m.EventType, &m.JournalID, &m.DebitAccountID,
			&m.CreditAccountID, &m.Description, &m.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan subledger event mapping: %w", err)
		}
		mappings = append(mappings, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subledger event mappings: %w", err)
	}
	return mappings, nil
}

func (r *PostgresRepository) SaveSubledgerEventMapping(ctx context.Context, mapping *accounting.SubledgerEventMapping) error {
	const query = `
		INSERT INTO subledger_event_mappings (
			event_type, journal_id, debit_account_id, credit_account_id, description, is_active, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (event_type) DO UPDATE SET
			journal_id = EXCLUDED.journal_id,
			debit_account_id = EXCLUDED.debit_account_id,
			credit_account_id = EXCLUDED.credit_account_id,
			description = EXCLUDED.description,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()`
	_, err := r.pool.Exec(ctx, query,
		mapping.EventType, mapping.JournalID, mapping.DebitAccountID,
		mapping.CreditAccountID, mapping.Description, mapping.IsActive,
	)
	if err != nil {
		return fmt.Errorf("save subledger event mapping: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetAccountingConfiguration(ctx context.Context) (*accounting.AccountingConfiguration, error) {
	const query = `
		SELECT retained_earnings_account_id, fx_gain_account_id, fx_loss_account_id,
			fx_reconciliation_journal_id, functional_currency
		FROM accounting_configurations
		ORDER BY updated_at DESC
		LIMIT 1`
	var configuration accounting.AccountingConfiguration
	if err := r.pool.QueryRow(ctx, query).Scan(
		&configuration.RetainedEarningsAccountID, &configuration.FXGainAccountID,
		&configuration.FXLossAccountID, &configuration.FXReconciliationJournalID,
		&configuration.FunctionalCurrency,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, accounting.ErrAccountingConfigurationMissing
		}
		return nil, fmt.Errorf("get accounting configuration: %w", err)
	}
	return &configuration, nil
}

func (r *PostgresRepository) UpdateAccountingConfiguration(ctx context.Context, config *accounting.AccountingConfiguration) error {
	const query = `
		INSERT INTO accounting_configurations (
			retained_earnings_account_id, fx_gain_account_id, fx_loss_account_id,
			fx_reconciliation_journal_id, functional_currency, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)`
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, query,
		config.RetainedEarningsAccountID, config.FXGainAccountID, config.FXLossAccountID,
		config.FXReconciliationJournalID, config.FunctionalCurrency, now,
	)
	if err != nil {
		return fmt.Errorf("update accounting configuration: %w", err)
	}
	return nil
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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin fiscal period state update: %w", err)
	}
	defer tx.Rollback(ctx)

	var yearState string
	var periodState accounting.PeriodState
	if err := tx.QueryRow(ctx, `
		SELECT fy.state, fp.state
		FROM fiscal_periods fp
		JOIN fiscal_years fy ON fy.id = fp.fiscal_year_id
		WHERE fp.id = $1
		FOR UPDATE OF fy, fp`,
		id,
	).Scan(&yearState, &periodState); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accounting.ErrPeriodNotFound
		}
		return fmt.Errorf("lock fiscal year for period state update: %w", err)
	}
	if yearState != "open" {
		return accounting.ErrPeriodClosed
	}
	if periodState == state {
		return nil
	}
	switch periodState {
	case accounting.PeriodStateOpen:
		if state != accounting.PeriodStateSoftClosed && state != accounting.PeriodStateHardClosed {
			return accounting.ErrInvalidPeriodTransition
		}
	case accounting.PeriodStateSoftClosed:
		if state != accounting.PeriodStateOpen && state != accounting.PeriodStateHardClosed {
			return accounting.ErrInvalidPeriodTransition
		}
	default:
		return accounting.ErrInvalidPeriodTransition
	}

	const query = `
		UPDATE fiscal_periods
		SET state = $2,
			closed_at = CASE WHEN $2 = 'open' THEN NULL ELSE NOW() END,
			closed_by = CASE WHEN $2 = 'open' THEN NULL ELSE $3 END
		WHERE id = $1`
	tag, err := tx.Exec(ctx, query, id, state, actorID)
	if err != nil {
		return fmt.Errorf("update fiscal period state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return accounting.ErrPeriodNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit fiscal period state update: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListFiscalYears(ctx context.Context) ([]*accounting.FiscalYearSummary, error) {
	const yearsQuery = `
		SELECT id, year, start_date, end_date, state, created_at
		FROM fiscal_years
		ORDER BY start_date DESC`
	rows, err := r.pool.Query(ctx, yearsQuery)
	if err != nil {
		return nil, fmt.Errorf("list fiscal years: %w", err)
	}
	defer rows.Close()
	summaries := make([]*accounting.FiscalYearSummary, 0)
	for rows.Next() {
		year := &accounting.FiscalYear{}
		if err := rows.Scan(&year.ID, &year.Year, &year.StartDate, &year.EndDate, &year.State, &year.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan fiscal year: %w", err)
		}
		summaries = append(summaries, &accounting.FiscalYearSummary{
			Year: year, Periods: make([]*accounting.FiscalPeriod, 0),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fiscal years: %w", err)
	}
	for _, summary := range summaries {
		const periodsQuery = `
			SELECT id, fiscal_year_id, period_number, name, start_date, end_date, state,
				closed_at, closed_by, created_at
			FROM fiscal_periods
			WHERE fiscal_year_id = $1
			ORDER BY period_number`
		periodRows, err := r.pool.Query(ctx, periodsQuery, summary.Year.ID)
		if err != nil {
			return nil, fmt.Errorf("list fiscal year periods: %w", err)
		}
		for periodRows.Next() {
			period, err := scanPeriod(periodRows)
			if err != nil {
				periodRows.Close()
				return nil, fmt.Errorf("scan fiscal period: %w", err)
			}
			summary.Periods = append(summary.Periods, period)
		}
		if err := periodRows.Err(); err != nil {
			periodRows.Close()
			return nil, fmt.Errorf("iterate fiscal periods: %w", err)
		}
		periodRows.Close()
	}
	return summaries, nil
}

func (r *PostgresRepository) CreateJournal(ctx context.Context, journal *accounting.Journal) error {
	if journal.ID == uuid.Nil {
		journal.ID = uuid.New()
	}
	if journal.CreatedAt.IsZero() {
		journal.CreatedAt = time.Now().UTC()
	}
	if journal.NextNumber < 1 {
		journal.NextNumber = 1
	}
	const query = `
		INSERT INTO journals (id, code, name, type, sequence_prefix, next_number, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query,
		journal.ID, journal.Code, journal.Name, journal.Type,
		journal.SequencePrefix, journal.NextNumber, journal.IsActive, journal.CreatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return accounting.ErrDuplicateJournalCode
		}
		return fmt.Errorf("create journal: %w", err)
	}
	return nil
}

func (r *PostgresRepository) UpdateJournal(ctx context.Context, journal *accounting.Journal) error {
	const query = `
		UPDATE journals
		SET name = $2, sequence_prefix = $3, is_active = $4
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, journal.ID, journal.Name, journal.SequencePrefix, journal.IsActive)
	if err != nil {
		return fmt.Errorf("update journal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return accounting.ErrJournalNotFound
	}
	return nil
}

func (r *PostgresRepository) GetJournalByCode(ctx context.Context, code string) (*accounting.Journal, error) {
	const query = `
		SELECT id, code, name, type, sequence_prefix, next_number, is_active, created_at
		FROM journals WHERE code = $1`
	var journal accounting.Journal
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&journal.ID, &journal.Code, &journal.Name, &journal.Type, &journal.SequencePrefix,
		&journal.NextNumber, &journal.IsActive, &journal.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, accounting.ErrJournalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get journal by code: %w", err)
	}
	return &journal, nil
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

func (r *PostgresRepository) ListJournals(ctx context.Context) ([]*accounting.Journal, error) {
	const query = `
		SELECT id, code, name, type, sequence_prefix, next_number, is_active, created_at
		FROM journals
		ORDER BY code`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list journals: %w", err)
	}
	defer rows.Close()
	journals := make([]*accounting.Journal, 0)
	for rows.Next() {
		journal := &accounting.Journal{}
		if err := rows.Scan(
			&journal.ID, &journal.Code, &journal.Name, &journal.Type, &journal.SequencePrefix,
			&journal.NextNumber, &journal.IsActive, &journal.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan journal: %w", err)
		}
		journals = append(journals, journal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate journals: %w", err)
	}
	return journals, nil
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
	move.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)

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

func (r *PostgresRepository) DeleteDraftMove(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete draft move: %w", err)
	}
	defer tx.Rollback(ctx)
	var state accounting.MoveState
	if err := tx.QueryRow(ctx, `SELECT state FROM account_moves WHERE id = $1 FOR UPDATE`, id).Scan(&state); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accounting.ErrMoveNotFound
		}
		return fmt.Errorf("lock draft move for delete: %w", err)
	}
	if state != accounting.MoveStateDraft {
		return accounting.ErrMoveAlreadyPosted
	}
	if _, err := tx.Exec(ctx, `DELETE FROM account_moves WHERE id = $1 AND state = 'draft'`, id); err != nil {
		return fmt.Errorf("delete draft move: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit draft move deletion: %w", err)
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

func (r *PostgresRepository) ListMoves(ctx context.Context, filter accounting.MoveFilter) ([]*accounting.AccountMove, int, error) {
	const countQuery = `
		SELECT count(*)
		FROM account_moves
		WHERE ($1::uuid IS NULL OR journal_id = $1)
			AND ($2::date IS NULL OR date >= $2)
			AND ($3::date IS NULL OR date <= $3)
			AND ($4::text = '' OR state = $4)`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery,
		filter.JournalID, filter.FromDate, filter.ToDate, string(filter.State),
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count accounting moves: %w", err)
	}
	const query = `
		SELECT id
		FROM account_moves
		WHERE ($1::uuid IS NULL OR journal_id = $1)
			AND ($2::date IS NULL OR date >= $2)
			AND ($3::date IS NULL OR date <= $3)
			AND ($4::text = '' OR state = $4)
		ORDER BY date DESC, created_at DESC, id DESC
		LIMIT $5 OFFSET $6`
	rows, err := r.pool.Query(ctx, query,
		filter.JournalID, filter.FromDate, filter.ToDate, string(filter.State), filter.Limit, filter.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("query accounting moves: %w", err)
	}
	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, 0, fmt.Errorf("scan accounting move id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, 0, fmt.Errorf("iterate accounting moves: %w", err)
	}
	rows.Close()
	moves := make([]*accounting.AccountMove, 0, len(ids))
	for _, id := range ids {
		move, err := r.GetMoveByID(ctx, id)
		if err != nil {
			return nil, 0, fmt.Errorf("load accounting move %s: %w", id, err)
		}
		moves = append(moves, move)
	}
	return moves, total, nil
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
	const lockMoveQuery = `
		SELECT state, journal_id, period_id, date, updated_at
		FROM account_moves WHERE id = $1 FOR UPDATE`
	var state accounting.MoveState
	var journalID, periodID uuid.UUID
	var moveDate time.Time
	var updatedAt time.Time
	if err := tx.QueryRow(ctx, lockMoveQuery, move.ID).Scan(&state, &journalID, &periodID, &moveDate, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return accounting.ErrMoveNotFound
		}
		return fmt.Errorf("lock draft move: %w", err)
	}
	if state != accounting.MoveStateDraft {
		return accounting.ErrMoveAlreadyPosted
	}
	if journalID != move.JournalID || periodID != move.PeriodID || !sameDate(moveDate, move.Date) {
		return accounting.ErrMoveChanged
	}
	if !createMove && !move.UpdatedAt.IsZero() && !move.UpdatedAt.Equal(updatedAt) {
		return accounting.ErrMoveChanged
	}
	lines := append([]accounting.AccountMoveLine(nil), move.Lines...)
	if _, err := tx.Exec(ctx, `DELETE FROM account_move_lines WHERE move_id = $1`, move.ID); err != nil {
		return fmt.Errorf("replace posted move lines: %w", err)
	}
	if err := insertMoveLines(ctx, tx, move.ID, move.Date, lines); err != nil {
		return err
	}
	sequence, err := r.GetNextSequenceWithLock(ctx, tx, move.JournalID)
	if err != nil {
		return err
	}
	previousMoveHash, err := previousPostedMoveHash(ctx, tx, move.JournalID)
	if err != nil {
		return err
	}
	hash := hashMove(move.ID, sequence, move.Date, move.JournalID, lines, previousMoveHash)
	postedAt := time.Now().UTC()
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
	move.Lines = lines
	for i := range move.Lines {
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
	previousMoveHash, err := previousPostedMoveHash(ctx, tx, reversalMove.JournalID)
	if err != nil {
		return err
	}
	reversalHash := hashMove(reversalID, sequence, reversalMove.Date, reversalMove.JournalID, reversalLines, previousMoveHash)
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

func (r *PostgresRepository) ListProcessedEvents(ctx context.Context, limit, offset int) ([]*accounting.ProcessedEventSummary, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM subledger_processed_events`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count processed events: %w", err)
	}
	const query = `
		SELECT id, idempotency_key, event_type, move_id, processed_at
		FROM subledger_processed_events
		ORDER BY processed_at DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list processed events: %w", err)
	}
	defer rows.Close()
	events := make([]*accounting.ProcessedEventSummary, 0)
	for rows.Next() {
		var e accounting.ProcessedEventSummary
		if err := rows.Scan(&e.ID, &e.IdempotencyKey, &e.EventType, &e.MoveID, &e.ProcessedAt); err != nil {
			return nil, 0, fmt.Errorf("scan processed event: %w", err)
		}
		events = append(events, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate processed events: %w", err)
	}
	return events, total, nil
}

func (r *PostgresRepository) ListFinancialAuditLogs(ctx context.Context, limit, offset int) ([]*accounting.FinancialAuditLog, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM financial_audit_logs`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count financial audit logs: %w", err)
	}
	const query = `
		SELECT id, entity_type, entity_id, action, actor_id,
			COALESCE(payload_after::text, ''),
			COALESCE(previous_log_hash, ''), current_log_hash, timestamp
		FROM financial_audit_logs
		ORDER BY id DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list financial audit logs: %w", err)
	}
	defer rows.Close()
	logs := make([]*accounting.FinancialAuditLog, 0)
	for rows.Next() {
		var log accounting.FinancialAuditLog
		var payloadText string
		if err := rows.Scan(
			&log.ID, &log.EntityType, &log.EntityID, &log.Action, &log.ActorID,
			&payloadText, &log.PreviousLogHash, &log.CurrentLogHash, &log.Timestamp,
		); err != nil {
			return nil, 0, fmt.Errorf("scan financial audit log: %w", err)
		}
		if payloadText != "" {
			log.PayloadAfter = []byte(payloadText)
		}
		logs = append(logs, &log)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate financial audit logs: %w", err)
	}
	return logs, total, nil
}

func scanAccount(row pgx.Row) (*accounting.Account, error) {
	var account accounting.Account
	var parentID pgtype.UUID
	err := row.Scan(
		&account.ID, &account.Code, &account.Name, &account.Type, &account.NormalBalance,
		&account.CurrencyID, &parentID, &account.IsReconcilable, &account.IsActive,
		&account.IsCashEquivalent, &account.IsNonCashAdjustment, &account.CashFlowActivity,
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
		line.MoveID = moveID
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

func hashMove(
	id uuid.UUID,
	name string,
	date time.Time,
	journalID uuid.UUID,
	lines []accounting.AccountMoveLine,
	previousHash string,
) string {
	type moveLineHashPayload struct {
		ID             string `json:"id"`
		MoveID         string `json:"move_id"`
		AccountID      string `json:"account_id"`
		Debit          string `json:"debit"`
		Credit         string `json:"credit"`
		AmountCurrency string `json:"amount_currency"`
		CurrencyID     string `json:"currency_id"`
		MatchingNumber string `json:"matching_number"`
		Date           string `json:"date"`
		PartnerID      string `json:"partner_id,omitempty"`
		CostCenterID   string `json:"cost_center_id,omitempty"`
	}
	linePayloads := make([]moveLineHashPayload, len(lines))
	for i, line := range lines {
		entry := moveLineHashPayload{
			ID: line.ID.String(), MoveID: line.MoveID.String(),
			AccountID: line.AccountID.String(), Debit: line.Debit.String(),
			Credit: line.Credit.String(), AmountCurrency: line.AmountCurrency.String(),
			CurrencyID: line.CurrencyID, MatchingNumber: line.MatchingNumber,
			Date: line.Date.Format("2006-01-02"),
		}
		if line.PartnerID != nil {
			entry.PartnerID = line.PartnerID.String()
		}
		if line.CostCenterID != nil {
			entry.CostCenterID = line.CostCenterID.String()
		}
		linePayloads[i] = entry
	}
	sort.Slice(linePayloads, func(i, j int) bool {
		left, _ := json.Marshal(linePayloads[i])
		right, _ := json.Marshal(linePayloads[j])
		return string(left) < string(right)
	})
	payload, _ := json.Marshal(struct {
		ID           string                `json:"id"`
		Name         string                `json:"name"`
		Date         string                `json:"date"`
		JournalID    string                `json:"journal_id"`
		Lines        []moveLineHashPayload `json:"lines"`
		PreviousHash string                `json:"previous_hash"`
	}{
		ID: id.String(), Name: name, Date: date.Format("2006-01-02"),
		JournalID: journalID.String(), Lines: linePayloads, PreviousHash: previousHash,
	})
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

func previousPostedMoveHash(ctx context.Context, tx pgx.Tx, journalID uuid.UUID) (string, error) {
	var previousHash string
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(hash_signature, '')
		FROM account_moves
		WHERE journal_id = $1 AND state IN ('posted', 'reversed')
		ORDER BY name DESC
		LIMIT 1`, journalID).Scan(&previousHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get previous journal move hash: %w", err)
	}
	return previousHash, nil
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
