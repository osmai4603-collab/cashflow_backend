package accountingstorage

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/platform/database"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepo implements accounting.Repository using pgx against PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// ─────────────────────────────────────────────────────────────────────────────
// Chart of Accounts
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateAccount(ctx context.Context, a *accounting.Account) error {
	query := `
		INSERT INTO account_accounts (
			code, name, type, reconcile, currency, parent_id, company_id,
			active, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12
		) RETURNING id, created_at, updated_at
	`
	now := time.Now().UTC()
	a.Active = true
	a.Audit.CreatedAt = now
	a.Audit.UpdatedAt = now

	err := r.pool.QueryRow(ctx, query,
		a.Code, a.Name, string(a.Type), a.Reconcile, a.Currency, a.ParentID, a.CompanyID,
		a.Active, a.Audit.CreatedAt, a.Audit.UpdatedAt, a.Audit.CreatedBy, a.Audit.UpdatedBy,
	).Scan(&a.ID, &a.Audit.CreatedAt, &a.Audit.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "account_accounts_code_key") {
			return platformerrors.Conflict(fmt.Sprintf("account with code '%s' already exists", a.Code), err)
		}
		return platformerrors.Internal("failed to create account", err)
	}
	return nil
}

func (r *PostgresRepo) GetAccountByID(ctx context.Context, id int64) (*accounting.Account, error) {
	query := `
		SELECT id, code, name, type, reconcile, currency, parent_id, company_id,
		       active, created_at, updated_at, created_by, updated_by
		FROM account_accounts
		WHERE id = $1 AND active = true
	`
	var a accounting.Account
	var accType string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.Code, &a.Name, &accType, &a.Reconcile, &a.Currency, &a.ParentID, &a.CompanyID,
		&a.Active, &a.Audit.CreatedAt, &a.Audit.UpdatedAt, &a.Audit.CreatedBy, &a.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("account with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch account", err)
	}
	a.Type = accounting.AccountType(accType)
	return &a, nil
}

func (r *PostgresRepo) GetAccountByCode(ctx context.Context, code string) (*accounting.Account, error) {
	query := `
		SELECT id, code, name, type, reconcile, currency, parent_id, company_id,
		       active, created_at, updated_at, created_by, updated_by
		FROM account_accounts
		WHERE code = $1 AND active = true
	`
	var a accounting.Account
	var accType string
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&a.ID, &a.Code, &a.Name, &accType, &a.Reconcile, &a.Currency, &a.ParentID, &a.CompanyID,
		&a.Active, &a.Audit.CreatedAt, &a.Audit.UpdatedAt, &a.Audit.CreatedBy, &a.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("account with code '%s' not found", code))
		}
		return nil, platformerrors.Internal("failed to fetch account by code", err)
	}
	a.Type = accounting.AccountType(accType)
	return &a, nil
}

