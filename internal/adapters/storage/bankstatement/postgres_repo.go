package bankstatementstorage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cashflow_backend/internal/domain/bankstatement"
	"cashflow_backend/internal/platform/database"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepo implements bankstatement.Repository using pgx against PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

const bankStatementColumns = `id, name, journal_id, partner_id, date,
	balance_start, balance_end, balance_end_real, currency, state,
	is_complete, is_valid, COALESCE(problem_description, ''),
	active, created_at, updated_at, created_by, updated_by`

const bankStatementLineColumns = `id, statement_id, name, COALESCE(ref, ''), sequence, date,
	amount, amount_currency, currency, partner_id, account_id, move_id, journal_id,
	checked, running_balance, amount_residual, reconciled, matching_number,
	COALESCE(internal_index, ''), COALESCE(import_batch_id, ''), created_at, updated_at`

// ─────────────────────────────────────────────────────────────────────────────
// Statements
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateStatement(ctx context.Context, st *bankstatement.BankStatement) error {
	now := time.Now().UTC()
	st.Active = true
	if st.State == "" {
		st.State = bankstatement.StatementStateOpen
	}
	if st.Currency == "" {
		st.Currency = "USD"
	}

	query := `
		INSERT INTO account_bank_statements (
			name, journal_id, partner_id, date, balance_start, balance_end,
			balance_end_real, currency, state, is_complete, is_valid, problem_description,
			active, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17
		) RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		st.Name, st.JournalID, st.PartnerID, st.Date, st.BalanceStart, st.BalanceEnd,
		st.BalanceEndReal, st.Currency, string(st.State), st.IsComplete, st.IsValid, st.ProblemDescription,
		st.Active, now, now, st.Audit.CreatedBy, st.Audit.UpdatedBy,
	).Scan(&st.ID, &st.Audit.CreatedAt, &st.Audit.UpdatedAt)
	if err != nil {
		return platformerrors.Internal("failed to create bank statement", err)
	}
	st.Lines = nil
	return nil
}

func (r *PostgresRepo) scanStatement(row pgx.Row) (*bankstatement.BankStatement, error) {
	var st bankstatement.BankStatement
	var state string
	err := row.Scan(
		&st.ID, &st.Name, &st.JournalID, &st.PartnerID, &st.Date,
		&st.BalanceStart, &st.BalanceEnd, &st.BalanceEndReal, &st.Currency, &state,
		&st.IsComplete, &st.IsValid, &st.ProblemDescription,
		&st.Active, &st.Audit.CreatedAt, &st.Audit.UpdatedAt, &st.Audit.CreatedBy, &st.Audit.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}
	st.State = bankstatement.StatementState(state)
	return &st, nil
}

func (r *PostgresRepo) GetStatementByID(ctx context.Context, id int64) (*bankstatement.BankStatement, error) {
	query := `SELECT ` + bankStatementColumns + ` FROM account_bank_statements WHERE id = $1 AND active = true`
	st, err := r.scanStatement(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("bank statement with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch bank statement", err)
	}
	return st, nil
}

func (r *PostgresRepo) GetStatementWithLines(ctx context.Context, id int64) (*bankstatement.BankStatement, error) {
	st, err := r.GetStatementByID(ctx, id)
	if err != nil {
		return nil, err
	}

	linesQuery := `SELECT ` + bankStatementLineColumns + `
		FROM account_bank_statement_lines
		WHERE statement_id = $1
		ORDER BY sequence ASC, id ASC`
	rows, err := r.pool.Query(ctx, linesQuery, id)
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch bank statement lines", err)
	}
	defer rows.Close()

	for rows.Next() {
		var l bankstatement.BankStatementLine
		if err := scanStatementLine(rows, &l); err != nil {
			return nil, platformerrors.Internal("failed to scan bank statement line", err)
		}
		st.Lines = append(st.Lines, l)
	}
	return st, nil
}

func (r *PostgresRepo) UpdateStatement(ctx context.Context, st *bankstatement.BankStatement) error {
	query := `
		UPDATE account_bank_statements
		SET name = $1, journal_id = $2, partner_id = $3, date = $4,
		    balance_start = $5, balance_end = $6, balance_end_real = $7,
		    currency = $8, state = $9, is_complete = $10, is_valid = $11,
		    problem_description = $12, updated_at = NOW(), updated_by = $13
		WHERE id = $14 AND active = true
		RETURNING updated_at
	`
	if err := r.pool.QueryRow(ctx, query,
		st.Name, st.JournalID, st.PartnerID, st.Date,
		st.BalanceStart, st.BalanceEnd, st.BalanceEndReal,
		st.Currency, string(st.State), st.IsComplete, st.IsValid,
		st.ProblemDescription, st.Audit.UpdatedBy, st.ID,
	).Scan(&st.Audit.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("bank statement with id %d not found", st.ID))
		}
		return platformerrors.Internal("failed to update bank statement", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteStatement(ctx context.Context, id int64) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM account_bank_statement_lines WHERE statement_id = $1`, id); err != nil {
			return platformerrors.Internal("failed to delete bank statement lines", err)
		}
		cmdTag, err := tx.Exec(ctx, `UPDATE account_bank_statements SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`, id)
		if err != nil {
			return platformerrors.Internal("failed to delete bank statement", err)
		}
		if cmdTag.RowsAffected() == 0 {
			return platformerrors.NotFound(fmt.Sprintf("bank statement with id %d not found", id))
		}
		return nil
	})
}

