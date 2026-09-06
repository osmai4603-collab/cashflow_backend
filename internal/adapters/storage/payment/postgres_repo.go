package paymentstorage

import (
	"context"
	"errors"
	"fmt"

	"cashflow_backend/internal/domain/payment"
	"cashflow_backend/internal/platform/database"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedPaymentFilterFields = map[string]string{
	"partner_id":     "partner_id",
	"journal_id":     "journal_id",
	"payment_type":   "payment_type",
	"partner_type":   "partner_type",
	"payment_method": "payment_method",
	"state":          "state",
	"name":           "name",
	"active":         "active",
}

// PostgresRepo implements payment.Repository against a PostgreSQL database.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// ─────────────────────────────────────────────────────────────────────────────
// Payments CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreatePayment(ctx context.Context, p *payment.Payment) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			INSERT INTO account_payments (
				name, payment_type, partner_type, partner_id, amount, currency,
				payment_method, journal_id, date, state, ref, move_id,
				reconciled_amount, residual_amount, company_id, active, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11, $12,
				$13, $14, $15, $16, NOW(), NOW()
			)
			RETURNING id, created_at, updated_at
		`
		return tx.QueryRow(ctx, query,
			p.Name, string(p.PaymentType), string(p.PartnerType), p.PartnerID, p.Amount, p.Currency,
			string(p.PaymentMethod), p.JournalID, p.Date, string(p.State), p.Ref, p.MoveID,
			p.ReconciledAmount, p.ResidualAmount, p.CompanyID, p.Active,
		).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	})
}

func (r *PostgresRepo) GetPaymentByID(ctx context.Context, id int64) (*payment.Payment, error) {
	query := `
		SELECT
			id, name, payment_type, partner_type, partner_id, amount, currency,
			payment_method, journal_id, date, state, ref, move_id,
			reconciled_amount, residual_amount, company_id, active,
			created_at, updated_at, created_by, updated_by
		FROM account_payments
		WHERE id = $1 AND active = true
	`

	var (
		p           payment.Payment
		pType       string
		partnerType string
		pMethod     string
		state       string
	)

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &pType, &partnerType, &p.PartnerID, &p.Amount, &p.Currency,
		&pMethod, &p.JournalID, &p.Date, &state, &p.Ref, &p.MoveID,
		&p.ReconciledAmount, &p.ResidualAmount, &p.CompanyID, &p.Active,
		&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("payment with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch payment", err)
	}

	p.PaymentType = payment.PaymentType(pType)
	p.PartnerType = payment.PartnerType(partnerType)
	p.PaymentMethod = payment.PaymentMethod(pMethod)
	p.State = payment.PaymentState(state)

	return &p, nil
}

func (r *PostgresRepo) UpdatePayment(ctx context.Context, p *payment.Payment) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			UPDATE account_payments
			SET
				name = $1, payment_type = $2, partner_type = $3, partner_id = $4,
				amount = $5, currency = $6, payment_method = $7, journal_id = $8,
				date = $9, state = $10, ref = $11, move_id = $12,
				reconciled_amount = $13, residual_amount = $14, company_id = $15,
				updated_at = NOW(), updated_by = $16
			WHERE id = $17 AND active = true
			RETURNING updated_at
		`
		err := tx.QueryRow(ctx, query,
			p.Name, string(p.PaymentType), string(p.PartnerType), p.PartnerID,
			p.Amount, p.Currency, string(p.PaymentMethod), p.JournalID,
			p.Date, string(p.State), p.Ref, p.MoveID,
			p.ReconciledAmount, p.ResidualAmount, p.CompanyID,
			p.UpdatedBy, p.ID,
		).Scan(&p.UpdatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return platformerrors.NotFound(fmt.Sprintf("payment with id %d not found", p.ID))
			}
			return platformerrors.Internal("failed to update payment", err)
		}
		return nil
	})
}