func (r *PostgresRepo) UpdateAccount(ctx context.Context, a *accounting.Account) error {
	query := `
		UPDATE account_accounts
		SET code = $1, name = $2, type = $3, reconcile = $4, currency = $5,
		    parent_id = $6, company_id = $7, updated_at = NOW(), updated_by = $8
		WHERE id = $9 AND active = true
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		a.Code, a.Name, string(a.Type), a.Reconcile, a.Currency,
		a.ParentID, a.CompanyID, a.Audit.UpdatedBy, a.ID,
	).Scan(&a.Audit.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("account with id %d not found", a.ID))
		}
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "account_accounts_code_key") {
			return platformerrors.Conflict(fmt.Sprintf("account with code '%s' already exists", a.Code), err)
		}
		return platformerrors.Internal("failed to update account", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteAccount(ctx context.Context, id int64) error {
	// Check reference in move lines
	var count int
	checkQuery := `SELECT COUNT(*) FROM account_move_lines WHERE account_id = $1`
	if err := r.pool.QueryRow(ctx, checkQuery, id).Scan(&count); err == nil && count > 0 {
		return platformerrors.Conflict("cannot delete account referenced by journal entries")
	}

	query := `UPDATE account_accounts SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to soft-delete account", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("account with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListAccounts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[accounting.Account], error) {
	countQuery := `SELECT COUNT(*) FROM account_accounts WHERE active = true`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return pagination.PageResult[accounting.Account]{}, platformerrors.Internal("failed to count accounts", err)
	}

	offset := page.Offset()
	limit := page.LimitClamped()
	query := `
		SELECT id, code, name, type, reconcile, currency, parent_id, company_id,
		       active, created_at, updated_at, created_by, updated_by
		FROM account_accounts
		WHERE active = true
		ORDER BY code ASC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return pagination.PageResult[accounting.Account]{}, platformerrors.Internal("failed to list accounts", err)
	}
	defer rows.Close()

	var accounts []accounting.Account
	for rows.Next() {
		var a accounting.Account
		var accType string
		if err := rows.Scan(
			&a.ID, &a.Code, &a.Name, &accType, &a.Reconcile, &a.Currency, &a.ParentID, &a.CompanyID,
			&a.Active, &a.Audit.CreatedAt, &a.Audit.UpdatedAt, &a.Audit.CreatedBy, &a.Audit.UpdatedBy,
		); err != nil {
			return pagination.PageResult[accounting.Account]{}, platformerrors.Internal("failed to scan account", err)
		}
		a.Type = accounting.AccountType(accType)
		accounts = append(accounts, a)
	}

	return pagination.NewPageResult(accounts, total, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Journals
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateJournal(ctx context.Context, j *accounting.Journal) error {
	query := `
		INSERT INTO account_journals (
			name, code, type, default_account_id, suspense_account_id,
			sequence_prefix, next_number, active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10
		) RETURNING id, created_at, updated_at
	`
	now := time.Now().UTC()
	j.Active = true
	j.CreatedAt = now
	j.UpdatedAt = now

	err := r.pool.QueryRow(ctx, query,
		j.Name, j.Code, string(j.Type), j.DefaultAccountID, j.SuspenseAccountID,
		j.SequencePrefix, j.NextNumber, j.Active, j.CreatedAt, j.UpdatedAt,
	).Scan(&j.ID, &j.CreatedAt, &j.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "account_journals_code_key") {
			return platformerrors.Conflict(fmt.Sprintf("journal with code '%s' already exists", j.Code), err)
		}
		return platformerrors.Internal("failed to create journal", err)
	}
	return nil
}

func (r *PostgresRepo) GetJournalByID(ctx context.Context, id int64) (*accounting.Journal, error) {
	query := `
		SELECT id, name, code, type, default_account_id, suspense_account_id,
		       sequence_prefix, next_number, active, created_at, updated_at
		FROM account_journals
		WHERE id = $1 AND active = true
	`
	var j accounting.Journal
	var jType string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&j.ID, &j.Name, &j.Code, &jType, &j.DefaultAccountID, &j.SuspenseAccountID,
		&j.SequencePrefix, &j.NextNumber, &j.Active, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("journal with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch journal", err)
	}
	j.Type = accounting.JournalType(jType)
	return &j, nil
}

func (r *PostgresRepo) GetJournalByCode(ctx context.Context, code string) (*accounting.Journal, error) {
	query := `
		SELECT id, name, code, type, default_account_id, suspense_account_id,
		       sequence_prefix, next_number, active, created_at, updated_at
		FROM account_journals
		WHERE code = $1 AND active = true
	`
	var j accounting.Journal
	var jType string
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&j.ID, &j.Name, &j.Code, &jType, &j.DefaultAccountID, &j.SuspenseAccountID,
		&j.SequencePrefix, &j.NextNumber, &j.Active, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("journal with code '%s' not found", code))
		}
		return nil, platformerrors.Internal("failed to fetch journal by code", err)
	}
	j.Type = accounting.JournalType(jType)
	return &j, nil
}

func (r *PostgresRepo) UpdateJournal(ctx context.Context, j *accounting.Journal) error {
	query := `
		UPDATE account_journals
		SET name = $1, code = $2, type = $3, default_account_id = $4,
		    suspense_account_id = $5, sequence_prefix = $6, updated_at = NOW()
		WHERE id = $7 AND active = true
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		j.Name, j.Code, string(j.Type), j.DefaultAccountID,
		j.SuspenseAccountID, j.SequencePrefix, j.ID,
	).Scan(&j.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("journal with id %d not found", j.ID))
		}
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "account_journals_code_key") {
			return platformerrors.Conflict(fmt.Sprintf("journal with code '%s' already exists", j.Code), err)
		}
		return platformerrors.Internal("failed to update journal", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteJournal(ctx context.Context, id int64) error {
	var count int
	checkQuery := `SELECT COUNT(*) FROM account_moves WHERE journal_id = $1`
	if err := r.pool.QueryRow(ctx, checkQuery, id).Scan(&count); err == nil && count > 0 {
		return platformerrors.Conflict("cannot delete journal with associated account moves")
	}

	query := `UPDATE account_journals SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to soft-delete journal", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("journal with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListJournals(ctx context.Context) ([]accounting.Journal, error) {
	query := `
		SELECT id, name, code, type, default_account_id, suspense_account_id,
		       sequence_prefix, next_number, active, created_at, updated_at
		FROM account_journals
		WHERE active = true
		ORDER BY id ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, platformerrors.Internal("failed to list journals", err)
	}
	defer rows.Close()

	var journals []accounting.Journal
	for rows.Next() {
		var j accounting.Journal
		var jType string
		if err := rows.Scan(
			&j.ID, &j.Name, &j.Code, &jType, &j.DefaultAccountID, &j.SuspenseAccountID,
			&j.SequencePrefix, &j.NextNumber, &j.Active, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan journal", err)
		}
		j.Type = accounting.JournalType(jType)
		journals = append(journals, j)
	}
	return journals, nil
}

func (r *PostgresRepo) GetNextSequence(ctx context.Context, journalID int64, year int) (string, error) {
	query := `
		UPDATE account_journals
		SET next_number = next_number + 1, updated_at = NOW()
		WHERE id = $1 AND active = true
		RETURNING code, sequence_prefix, next_number - 1
	`
	var code, prefix string
	var currentNum int
	err := r.pool.QueryRow(ctx, query, journalID).Scan(&code, &prefix, &currentNum)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", platformerrors.NotFound(fmt.Sprintf("journal with id %d not found", journalID))
		}
		return "", platformerrors.Internal("failed to advance journal sequence", err)
	}

	j := accounting.Journal{
		Code:           code,
		SequencePrefix: prefix,
	}
	return j.FormatSequence(year, currentNum), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Taxes
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateTax(ctx context.Context, t *accounting.Tax) error {
	query := `
		INSERT INTO account_taxes (
			name, type, type_tax_use, amount, account_id, refund_account_id,
			price_include, active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10
		) RETURNING id, created_at, updated_at
	`
	now := time.Now().UTC()
	t.Active = true
	t.CreatedAt = now
	t.UpdatedAt = now

	err := r.pool.QueryRow(ctx, query,
		t.Name, string(t.Type), string(t.TypeTaxUse), t.Amount, t.AccountID, t.RefundAccountID,
		t.PriceInclude, t.Active, t.CreatedAt, t.UpdatedAt,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create tax", err)
	}
	return nil
}

func (r *PostgresRepo) GetTaxByID(ctx context.Context, id int64) (*accounting.Tax, error) {
	query := `
		SELECT id, name, type, type_tax_use, amount, account_id, refund_account_id,
		       price_include, active, created_at, updated_at
		FROM account_taxes
		WHERE id = $1 AND active = true
	`
	var t accounting.Tax
	var tType, tUse string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &tType, &tUse, &t.Amount, &t.AccountID, &t.RefundAccountID,
		&t.PriceInclude, &t.Active, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("tax with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch tax", err)
	}
	t.Type = accounting.TaxType(tType)
	t.TypeTaxUse = accounting.TaxScope(tUse)
	return &t, nil
}