func (r *PostgresRepo) ListStatements(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[bankstatement.BankStatement], error) {
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM account_bank_statements WHERE active = true`).Scan(&total); err != nil {
		return pagination.PageResult[bankstatement.BankStatement]{}, platformerrors.Internal("failed to count bank statements", err)
	}

	offset := page.Offset()
	limit := page.LimitClamped()
	query := `SELECT ` + bankStatementColumns + `
		FROM account_bank_statements
		WHERE active = true
		ORDER BY date DESC, id DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return pagination.PageResult[bankstatement.BankStatement]{}, platformerrors.Internal("failed to list bank statements", err)
	}
	defer rows.Close()

	var statements []bankstatement.BankStatement
	for rows.Next() {
		st, err := r.scanStatement(rows)
		if err != nil {
			return pagination.PageResult[bankstatement.BankStatement]{}, platformerrors.Internal("failed to scan bank statement", err)
		}
		statements = append(statements, *st)
	}
	return pagination.NewPageResult(statements, total, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Statement Lines
// ─────────────────────────────────────────────────────────────────────────────

func scanStatementLine(row pgx.Row, l *bankstatement.BankStatementLine) error {
	return row.Scan(
		&l.ID, &l.StatementID, &l.Name, &l.Ref, &l.Sequence, &l.Date,
		&l.Amount, &l.AmountCurrency, &l.Currency, &l.PartnerID, &l.AccountID, &l.MoveID, &l.JournalID,
		&l.Checked, &l.RunningBalance, &l.AmountResidual, &l.Reconciled, &l.MatchingNumber,
		&l.InternalIndex, &l.ImportBatchID, &l.CreatedAt, &l.UpdatedAt,
	)
}

func (r *PostgresRepo) AddStatementLines(ctx context.Context, statementID int64, lines []bankstatement.BankStatementLine) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			INSERT INTO account_bank_statement_lines (
				statement_id, name, ref, sequence, date, amount, amount_currency,
				currency, partner_id, account_id, move_id, journal_id, checked,
				running_balance, amount_residual, reconciled, matching_number,
				internal_index, import_batch_id, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7,
				$8, $9, $10, $11, $12, $13,
				$14, $15, $16, $17,
				$18, $19, $20, $21
			) RETURNING id, created_at, updated_at
		`
		now := time.Now().UTC()
		for i := range lines {
			l := &lines[i]
			l.StatementID = statementID
			if l.Currency == "" {
				l.Currency = "USD"
			}
			if err := tx.QueryRow(ctx, query,
				statementID, l.Name, nullableString(l.Ref), l.Sequence, l.Date, l.Amount, l.AmountCurrency,
				l.Currency, l.PartnerID, l.AccountID, l.MoveID, l.JournalID, l.Checked,
				l.RunningBalance, l.AmountResidual, l.Reconciled, l.MatchingNumber,
				nullableString(l.InternalIndex), nullableString(l.ImportBatchID), now, now,
			).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt); err != nil {
				return platformerrors.Internal("failed to insert bank statement line", err)
			}
		}
		return nil
	})
}

func (r *PostgresRepo) GetStatementLineByID(ctx context.Context, id int64) (*bankstatement.BankStatementLine, error) {
	query := `SELECT ` + bankStatementLineColumns + `
		FROM account_bank_statement_lines
		WHERE id = $1`
	var l bankstatement.BankStatementLine
	if err := scanStatementLine(r.pool.QueryRow(ctx, query, id), &l); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("bank statement line with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch bank statement line", err)
	}
	return &l, nil
}

func (r *PostgresRepo) UpdateStatementLine(ctx context.Context, line *bankstatement.BankStatementLine) error {
	query := `
		UPDATE account_bank_statement_lines
		SET name = $2, ref = $3, date = $4, amount = $5, amount_currency = $6,
		    currency = $7, partner_id = $8, account_id = $9, move_id = $10,
		    journal_id = $11, checked = $12, amount_residual = $13,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	if err := r.pool.QueryRow(ctx, query,
		line.ID, line.Name, nullableString(line.Ref), line.Date, line.Amount, line.AmountCurrency,
		line.Currency, line.PartnerID, line.AccountID, line.MoveID, line.JournalID,
		line.Checked, line.AmountResidual,
	).Scan(&line.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("bank statement line with id %d not found", line.ID))
		}
		return platformerrors.Internal("failed to update bank statement line", err)
	}
	return nil
}