func (r *PostgresRepo) DeletePayment(ctx context.Context, id int64) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `UPDATE account_payments SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
		tag, err := tx.Exec(ctx, query, id)
		if err != nil {
			return platformerrors.Internal("failed to delete payment", err)
		}
		if tag.RowsAffected() == 0 {
			return platformerrors.NotFound(fmt.Sprintf("payment with id %d not found", id))
		}

		// Delete reconciliations
		_, err = tx.Exec(ctx, `DELETE FROM account_payment_reconciliations WHERE payment_id = $1`, id)
		if err != nil {
			return platformerrors.Internal("failed to clean payment reconciliations", err)
		}
		return nil
	})
}

func (r *PostgresRepo) ListPayments(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[payment.Payment], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedPaymentFilterFields, 1)
	if err != nil {
		return pagination.PageResult[payment.Payment]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}

	if whereClause == "" {
		whereClause = "WHERE active = true"
	} else {
		whereClause += " AND active = true"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM account_payments %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[payment.Payment]{}, platformerrors.Internal("failed to count payments", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedPaymentFilterFields[page.SortBy]; ok {
			sortBy = col
		}
	}

	limit := page.LimitClamped()
	offset := page.Offset()

	dataQuery := fmt.Sprintf(`
		SELECT
			id, name, payment_type, partner_type, partner_id, amount, currency,
			payment_method, journal_id, date, state, ref, move_id,
			reconciled_amount, residual_amount, company_id, active,
			created_at, updated_at, created_by, updated_by
		FROM account_payments
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return pagination.PageResult[payment.Payment]{}, platformerrors.Internal("failed to list payments", err)
	}
	defer rows.Close()

	var items []payment.Payment
	for rows.Next() {
		var (
			p           payment.Payment
			pType       string
			partnerType string
			pMethod     string
			state       string
		)

		if err := rows.Scan(
			&p.ID, &p.Name, &pType, &partnerType, &p.PartnerID, &p.Amount, &p.Currency,
			&pMethod, &p.JournalID, &p.Date, &state, &p.Ref, &p.MoveID,
			&p.ReconciledAmount, &p.ResidualAmount, &p.CompanyID, &p.Active,
			&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
		); err != nil {
			return pagination.PageResult[payment.Payment]{}, platformerrors.Internal("failed to scan payment", err)
		}

		p.PaymentType = payment.PaymentType(pType)
		p.PartnerType = payment.PartnerType(partnerType)
		p.PaymentMethod = payment.PaymentMethod(pMethod)
		p.State = payment.PaymentState(state)

		items = append(items, p)
	}

	if err := rows.Err(); err != nil {
		return pagination.PageResult[payment.Payment]{}, platformerrors.Internal("error iterating payments", err)
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}

func (r *PostgresRepo) NextSequence(ctx context.Context, year int) (string, error) {
	var seqVal int64
	query := `SELECT nextval('account_payment_seq')`
	if err := r.pool.QueryRow(ctx, query).Scan(&seqVal); err != nil {
		return "", platformerrors.Internal("failed to generate payment sequence", err)
	}
	return fmt.Sprintf("PAY/%04d/%05d", year, seqVal), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Reconciliations
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateReconciliation(ctx context.Context, rec *payment.PaymentReconciliation) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			INSERT INTO account_payment_reconciliations (
				payment_id, invoice_id, amount, reconciled_at
			) VALUES ($1, $2, $3, $4)
			RETURNING id
		`
		return tx.QueryRow(ctx, query,
			rec.PaymentID, rec.InvoiceID, rec.Amount, rec.ReconciledAt,
		).Scan(&rec.ID)
	})
}

func (r *PostgresRepo) GetReconciliationsByPaymentID(ctx context.Context, paymentID int64) ([]payment.PaymentReconciliation, error) {
	query := `
		SELECT id, payment_id, invoice_id, amount, reconciled_at
		FROM account_payment_reconciliations
		WHERE payment_id = $1
		ORDER BY id ASC
	`
	rows, err := r.pool.Query(ctx, query, paymentID)
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch reconciliations by payment", err)
	}
	defer rows.Close()

	var list []payment.PaymentReconciliation
	for rows.Next() {
		var rec payment.PaymentReconciliation
		if err := rows.Scan(&rec.ID, &rec.PaymentID, &rec.InvoiceID, &rec.Amount, &rec.ReconciledAt); err != nil {
			return nil, platformerrors.Internal("failed to scan reconciliation", err)
		}
		list = append(list, rec)
	}

	return list, nil
}

func (r *PostgresRepo) GetReconciliationsByInvoiceID(ctx context.Context, invoiceID int64) ([]payment.PaymentReconciliation, error) {
	query := `
		SELECT id, payment_id, invoice_id, amount, reconciled_at
		FROM account_payment_reconciliations
		WHERE invoice_id = $1
		ORDER BY id ASC
	`
	rows, err := r.pool.Query(ctx, query, invoiceID)
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch reconciliations by invoice", err)
	}
	defer rows.Close()

	var list []payment.PaymentReconciliation
	for rows.Next() {
		var rec payment.PaymentReconciliation
		if err := rows.Scan(&rec.ID, &rec.PaymentID, &rec.InvoiceID, &rec.Amount, &rec.ReconciledAt); err != nil {
			return nil, platformerrors.Internal("failed to scan reconciliation", err)
		}
		list = append(list, rec)
	}

	return list, nil
}

func (r *PostgresRepo) DeleteReconciliationsByPaymentID(ctx context.Context, paymentID int64) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `DELETE FROM account_payment_reconciliations WHERE payment_id = $1`
		_, err := tx.Exec(ctx, query, paymentID)
		if err != nil {
			return platformerrors.Internal("failed to delete reconciliations", err)
		}
		return nil
	})
}
