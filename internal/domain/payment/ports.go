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

	// ─── External Transactions ────────────────────────────────────────────
	CreateTransaction(ctx context.Context, t *PaymentTransaction) error
	GetTransactionByID(ctx context.Context, id int64) (*PaymentTransaction, error)
	GetTransactionByReference(ctx context.Context, ref string) (*PaymentTransaction, error)
	UpdateTransaction(ctx context.Context, t *PaymentTransaction) error
	GetProviderByCode(ctx context.Context, code string, companyID int64) (*PaymentProvider, error)
}