func (r *PostgresRepo) UpdateStatementLineReconcileState(ctx context.Context, lineID int64, reconciled bool, residual float64, matchingNumber *string) error {
	query := `
		UPDATE account_bank_statement_lines
		SET reconciled = $2, amount_residual = $3, matching_number = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	if err := r.pool.QueryRow(ctx, query, lineID, reconciled, residual, matchingNumber).Scan(new(time.Time)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("bank statement line with id %d not found", lineID))
		}
		return platformerrors.Internal("failed to update bank statement line reconcile state", err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Reconciles
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreatePartialReconcile(ctx context.Context, pr *bankstatement.PartialReconcile) error {
	query := `
		INSERT INTO account_partial_reconciles (
			debit_move_id, credit_move_id, debit_line_id, credit_line_id,
			amount, amount_currency, currency, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	now := time.Now().UTC()
	if pr.Currency == "" {
		pr.Currency = "USD"
	}
	err := r.pool.QueryRow(ctx, query,
		pr.DebitMoveID, pr.CreditMoveID, pr.DebitLineID, pr.CreditLineID,
		pr.Amount, pr.AmountCurrency, pr.Currency,
	).Scan(&pr.ID, &pr.CreatedAt, &pr.UpdatedAt)
	if err != nil {
		return platformerrors.Internal("failed to create partial reconcile", err)
	}
	pr.CreatedAt = now
	return nil
}

func (r *PostgresRepo) ListPartialReconcilesByLine(ctx context.Context, lineID int64) ([]bankstatement.PartialReconcile, error) {
	query := `
		SELECT id, debit_move_id, credit_move_id, debit_line_id, credit_line_id,
		       amount, amount_currency, currency, created_at, updated_at
		FROM account_partial_reconciles
		WHERE debit_line_id = $1 OR credit_line_id = $1
		ORDER BY id ASC
	`
	rows, err := r.pool.Query(ctx, query, lineID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list partial reconciles", err)
	}
	defer rows.Close()

	var result []bankstatement.PartialReconcile
	for rows.Next() {
		var pr bankstatement.PartialReconcile
		if err := rows.Scan(
			&pr.ID, &pr.DebitMoveID, &pr.CreditMoveID, &pr.DebitLineID, &pr.CreditLineID,
			&pr.Amount, &pr.AmountCurrency, &pr.Currency, &pr.CreatedAt, &pr.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan partial reconcile", err)
		}
		result = append(result, pr)
	}
	return result, nil
}

func (r *PostgresRepo) DeletePartialReconcile(ctx context.Context, id int64) error {
	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM account_partial_reconciles WHERE id = $1`, id)
	if err != nil {
		return platformerrors.Internal("failed to delete partial reconcile", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("partial reconcile with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) CreateFullReconcile(ctx context.Context, fr *bankstatement.FullReconcile) error {
	query := `
		INSERT INTO account_full_reconciles (matching_number, exchange_move_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	if err := r.pool.QueryRow(ctx, query, fr.MatchingNumber, fr.ExchangeMoveID).Scan(&fr.ID, &fr.CreatedAt, &fr.UpdatedAt); err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "matching_number") {
			return platformerrors.Conflict(fmt.Sprintf("full reconcile with matching number '%s' already exists", fr.MatchingNumber), err)
		}
		return platformerrors.Internal("failed to create full reconcile", err)
	}
	return nil
}

