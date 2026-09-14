package cleanarch

import (
	"context"
	"database/sql"
	"fmt"
)

// =============================================================================
// LAYER 3 & 4: ADAPTERS & INFRASTRUCTURE LAYER (Secondary / Driven Adapter)
// =============================================================================
// Rules:
// - Implements the InvoiceRepository interface declared by the use case layer.
// - Knows SQL, database table schemas, and serialization details.
// - Maps raw database rows to pure Domain Entities.
// =============================================================================

type PostgresInvoiceRepository struct {
	db *sql.DB
}

func NewPostgresInvoiceRepository(db *sql.DB) *PostgresInvoiceRepository {
	return &PostgresInvoiceRepository{db: db}
}

func (r *PostgresInvoiceRepository) GetByID(ctx context.Context, id string) (*Invoice, error) {
	query := `SELECT id, customer_id, amount, status FROM invoices WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var invID, customerID, statusStr string
	var amount float64

	if err := row.Scan(&invID, &customerID, &amount, &statusStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query invoice %s: %w", id, err)
	}

	inv, err := NewInvoice(invID, customerID, amount)
	if err != nil {
		return nil, fmt.Errorf("reconstitute invoice entity: %w", err)
	}
	if InvoiceStatus(statusStr) == InvoiceStatusPaid {
		_ = inv.MarkAsPaid(inv.CreatedAt())
	}

	return inv, nil
}

func (r *PostgresInvoiceRepository) Save(ctx context.Context, invoice *Invoice) error {
	query := `
		INSERT INTO invoices (id, customer_id, amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status;
	`
	_, err := r.db.ExecContext(ctx, query,
		invoice.ID(),
		invoice.CustomerID(),
		invoice.Amount(),
		string(invoice.Status()),
		invoice.CreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("upsert invoice %s: %w", invoice.ID(), err)
	}
	return nil
}

func (r *PostgresInvoiceRepository) NextID(ctx context.Context) (string, error) {
	var seq int64
	if err := r.db.QueryRowContext(ctx, "SELECT nextval('invoice_id_seq')").Scan(&seq); err != nil {
		return "", fmt.Errorf("generate next sequence id: %w", err)
	}
	return fmt.Sprintf("INV-%06d", seq), nil
}