func (r *PostgresRepo) UpdateTax(ctx context.Context, t *accounting.Tax) error {
	query := `
		UPDATE account_taxes
		SET name = $1, type = $2, type_tax_use = $3, amount = $4,
		    account_id = $5, refund_account_id = $6, price_include = $7, updated_at = NOW()
		WHERE id = $8 AND active = true
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		t.Name, string(t.Type), string(t.TypeTaxUse), t.Amount,
		t.AccountID, t.RefundAccountID, t.PriceInclude, t.ID,
	).Scan(&t.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("tax with id %d not found", t.ID))
		}
		return platformerrors.Internal("failed to update tax", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteTax(ctx context.Context, id int64) error {
	query := `UPDATE account_taxes SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to soft-delete tax", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("tax with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListTaxes(ctx context.Context, scope *accounting.TaxScope) ([]accounting.Tax, error) {
	query := `
		SELECT id, name, type, type_tax_use, amount, account_id, refund_account_id,
		       price_include, active, created_at, updated_at
		FROM account_taxes
		WHERE active = true AND ($1::text IS NULL OR type_tax_use = $1)
		ORDER BY id ASC
	`
	var scopeParam *string
	if scope != nil && *scope != "" {
		s := string(*scope)
		scopeParam = &s
	}

	rows, err := r.pool.Query(ctx, query, scopeParam)
	if err != nil {
		return nil, platformerrors.Internal("failed to list taxes", err)
	}
	defer rows.Close()

	var taxes []accounting.Tax
	for rows.Next() {
		var t accounting.Tax
		var tType, tUse string
		if err := rows.Scan(
			&t.ID, &t.Name, &tType, &tUse, &t.Amount, &t.AccountID, &t.RefundAccountID,
			&t.PriceInclude, &t.Active, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan tax", err)
		}
		t.Type = accounting.TaxType(tType)
		t.TypeTaxUse = accounting.TaxScope(tUse)
		taxes = append(taxes, t)
	}
	return taxes, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Payment Terms
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreatePaymentTerm(ctx context.Context, pt *accounting.PaymentTerm) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			INSERT INTO account_payment_terms (name, note, active, created_at, updated_at)
			VALUES ($1, $2, true, NOW(), NOW())
			RETURNING id, created_at, updated_at
		`
		if err := tx.QueryRow(ctx, query, pt.Name, pt.Note).Scan(&pt.ID, &pt.CreatedAt, &pt.UpdatedAt); err != nil {
			return platformerrors.Internal("failed to create payment term", err)
		}

		lineQuery := `
			INSERT INTO account_payment_term_lines (payment_term_id, value_type, value_amount, days, day_of_month)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`
		for i := range pt.Lines {
			pt.Lines[i].PaymentTermID = pt.ID
			if err := tx.QueryRow(ctx, lineQuery,
				pt.ID, string(pt.Lines[i].ValueType), pt.Lines[i].ValueAmount, pt.Lines[i].Days, pt.Lines[i].DayOfMonth,
			).Scan(&pt.Lines[i].ID); err != nil {
				return platformerrors.Internal("failed to insert payment term line", err)
			}
		}
		return nil
	})
}

func (r *PostgresRepo) GetPaymentTermByID(ctx context.Context, id int64) (*accounting.PaymentTerm, error) {
	query := `
		SELECT id, name, COALESCE(note, ''), active, created_at, updated_at
		FROM account_payment_terms
		WHERE id = $1 AND active = true
	`
	var pt accounting.PaymentTerm
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&pt.ID, &pt.Name, &pt.Note, &pt.Active, &pt.CreatedAt, &pt.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("payment term with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch payment term", err)
	}

	linesQuery := `
		SELECT id, payment_term_id, value_type, value_amount, days, day_of_month
		FROM account_payment_term_lines
		WHERE payment_term_id = $1
		ORDER BY id ASC
	`
	rows, err := r.pool.Query(ctx, linesQuery, id)
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch payment term lines", err)
	}
	defer rows.Close()

	for rows.Next() {
		var l accounting.PaymentTermLine
		var vType string
		if err := rows.Scan(&l.ID, &l.PaymentTermID, &vType, &l.ValueAmount, &l.Days, &l.DayOfMonth); err != nil {
			return nil, platformerrors.Internal("failed to scan payment term line", err)
		}
		l.ValueType = accounting.PaymentTermValueType(vType)
		pt.Lines = append(pt.Lines, l)
	}
	return &pt, nil
}

func (r *PostgresRepo) UpdatePaymentTerm(ctx context.Context, pt *accounting.PaymentTerm) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			UPDATE account_payment_terms
			SET name = $1, note = $2, updated_at = NOW()
			WHERE id = $3 AND active = true
			RETURNING updated_at
		`
		if err := tx.QueryRow(ctx, query, pt.Name, pt.Note, pt.ID).Scan(&pt.UpdatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return platformerrors.NotFound(fmt.Sprintf("payment term with id %d not found", pt.ID))
			}
			return platformerrors.Internal("failed to update payment term", err)
		}

		if len(pt.Lines) > 0 {
			if _, err := tx.Exec(ctx, `DELETE FROM account_payment_term_lines WHERE payment_term_id = $1`, pt.ID); err != nil {
				return platformerrors.Internal("failed to delete previous payment term lines", err)
			}
			lineQuery := `
				INSERT INTO account_payment_term_lines (payment_term_id, value_type, value_amount, days, day_of_month)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
			`
			for i := range pt.Lines {
				pt.Lines[i].PaymentTermID = pt.ID
				if err := tx.QueryRow(ctx, lineQuery,
					pt.ID, string(pt.Lines[i].ValueType), pt.Lines[i].ValueAmount, pt.Lines[i].Days, pt.Lines[i].DayOfMonth,
				).Scan(&pt.Lines[i].ID); err != nil {
					return platformerrors.Internal("failed to insert payment term line", err)
				}
			}
		}
		return nil
	})
}