func (r *PostgresRepo) GetFullReconcileByMatchingNumber(ctx context.Context, matchingNumber string) (*bankstatement.FullReconcile, error) {
	query := `
		SELECT id, matching_number, exchange_move_id, created_at, updated_at
		FROM account_full_reconciles
		WHERE matching_number = $1
	`
	var fr bankstatement.FullReconcile
	if err := r.pool.QueryRow(ctx, query, matchingNumber).Scan(
		&fr.ID, &fr.MatchingNumber, &fr.ExchangeMoveID, &fr.CreatedAt, &fr.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("full reconcile '%s' not found", matchingNumber))
		}
		return nil, platformerrors.Internal("failed to fetch full reconcile", err)
	}
	return &fr, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Reconcile Models
// ─────────────────────────────────────────────────────────────────────────────

const reconcileModelColumns = `id, name, sequence, is_auto_reconcile, match_nature,
	match_amount, match_amount_min, match_amount_max, match_label, match_label_param,
	match_journal_ids, match_partner_ids, mapped_partner_id, active, created_at, updated_at`

func scanReconcileModel(row pgx.Row, m *bankstatement.ReconcileModel) error {
	var nature string
	var matchAmount, matchLabel *string
	if err := row.Scan(
		&m.ID, &m.Name, &m.Sequence, &m.IsAutoReconcile, &nature,
		&matchAmount, &m.MatchAmountMin, &m.MatchAmountMax, &matchLabel, &m.MatchLabelParam,
		&m.MatchJournalIDs, &m.MatchPartnerIDs, &m.MappedPartnerID, &m.Active, &m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		return err
	}
	m.MatchNature = bankstatement.ReconcileMatchNature(nature)
	if matchAmount != nil {
		v := bankstatement.ReconcileMatchAmount(*matchAmount)
		m.MatchAmount = &v
	}
	if matchLabel != nil {
		v := bankstatement.ReconcileMatchLabel(*matchLabel)
		m.MatchLabel = &v
	}
	return nil
}

func loadModelLines(ctx context.Context, q interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, modelID int64) ([]bankstatement.ReconcileModelLine, error) {
	query := `
		SELECT id, reconcile_model_id, amount_type, amount, account_id,
		       COALESCE(label, ''), tax_ids
		FROM account_reconcile_model_lines
		WHERE reconcile_model_id = $1
		ORDER BY id ASC
	`
	rows, err := q.Query(ctx, query, modelID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list reconcile model lines", err)
	}
	defer rows.Close()

	var lines []bankstatement.ReconcileModelLine
	for rows.Next() {
		var l bankstatement.ReconcileModelLine
		var amountType string
		if err := rows.Scan(&l.ID, &l.ReconcileModelID, &amountType, &l.Amount, &l.AccountID, &l.Label, &l.TaxIDs); err != nil {
			return nil, platformerrors.Internal("failed to scan reconcile model line", err)
		}
		l.AmountType = bankstatement.ReconcileLineAmountType(amountType)
		lines = append(lines, l)
	}
	return lines, nil
}

