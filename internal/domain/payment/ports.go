package payment

import (
	"context"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines the persistent storage contract for all Payment operations.
type Repository interface {
	// ─── Payments ─────────────────────────────────────────────────────────
	CreatePayment(ctx context.Context, p *Payment) error
	GetPaymentByID(ctx context.Context, id int64) (*Payment, error)
	UpdatePayment(ctx context.Context, p *Payment) error
	DeletePayment(ctx context.Context, id int64) error
	ListPayments(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Payment], error)
	NextSequence(ctx context.Context, year int) (string, error)

	// ─── Reconciliations ──────────────────────────────────────────────────
	CreateReconciliation(ctx context.Context, r *PaymentReconciliation) error
	GetReconciliationsByPaymentID(ctx context.Context, paymentID int64) ([]PaymentReconciliation, error)
	GetReconciliationsByInvoiceID(ctx context.Context, invoiceID int64) ([]PaymentReconciliation, error)
	DeleteReconciliationsByPaymentID(ctx context.Context, paymentID int64) error
}