func (r *PostgresRepo) DeletePaymentTerm(ctx context.Context, id int64) error {
	query := `UPDATE account_payment_terms SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete payment term", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("payment term with id %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListPaymentTerms(ctx context.Context) ([]accounting.PaymentTerm, error) {
	query := `
		SELECT id, name, COALESCE(note, ''), active, created_at, updated_at
		FROM account_payment_terms
		WHERE active = true
		ORDER BY id ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, platformerrors.Internal("failed to list payment terms", err)
	}
	defer rows.Close()

	var terms []accounting.PaymentTerm
	for rows.Next() {
		var pt accounting.PaymentTerm
		if err := rows.Scan(&pt.ID, &pt.Name, &pt.Note, &pt.Active, &pt.CreatedAt, &pt.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan payment term", err)
		}
		terms = append(terms, pt)
	}
	return terms, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Account Moves & Invoices
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateMove(ctx context.Context, m *accounting.AccountMove) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			INSERT INTO account_moves (
				name, move_type, journal_id, partner_id, date,
				invoice_date, invoice_date_due, payment_term_id, state, payment_state,
				amount_untaxed, amount_tax, amount_total, amount_residual, currency,
				ref, reversed_entry_id, active, created_at, updated_at, created_by, updated_by
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15,
				$16, $17, $18, $19, $20, $21, $22
			) RETURNING id, created_at, updated_at
		`
		now := time.Now().UTC()
		m.Active = true
		m.Audit.CreatedAt = now
		m.Audit.UpdatedAt = now

		if err := tx.QueryRow(ctx, query,
			m.Name, string(m.MoveType), m.JournalID, m.PartnerID, m.Date,
			m.InvoiceDate, m.InvoiceDueDate, m.PaymentTermID, string(m.State), string(m.PaymentState),
			m.AmountUntaxed, m.AmountTax, m.AmountTotal, m.AmountResidual, m.Currency,
			m.Ref, m.ReversedEntryID, m.Active, m.Audit.CreatedAt, m.Audit.UpdatedAt, m.Audit.CreatedBy, m.Audit.UpdatedBy,
		).Scan(&m.ID, &m.Audit.CreatedAt, &m.Audit.UpdatedAt); err != nil {
			return platformerrors.Internal("failed to create account move", err)
		}

		reconcileFlags, err := loadReconcileFlags(ctx, tx, m.Lines)
		if err != nil {
			return err
		}

		lineQuery := `
			INSERT INTO account_move_lines (
				move_id, account_id, partner_id, product_id, name,
				quantity, price_unit, discount, debit, credit, balance,
				tax_ids, tax_amount, reconcile, reconciled, amount_residual,
				matching_number, statement_line_id, display_type, cogs_origin_id, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10, $11,
				$12, $13, $14, $15, $16,
				$17, $18, $19, $20, NOW(), NOW()
			) RETURNING id, created_at, updated_at
		`
		for i := range m.Lines {
			l := &m.Lines[i]
			l.MoveID = m.ID
			l.Balance = l.Debit - l.Credit
			l.CreatedAt = now
			l.UpdatedAt = now
			applyReconcileDefaults(l, reconcileFlags[l.AccountID])

			if err := tx.QueryRow(ctx, lineQuery,
				m.ID, l.AccountID, l.PartnerID, l.ProductID, l.Name,
				l.Quantity, l.PriceUnit, l.Discount, l.Debit, l.Credit, l.Balance,
				l.TaxIDs, l.TaxAmount, l.Reconcile, l.Reconciled, l.AmountResidual,
				l.MatchingNumber, l.StatementLineID, l.DisplayType, l.CogsOriginID, l.CreatedAt, l.UpdatedAt,
			).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt); err != nil {
				return platformerrors.Internal("failed to insert move line", err)
			}
		}
		return nil
	})
}

func (r *PostgresRepo) GetMoveByID(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	query := `
		SELECT id, name, move_type, journal_id, partner_id, date,
		       invoice_date, invoice_date_due, payment_term_id, state, payment_state,
		       amount_untaxed, amount_tax, amount_total, amount_residual, currency,
		       COALESCE(ref, ''), reversed_entry_id, active, created_at, updated_at, created_by, updated_by
		FROM account_moves
		WHERE id = $1 AND active = true
	`
	var m accounting.AccountMove
	var mType, state, pState string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.Name, &mType, &m.JournalID, &m.PartnerID, &m.Date,
		&m.InvoiceDate, &m.InvoiceDueDate, &m.PaymentTermID, &state, &pState,
		&m.AmountUntaxed, &m.AmountTax, &m.AmountTotal, &m.AmountResidual, &m.Currency,
		&m.Ref, &m.ReversedEntryID, &m.Active, &m.Audit.CreatedAt, &m.Audit.UpdatedAt, &m.Audit.CreatedBy, &m.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("account move with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch move", err)
	}
	m.MoveType = accounting.MoveType(mType)
	m.State = accounting.MoveState(state)
	m.PaymentState = accounting.PaymentState(pState)
	return &m, nil
}

func (r *PostgresRepo) GetMoveWithLines(ctx context.Context, id int64) (*accounting.AccountMove, error) {
	m, err := r.GetMoveByID(ctx, id)
	if err != nil {
		return nil, err
	}

	linesQuery := `
		SELECT id, move_id, account_id, partner_id, product_id, name,
		       quantity, price_unit, discount, debit, credit, balance,
		       tax_ids, tax_amount, reconcile, reconciled, amount_residual,
		       matching_number, statement_line_id, display_type, cogs_origin_id, created_at, updated_at
		FROM account_move_lines
		WHERE move_id = $1
		ORDER BY id ASC
	`
	rows, err := r.pool.Query(ctx, linesQuery, id)
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch move lines", err)
	}
	defer rows.Close()

	for rows.Next() {
		var l accounting.AccountMoveLine
		if err := rows.Scan(
			&l.ID, &l.MoveID, &l.AccountID, &l.PartnerID, &l.ProductID, &l.Name,
			&l.Quantity, &l.PriceUnit, &l.Discount, &l.Debit, &l.Credit, &l.Balance,
			&l.TaxIDs, &l.TaxAmount, &l.Reconcile, &l.Reconciled, &l.AmountResidual,
			&l.MatchingNumber, &l.StatementLineID, &l.DisplayType, &l.CogsOriginID, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan move line", err)
		}
		m.Lines = append(m.Lines, l)
	}
	return m, nil
}

func (r *PostgresRepo) UpdateMove(ctx context.Context, m *accounting.AccountMove) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			UPDATE account_moves
			SET name = $1, move_type = $2, journal_id = $3, partner_id = $4, date = $5,
			    invoice_date = $6, invoice_date_due = $7, payment_term_id = $8,
			    state = $9, payment_state = $10, amount_untaxed = $11, amount_tax = $12,
			    amount_total = $13, amount_residual = $14, currency = $15, ref = $16,
			    updated_at = NOW(), updated_by = $17
			WHERE id = $18 AND active = true
			RETURNING updated_at
		`
		if err := tx.QueryRow(ctx, query,
			m.Name, string(m.MoveType), m.JournalID, m.PartnerID, m.Date,
			m.InvoiceDate, m.InvoiceDueDate, m.PaymentTermID,
			string(m.State), string(m.PaymentState), m.AmountUntaxed, m.AmountTax,
			m.AmountTotal, m.AmountResidual, m.Currency, m.Ref,
			m.Audit.UpdatedBy, m.ID,
		).Scan(&m.Audit.UpdatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return platformerrors.NotFound(fmt.Sprintf("account move with id %d not found", m.ID))
			}
			return platformerrors.Internal("failed to update move", err)
		}

		if len(m.Lines) > 0 {
			if _, err := tx.Exec(ctx, `DELETE FROM account_move_lines WHERE move_id = $1`, m.ID); err != nil {
				return platformerrors.Internal("failed to replace move lines", err)
			}
			reconcileFlags, err := loadReconcileFlags(ctx, tx, m.Lines)
			if err != nil {
				return err
			}
			lineQuery := `
				INSERT INTO account_move_lines (
					move_id, account_id, partner_id, product_id, name,
					quantity, price_unit, discount, debit, credit, balance,
					tax_ids, tax_amount, reconcile, reconciled, amount_residual,
					matching_number, statement_line_id, display_type, cogs_origin_id, created_at, updated_at
				) VALUES (
					$1, $2, $3, $4, $5,
					$6, $7, $8, $9, $10, $11,
					$12, $13, $14, $15, $16,
					$17, $18, $19, $20, NOW(), NOW()
				) RETURNING id
			`
			for i := range m.Lines {
				l := &m.Lines[i]
				l.MoveID = m.ID
				l.Balance = l.Debit - l.Credit
				applyReconcileDefaults(l, reconcileFlags[l.AccountID])
				if err := tx.QueryRow(ctx, lineQuery,
					m.ID, l.AccountID, l.PartnerID, l.ProductID, l.Name,
					l.Quantity, l.PriceUnit, l.Discount, l.Debit, l.Credit, l.Balance,
					l.TaxIDs, l.TaxAmount, l.Reconcile, l.Reconciled, l.AmountResidual,
					l.MatchingNumber, l.StatementLineID, l.DisplayType, l.CogsOriginID,
				).Scan(&l.ID); err != nil {
					return platformerrors.Internal("failed to insert move line on update", err)
				}
			}
		}
		return nil
	})
}