func (r *PostgresRepo) CreateReconcileModel(ctx context.Context, m *bankstatement.ReconcileModel) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			INSERT INTO account_reconcile_models (
				name, sequence, is_auto_reconcile, match_nature, match_amount,
				match_amount_min, match_amount_max, match_label, match_label_param,
				match_journal_ids, match_partner_ids, mapped_partner_id, active,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
			RETURNING id, created_at, updated_at
		`
		if m.MatchNature == "" {
			m.MatchNature = bankstatement.MatchNatureBoth
		}
		err := tx.QueryRow(ctx, query,
			m.Name, m.Sequence, m.IsAutoReconcile, string(m.MatchNature), stringPtr(m.MatchAmount),
			m.MatchAmountMin, m.MatchAmountMax, stringPtr(m.MatchLabel), m.MatchLabelParam,
			m.MatchJournalIDs, m.MatchPartnerIDs, m.MappedPartnerID, true,
		).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return platformerrors.Internal("failed to create reconcile model", err)
		}

		if err := replaceModelLines(ctx, tx, m.ID, m.Lines); err != nil {
			return err
		}
		return nil
	})
}

func (r *PostgresRepo) GetReconcileModelByID(ctx context.Context, id int64) (*bankstatement.ReconcileModel, error) {
	query := `SELECT ` + reconcileModelColumns + ` FROM account_reconcile_models WHERE id = $1 AND active = true`
	var m bankstatement.ReconcileModel
	if err := scanReconcileModel(r.pool.QueryRow(ctx, query, id), &m); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("reconcile model with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch reconcile model", err)
	}

	lines, err := loadModelLines(ctx, r.pool, id)
	if err != nil {
		return nil, err
	}
	m.Lines = lines
	return &m, nil
}

func (r *PostgresRepo) UpdateReconcileModel(ctx context.Context, m *bankstatement.ReconcileModel) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			UPDATE account_reconcile_models
			SET name = $2, sequence = $3, is_auto_reconcile = $4, match_nature = $5,
			    match_amount = $6, match_amount_min = $7, match_amount_max = $8,
			    match_label = $9, match_label_param = $10, match_journal_ids = $11,
			    match_partner_ids = $12, mapped_partner_id = $13, updated_at = NOW()
			WHERE id = $1 AND active = true
			RETURNING updated_at
		`
		if err := tx.QueryRow(ctx, query,
			m.ID, m.Name, m.Sequence, m.IsAutoReconcile, string(m.MatchNature),
			stringPtr(m.MatchAmount), m.MatchAmountMin, m.MatchAmountMax,
			stringPtr(m.MatchLabel), m.MatchLabelParam,
			m.MatchJournalIDs, m.MatchPartnerIDs, m.MappedPartnerID,
		).Scan(&m.UpdatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return platformerrors.NotFound(fmt.Sprintf("reconcile model with id %d not found", m.ID))
			}
			return platformerrors.Internal("failed to update reconcile model", err)
		}

		if err := replaceModelLines(ctx, tx, m.ID, m.Lines); err != nil {
			return err
		}
		return nil
	})
}

func replaceModelLines(ctx context.Context, tx pgx.Tx, modelID int64, lines []bankstatement.ReconcileModelLine) error {
	if _, err := tx.Exec(ctx, `DELETE FROM account_reconcile_model_lines WHERE reconcile_model_id = $1`, modelID); err != nil {
		return platformerrors.Internal("failed to delete previous reconcile model lines", err)
	}
	query := `
		INSERT INTO account_reconcile_model_lines (
			reconcile_model_id, amount_type, amount, account_id, label, tax_ids
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	for i := range lines {
		if err := tx.QueryRow(ctx, query,
			modelID, string(lines[i].AmountType), lines[i].Amount, lines[i].AccountID,
			nullableString(lines[i].Label), lines[i].TaxIDs,
		).Scan(&lines[i].ID); err != nil {
			return platformerrors.Internal("failed to insert reconcile model line", err)
		}
	}
	return nil
}

func (r *PostgresRepo) DeleteReconcileModel(ctx context.Context, id int64) error {
	cmdTag, err := r.pool.Exec(ctx, `UPDATE account_reconcile_models SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`, id)
	if err != nil {
		return platformerrors.Internal("failed to delete reconcile model", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("reconcile model with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) listModels(ctx context.Context, autoOnly bool) ([]bankstatement.ReconcileModel, error) {
	query := `SELECT ` + reconcileModelColumns + ` FROM account_reconcile_models WHERE active = true`
	if autoOnly {
		query += ` AND is_auto_reconcile = true`
	}
	query += ` ORDER BY sequence ASC, id ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, platformerrors.Internal("failed to list reconcile models", err)
	}
	defer rows.Close()

	var models []bankstatement.ReconcileModel
	for rows.Next() {
		var m bankstatement.ReconcileModel
		if err := scanReconcileModel(rows, &m); err != nil {
			return nil, platformerrors.Internal("failed to scan reconcile model", err)
		}
		models = append(models, m)
	}
	return models, nil
}

