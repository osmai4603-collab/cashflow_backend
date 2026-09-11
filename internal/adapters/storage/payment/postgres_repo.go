package paymentstorage

import (
	"context"
	"encoding/json"
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

func (r *PostgresRepo) CreateTransaction(ctx context.Context, t *payment.PaymentTransaction) error {
	metadata, err := json.Marshal(t.Metadata)
	if err != nil {
		return platformerrors.Validation("invalid transaction metadata", nil)
	}
	query := `INSERT INTO payment_transactions (reference, amount, currency, provider_id, partner_id, state, provider_reference, sale_order_id, invoice_id, payment_id, idempotency_key, return_url, webhook_received, last_error, metadata, company_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING id, created_at, updated_at`
	err = r.pool.QueryRow(ctx, query, t.Reference, t.Amount, t.Currency, t.ProviderID, t.PartnerID, string(t.State), t.ProviderReference, t.SaleOrderID, t.InvoiceID, t.PaymentID, t.IdempotencyKey, t.ReturnURL, t.WebhookReceived, t.LastError, metadata, t.CompanyID).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return platformerrors.Internal("failed to create payment transaction", err)
	}
	return nil
}

func (r *PostgresRepo) GetTransactionByID(ctx context.Context, id int64) (*payment.PaymentTransaction, error) {
	return r.getTransaction(ctx, "id = $1", id)
}

func (r *PostgresRepo) GetTransactionByReference(ctx context.Context, ref string) (*payment.PaymentTransaction, error) {
	return r.getTransaction(ctx, "reference = $1", ref)
}

func (r *PostgresRepo) UpdateTransaction(ctx context.Context, t *payment.PaymentTransaction) error {
	metadata, err := json.Marshal(t.Metadata)
	if err != nil {
		return platformerrors.Validation("invalid transaction metadata", nil)
	}
	query := `UPDATE payment_transactions SET state=$1, provider_reference=$2, sale_order_id=$3, invoice_id=$4, payment_id=$5, idempotency_key=$6, return_url=$7, webhook_received=$8, last_error=$9, metadata=$10, updated_at=NOW() WHERE id=$11 RETURNING updated_at`
	if err := r.pool.QueryRow(ctx, query, string(t.State), t.ProviderReference, t.SaleOrderID, t.InvoiceID, t.PaymentID, t.IdempotencyKey, t.ReturnURL, t.WebhookReceived, t.LastError, metadata, t.ID).Scan(&t.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("payment transaction %d not found", t.ID))
		}
		return platformerrors.Internal("failed to update payment transaction", err)
	}
	return nil
}

func (r *PostgresRepo) GetProviderByCode(ctx context.Context, code string, companyID int64) (*payment.PaymentProvider, error) {
	provider := &payment.PaymentProvider{}
	if err := r.pool.QueryRow(ctx, `SELECT id, name, code, state, active, company_id, created_at, updated_at FROM payment_providers WHERE code=$1 AND company_id=$2 AND active=true`, code, companyID).Scan(&provider.ID, &provider.Name, &provider.Code, &provider.State, &provider.Active, &provider.CompanyID, &provider.CreatedAt, &provider.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("payment provider %q not found", code))
		}
		return nil, platformerrors.Internal("failed to fetch payment provider", err)
	}
	return provider, nil
}

func (r *PostgresRepo) GetProviderByID(ctx context.Context, id int64) (*payment.PaymentProvider, error) {
	provider := &payment.PaymentProvider{}
	if err := r.pool.QueryRow(ctx, `SELECT id, name, code, state, active, company_id, created_at, updated_at FROM payment_providers WHERE id=$1`, id).Scan(&provider.ID, &provider.Name, &provider.Code, &provider.State, &provider.Active, &provider.CompanyID, &provider.CreatedAt, &provider.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("provider not found")
		}
		return nil, platformerrors.Internal("failed to fetch provider", err)
	}
	return provider, nil
}

func (r *PostgresRepo) getTransaction(ctx context.Context, predicate string, value any) (*payment.PaymentTransaction, error) {
	query := `SELECT id, reference, amount, currency, provider_id, partner_id, state, provider_reference, sale_order_id, invoice_id, payment_id, idempotency_key, return_url, webhook_received, last_error, metadata, created_at, updated_at, company_id FROM payment_transactions WHERE ` + predicate
	var transaction payment.PaymentTransaction
	var state string
	var metadata []byte
	err := r.pool.QueryRow(ctx, query, value).Scan(&transaction.ID, &transaction.Reference, &transaction.Amount, &transaction.Currency, &transaction.ProviderID, &transaction.PartnerID, &state, &transaction.ProviderReference, &transaction.SaleOrderID, &transaction.InvoiceID, &transaction.PaymentID, &transaction.IdempotencyKey, &transaction.ReturnURL, &transaction.WebhookReceived, &transaction.LastError, &metadata, &transaction.CreatedAt, &transaction.UpdatedAt, &transaction.CompanyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("payment transaction not found", nil)
		}
		return nil, platformerrors.Internal("failed to fetch payment transaction", err)
	}
	transaction.State = payment.TransactionState(state)
	if len(metadata) > 0 && string(metadata) != "null" {
		if err := json.Unmarshal(metadata, &transaction.Metadata); err != nil {
			return nil, platformerrors.Internal("failed to decode transaction metadata", err)
		}
	}
	return &transaction, nil
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