func (r *PostgresRepo) DeleteMove(ctx context.Context, id int64) error {
	var state string
	checkQuery := `SELECT state FROM account_moves WHERE id = $1 AND active = true`
	if err := r.pool.QueryRow(ctx, checkQuery, id).Scan(&state); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("account move with id %d not found", id))
		}
		return platformerrors.Internal("failed to check move state", err)
	}

	if state == string(accounting.MoveStatePosted) {
		return platformerrors.Conflict("cannot delete a posted move; cancel or reverse it instead")
	}

	query := `UPDATE account_moves SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	if _, err := r.pool.Exec(ctx, query, id); err != nil {
		return platformerrors.Internal("failed to delete move", err)
	}
	return nil
}

func (r *PostgresRepo) ListMoves(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[accounting.AccountMove], error) {
	countQuery := `SELECT COUNT(*) FROM account_moves WHERE active = true`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return pagination.PageResult[accounting.AccountMove]{}, platformerrors.Internal("failed to count moves", err)
	}

	offset := page.Offset()
	limit := page.LimitClamped()
	query := `
		SELECT id, name, move_type, journal_id, partner_id, date,
		       invoice_date, invoice_date_due, payment_term_id, state, payment_state,
		       amount_untaxed, amount_tax, amount_total, amount_residual, currency,
		       COALESCE(ref, ''), reversed_entry_id, active, created_at, updated_at, created_by, updated_by
		FROM account_moves
		WHERE active = true
		ORDER BY date DESC, id DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return pagination.PageResult[accounting.AccountMove]{}, platformerrors.Internal("failed to list moves", err)
	}
	defer rows.Close()

	var moves []accounting.AccountMove
	for rows.Next() {
		var m accounting.AccountMove
		var mType, state, pState string
		if err := rows.Scan(
			&m.ID, &m.Name, &mType, &m.JournalID, &m.PartnerID, &m.Date,
			&m.InvoiceDate, &m.InvoiceDueDate, &m.PaymentTermID, &state, &pState,
			&m.AmountUntaxed, &m.AmountTax, &m.AmountTotal, &m.AmountResidual, &m.Currency,
			&m.Ref, &m.ReversedEntryID, &m.Active, &m.Audit.CreatedAt, &m.Audit.UpdatedAt, &m.Audit.CreatedBy, &m.Audit.UpdatedBy,
		); err != nil {
			return pagination.PageResult[accounting.AccountMove]{}, platformerrors.Internal("failed to scan move", err)
		}
		m.MoveType = accounting.MoveType(mType)
		m.State = accounting.MoveState(state)
		m.PaymentState = accounting.PaymentState(pState)
		moves = append(moves, m)
	}

	return pagination.NewPageResult(moves, total, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Move Lines & Reconciliation
// ─────────────────────────────────────────────────────────────────────────────

// scanMoveLineColumns lists the persisted move-line columns in scan order.
const moveLineColumns = `id, move_id, account_id, partner_id, product_id, name,
		quantity, price_unit, discount, debit, credit, balance,
		tax_ids, tax_amount, reconcile, reconciled, amount_residual,
		matching_number, statement_line_id, display_type, cogs_origin_id, created_at, updated_at`

func (r *PostgresRepo) GetMoveLineByID(ctx context.Context, id int64) (*accounting.AccountMoveLine, error) {
	query := `
		SELECT ` + moveLineColumns + `
		FROM account_move_lines
		WHERE id = $1
	`
	var l accounting.AccountMoveLine
	if err := r.pool.QueryRow(ctx, query, id).Scan(
		&l.ID, &l.MoveID, &l.AccountID, &l.PartnerID, &l.ProductID, &l.Name,
		&l.Quantity, &l.PriceUnit, &l.Discount, &l.Debit, &l.Credit, &l.Balance,
		&l.TaxIDs, &l.TaxAmount, &l.Reconcile, &l.Reconciled, &l.AmountResidual,
		&l.MatchingNumber, &l.StatementLineID, &l.DisplayType, &l.CogsOriginID, &l.CreatedAt, &l.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("account move line with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch move line", err)
	}
	return &l, nil
}

func (r *PostgresRepo) UpdateMoveLine(ctx context.Context, l *accounting.AccountMoveLine) error {
	query := `
		UPDATE account_move_lines
		SET account_id = $2, partner_id = $3, product_id = $4, name = $5,
		    quantity = $6, price_unit = $7, discount = $8, debit = $9, credit = $10,
		    balance = $11, tax_ids = $12, tax_amount = $13,
		    reconcile = $14, reconciled = $15, amount_residual = $16,
		    matching_number = $17, statement_line_id = $18, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	if err := r.pool.QueryRow(ctx, query,
		l.ID, l.AccountID, l.PartnerID, l.ProductID, l.Name,
		l.Quantity, l.PriceUnit, l.Discount, l.Debit, l.Credit, l.Balance,
		l.TaxIDs, l.TaxAmount, l.Reconcile, l.Reconciled, l.AmountResidual,
		l.MatchingNumber, l.StatementLineID,
	).Scan(&l.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("account move line with id %d not found", l.ID))
		}
		return platformerrors.Internal("failed to update move line", err)
	}
	return nil
}

func (r *PostgresRepo) UpdateMoveLineReconcile(ctx context.Context, id int64, reconciled bool, residual float64, matchingNumber *string) error {
	query := `
		UPDATE account_move_lines
		SET reconciled = $2, amount_residual = $3, matching_number = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`
	var updatedID int64
	if err := r.pool.QueryRow(ctx, query, id, reconciled, residual, matchingNumber).Scan(&updatedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("account move line with id %d not found", id))
		}
		return platformerrors.Internal("failed to update move line reconciliation state", err)
	}
	return nil
}

// ListReconcilableMoveLines returns posted, reconcilable, unmatched move lines for a partner.
func (r *PostgresRepo) ListReconcilableMoveLines(ctx context.Context, partnerID *int64, excludeLineIDs []int64, limit int) ([]accounting.AccountMoveLine, error) {
	if limit <= 0 {
		limit = 50
	}
	var exclude []int64
	if len(excludeLineIDs) > 0 {
		exclude = excludeLineIDs
	}

	query := `
		SELECT ` + moveLineColumns + `
		FROM account_move_lines l
		INNER JOIN account_moves m ON m.id = l.move_id AND m.state = 'posted' AND m.active = true
		WHERE l.reconcile = true
		  AND l.reconciled = false
		  AND l.amount_residual > 0.004
		  AND l.statement_line_id IS NULL
		  AND ($1::bigint IS NULL OR l.partner_id = $1)
		  AND ($2::bigint[] IS NULL OR NOT (l.id = ANY($2)))
		ORDER BY l.id ASC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, query, partnerID, exclude, limit)
	if err != nil {
		return nil, platformerrors.Internal("failed to list reconcilable move lines", err)
	}
	defer rows.Close()

	var lines []accounting.AccountMoveLine
	for rows.Next() {
		var l accounting.AccountMoveLine
		if err := rows.Scan(
			&l.ID, &l.MoveID, &l.AccountID, &l.PartnerID, &l.ProductID, &l.Name,
			&l.Quantity, &l.PriceUnit, &l.Discount, &l.Debit, &l.Credit, &l.Balance,
			&l.TaxIDs, &l.TaxAmount, &l.Reconcile, &l.Reconciled, &l.AmountResidual,
			&l.MatchingNumber, &l.StatementLineID, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan reconcilable move line", err)
		}
		lines = append(lines, l)
	}
	return lines, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Financial Reports
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) GetTrialBalance(ctx context.Context, fromDate, toDate time.Time, onlyPosted bool) (*accounting.TrialBalanceReport, error) {
	stateFilter := ""
	if onlyPosted {
		stateFilter = "AND m.state = 'posted'"
	}

	query := fmt.Sprintf(`
		SELECT
			a.id, a.code, a.name, a.type,
			COALESCE(SUM(CASE WHEN $1::date IS NOT NULL AND m.date < $1 THEN l.debit - l.credit ELSE 0 END), 0) AS initial_bal,
			COALESCE(SUM(CASE WHEN ($1::date IS NULL OR m.date >= $1) AND ($2::date IS NULL OR m.date <= $2) THEN l.debit ELSE 0 END), 0) AS period_debit,
			COALESCE(SUM(CASE WHEN ($1::date IS NULL OR m.date >= $1) AND ($2::date IS NULL OR m.date <= $2) THEN l.credit ELSE 0 END), 0) AS period_credit
		FROM account_accounts a
		LEFT JOIN account_move_lines l ON l.account_id = a.id
		LEFT JOIN account_moves m ON m.id = l.move_id AND m.active = true %s
		WHERE a.active = true
		GROUP BY a.id, a.code, a.name, a.type
		ORDER BY a.code ASC
	`, stateFilter)

	var fDate, tDate *time.Time
	if !fromDate.IsZero() {
		fDate = &fromDate
	}
	if !toDate.IsZero() {
		tDate = &toDate
	}

	rows, err := r.pool.Query(ctx, query, fDate, tDate)
	if err != nil {
		return nil, platformerrors.Internal("failed to generate trial balance query", err)
	}
	defer rows.Close()

	report := &accounting.TrialBalanceReport{
		FromDate: fromDate,
		ToDate:   toDate,
		Lines:    make([]accounting.TrialBalanceLine, 0),
	}

	for rows.Next() {
		var line accounting.TrialBalanceLine
		var aType string
		if err := rows.Scan(&line.AccountID, &line.AccountCode, &line.AccountName, &aType, &line.InitialBalance, &line.Debit, &line.Credit); err != nil {
			return nil, platformerrors.Internal("failed to scan trial balance line", err)
		}
		line.AccountType = accounting.AccountType(aType)
		line.EndingBalance = line.InitialBalance + line.Debit - line.Credit
		report.Lines = append(report.Lines, line)
	}

	report.ComputeTotals()
	return report, nil
}