func (r *PostgresRepo) ListReconcileModels(ctx context.Context) ([]bankstatement.ReconcileModel, error) {
	return r.listModels(ctx, false)
}

func (r *PostgresRepo) ListAutoReconcileModels(ctx context.Context) ([]bankstatement.ReconcileModel, error) {
	return r.listModels(ctx, true)
}

// ─────────────────────────────────────────────────────────────────────────────
// Cash Rounding
// ─────────────────────────────────────────────────────────────────────────────

const cashRoundingColumns = `id, name, rounding_method, rounding, strategy,
	profit_account_id, loss_account_id, active, created_at, updated_at`

func (r *PostgresRepo) CreateCashRounding(ctx context.Context, cr *bankstatement.CashRounding) error {
	query := `
		INSERT INTO account_cash_roundings (
			name, rounding_method, rounding, strategy, profit_account_id,
			loss_account_id, active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		cr.Name, string(cr.RoundingMethod), cr.Rounding, string(cr.Strategy),
		cr.ProfitAccountID, cr.LossAccountID, true,
	).Scan(&cr.ID, &cr.CreatedAt, &cr.UpdatedAt)
	if err != nil {
		return platformerrors.Internal("failed to create cash rounding", err)
	}
	cr.Active = true
	return nil
}

func scanCashRounding(row pgx.Row, cr *bankstatement.CashRounding) error {
	var method, strategy string
	return row.Scan(
		&cr.ID, &cr.Name, &method, &cr.Rounding, &strategy,
		&cr.ProfitAccountID, &cr.LossAccountID, &cr.Active, &cr.CreatedAt, &cr.UpdatedAt,
	)
}

func (r *PostgresRepo) GetCashRoundingByID(ctx context.Context, id int64) (*bankstatement.CashRounding, error) {
	var cr bankstatement.CashRounding
	if err := scanCashRounding(r.pool.QueryRow(ctx,
		`SELECT `+cashRoundingColumns+` FROM account_cash_roundings WHERE id = $1 AND active = true`, id), &cr); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("cash rounding with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch cash rounding", err)
	}
	return &cr, nil
}

func (r *PostgresRepo) UpdateCashRounding(ctx context.Context, cr *bankstatement.CashRounding) error {
	query := `
		UPDATE account_cash_roundings
		SET name = $2, rounding_method = $3, rounding = $4, strategy = $5,
		    profit_account_id = $6, loss_account_id = $7, updated_at = NOW()
		WHERE id = $1 AND active = true
		RETURNING updated_at
	`
	if err := r.pool.QueryRow(ctx, query,
		cr.ID, cr.Name, string(cr.RoundingMethod), cr.Rounding, string(cr.Strategy),
		cr.ProfitAccountID, cr.LossAccountID,
	).Scan(&cr.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("cash rounding with id %d not found", cr.ID))
		}
		return platformerrors.Internal("failed to update cash rounding", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteCashRounding(ctx context.Context, id int64) error {
	cmdTag, err := r.pool.Exec(ctx, `UPDATE account_cash_roundings SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`, id)
	if err != nil {
		return platformerrors.Internal("failed to delete cash rounding", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("cash rounding with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListCashRoundings(ctx context.Context) ([]bankstatement.CashRounding, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+cashRoundingColumns+` FROM account_cash_roundings WHERE active = true ORDER BY id ASC`)
	if err != nil {
		return nil, platformerrors.Internal("failed to list cash roundings", err)
	}
	defer rows.Close()

	var list []bankstatement.CashRounding
	for rows.Next() {
		var cr bankstatement.CashRounding
		if err := scanCashRounding(rows, &cr); err != nil {
			return nil, platformerrors.Internal("failed to scan cash rounding", err)
		}
		list = append(list, cr)
	}
	return list, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringPtr(s any) *string {
	if s == nil {
		return nil
	}
	switch v := s.(type) {
	case *bankstatement.ReconcileMatchAmount:
		return nullableEnum(v)
	case *bankstatement.ReconcileMatchLabel:
		return nullableEnum(v)
	}
	return nil
}

func nullableEnum[T ~string](v *T) *string {
	if v == nil {
		return nil
	}
	str := string(*v)
	return &str
}