func (r *PostgresRepo) GetProfitAndLoss(ctx context.Context, fromDate, toDate time.Time) (*accounting.ProfitAndLossReport, error) {
	query := `
		SELECT
			a.id, a.code, a.name, a.type,
			COALESCE(SUM(
				CASE
					WHEN a.type IN ('income', 'income_other') THEN l.credit - l.debit
					WHEN a.type IN ('expense', 'expense_depreciation', 'expense_direct_cost') THEN l.debit - l.credit
					ELSE 0
				END
			), 0) AS amount
		FROM account_accounts a
		INNER JOIN account_move_lines l ON l.account_id = a.id
		INNER JOIN account_moves m ON m.id = l.move_id AND m.active = true AND m.state = 'posted'
		WHERE a.active = true
		  AND a.type IN ('income', 'income_other', 'expense', 'expense_depreciation', 'expense_direct_cost')
		  AND ($1::date IS NULL OR m.date >= $1)
		  AND ($2::date IS NULL OR m.date <= $2)
		GROUP BY a.id, a.code, a.name, a.type
		HAVING SUM(l.debit + l.credit) > 0
		ORDER BY a.code ASC
	`
	var fDate, tDate *time.Time
	if !fromDate.IsZero() {
		fDate = &fromDate
	}
	if !toDate.IsZero() {
		tDate = &toDate
	}

	rows, err := r.pool.Query(ctx, query, fDate, tDate)
	if err != nil {
		return nil, platformerrors.Internal("failed to generate profit and loss query", err)
	}
	defer rows.Close()

	report := &accounting.ProfitAndLossReport{
		FromDate:     fromDate,
		ToDate:       toDate,
		IncomeLines:  make([]accounting.ReportLine, 0),
		ExpenseLines: make([]accounting.ReportLine, 0),
	}

	for rows.Next() {
		var l accounting.ReportLine
		var aType string
		if err := rows.Scan(&l.AccountID, &l.AccountCode, &l.AccountName, &aType, &l.Amount); err != nil {
			return nil, platformerrors.Internal("failed to scan p&l line", err)
		}
		l.AccountType = accounting.AccountType(aType)
		if l.AccountType.IsIncome() {
			report.IncomeLines = append(report.IncomeLines, l)
		} else if l.AccountType.IsExpense() {
			report.ExpenseLines = append(report.ExpenseLines, l)
		}
	}

	report.ComputeTotals()
	return report, nil
}

func (r *PostgresRepo) GetBalanceSheet(ctx context.Context, asOfDate time.Time) (*accounting.BalanceSheetReport, error) {
	if asOfDate.IsZero() {
		asOfDate = time.Now().UTC()
	}

	// 1. Assets, Liabilities, Equity lines
	query := `
		SELECT
			a.id, a.code, a.name, a.type,
			COALESCE(SUM(
				CASE
					WHEN a.type IN ('asset_receivable', 'asset_cash', 'asset_current', 'asset_non_current') THEN l.debit - l.credit
					WHEN a.type IN ('liability_payable', 'liability_current', 'liability_non_current', 'equity') THEN l.credit - l.debit
					ELSE 0
				END
			), 0) AS amount
		FROM account_accounts a
		INNER JOIN account_move_lines l ON l.account_id = a.id
		INNER JOIN account_moves m ON m.id = l.move_id AND m.active = true AND m.state = 'posted'
		WHERE a.active = true
		  AND a.type IN ('asset_receivable', 'asset_cash', 'asset_current', 'asset_non_current',
		                 'liability_payable', 'liability_current', 'liability_non_current', 'equity')
		  AND m.date <= $1
		GROUP BY a.id, a.code, a.name, a.type
		HAVING SUM(l.debit + l.credit) > 0
		ORDER BY a.code ASC
	`
	rows, err := r.pool.Query(ctx, query, asOfDate)
	if err != nil {
		return nil, platformerrors.Internal("failed to generate balance sheet query", err)
	}
	defer rows.Close()

	report := &accounting.BalanceSheetReport{
		AsOfDate:       asOfDate,
		AssetLines:     make([]accounting.ReportLine, 0),
		LiabilityLines: make([]accounting.ReportLine, 0),
		EquityLines:    make([]accounting.ReportLine, 0),
	}

	for rows.Next() {
		var l accounting.ReportLine
		var aType string
		if err := rows.Scan(&l.AccountID, &l.AccountCode, &l.AccountName, &aType, &l.Amount); err != nil {
			return nil, platformerrors.Internal("failed to scan balance sheet line", err)
		}
		l.AccountType = accounting.AccountType(aType)
		if l.AccountType.IsAsset() {
			report.AssetLines = append(report.AssetLines, l)
		} else if l.AccountType.IsLiability() {
			report.LiabilityLines = append(report.LiabilityLines, l)
		} else if l.AccountType.IsEquity() {
			report.EquityLines = append(report.EquityLines, l)
		}
	}

	// 2. Compute Retained Earnings from all historical Income - Expenses up to asOfDate
	retainedQuery := `
		SELECT COALESCE(SUM(
			CASE
				WHEN a.type IN ('income', 'income_other') THEN l.credit - l.debit
				WHEN a.type IN ('expense', 'expense_depreciation', 'expense_direct_cost') THEN -(l.debit - l.credit)
				ELSE 0
			END
		), 0)
		FROM account_accounts a
		INNER JOIN account_move_lines l ON l.account_id = a.id
		INNER JOIN account_moves m ON m.id = l.move_id AND m.active = true AND m.state = 'posted'
		WHERE a.active = true AND m.date <= $1
	`
	var retainedEarnings float64
	if err := r.pool.QueryRow(ctx, retainedQuery, asOfDate).Scan(&retainedEarnings); err != nil {
		return nil, platformerrors.Internal("failed to compute retained earnings for balance sheet", err)
	}
	report.RetainedEarnings = retainedEarnings

	report.ComputeTotals()
	return report, nil
}

func (r *PostgresRepo) GetGeneralLedger(ctx context.Context, accountID *int64, partnerID *int64, fromDate, toDate *time.Time) ([]accounting.GeneralLedgerItem, error) {
	query := `
		SELECT
			m.date, m.id, m.name, l.id, l.account_id, a.code,
			l.partner_id, COALESCE(p.name, ''), l.name, l.debit, l.credit
		FROM account_move_lines l
		INNER JOIN account_moves m ON m.id = l.move_id AND m.active = true AND m.state = 'posted'
		INNER JOIN account_accounts a ON a.id = l.account_id
		LEFT JOIN res_partners p ON p.id = l.partner_id
		WHERE ($1::bigint IS NULL OR l.account_id = $1)
		  AND ($2::bigint IS NULL OR l.partner_id = $2)
		  AND ($3::date IS NULL OR m.date >= $3)
		  AND ($4::date IS NULL OR m.date <= $4)
		ORDER BY m.date ASC, l.id ASC
	`
	rows, err := r.pool.Query(ctx, query, accountID, partnerID, fromDate, toDate)
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch general ledger", err)
	}
	defer rows.Close()

	var items []accounting.GeneralLedgerItem
	var runningBalance float64

	for rows.Next() {
		var it accounting.GeneralLedgerItem
		if err := rows.Scan(
			&it.Date, &it.MoveID, &it.MoveName, &it.LineID, &it.AccountID, &it.AccountCode,
			&it.PartnerID, &it.PartnerName, &it.Label, &it.Debit, &it.Credit,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan general ledger item", err)
		}
		runningBalance += (it.Debit - it.Credit)
		it.RunningBalance = runningBalance
		items = append(items, it)
	}
	return items, nil
}

// moveLineQueryer abstracts pgxpool.Pool and pgx.Tx so reconcile flags can be
// resolved either at top-level or inside a transaction.
type moveLineQueryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// loadReconcileFlags returns, per account ID, whether the account allows reconciliation.
func loadReconcileFlags(ctx context.Context, q moveLineQueryer, lines []accounting.AccountMoveLine) (map[int64]bool, error) {
	ids := make([]int64, 0, len(lines))
	seen := make(map[int64]struct{}, len(lines))
	for _, l := range lines {
		if _, ok := seen[l.AccountID]; ok {
			continue
		}
		seen[l.AccountID] = struct{}{}
		ids = append(ids, l.AccountID)
	}

	flags := make(map[int64]bool, len(ids))
	if len(ids) == 0 {
		return flags, nil
	}

	rows, err := q.Query(ctx, `SELECT id, reconcile FROM account_accounts WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, platformerrors.Internal("failed to load reconcile flags", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var reconcile bool
		if err := rows.Scan(&id, &reconcile); err != nil {
			return nil, platformerrors.Internal("failed to scan reconcile flags", err)
		}
		flags[id] = reconcile
	}
	return flags, nil
}

// applyReconcileDefaults fills the reconciliation fields of a freshly-persisted
// line from the underlying account unless the line already carries explicit state.
func applyReconcileDefaults(l *accounting.AccountMoveLine, accountReconcile bool) {
	if accountReconcile && !l.Reconciled {
		l.Reconcile = true
		if l.AmountResidual <= 0 {
			l.AmountResidual = math.Abs(l.Balance)
		}
	}
}
