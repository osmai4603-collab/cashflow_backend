package sale

import (
	"context"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines the persistent storage contract for sale orders and order lines.
type Repository interface {
	// CreateOrder persists a new sale order along with its lines in a single atomic transaction.
	CreateOrder(ctx context.Context, order *SaleOrder) error

	// GetOrderByID retrieves an order by its ID, including all lines and linked invoice IDs.
	GetOrderByID(ctx context.Context, id int64) (*SaleOrder, error)

	// GetOrderByName retrieves an order by its sequence name (e.g. "SO/2026/00001").
	GetOrderByName(ctx context.Context, name string) (*SaleOrder, error)

	// UpdateOrder updates the master order and synchronizes its lines atomically.
	UpdateOrder(ctx context.Context, order *SaleOrder) error

	// DeleteOrder deletes a draft or cancelled order and its lines.
	DeleteOrder(ctx context.Context, id int64) error

	// ListOrders returns a paginated list of orders matching optional filters.
	ListOrders(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[SaleOrder], error)

	// NextSequence generates a thread-safe permanent sequence number formatted as "SO/YYYY/NNNNN".
	NextSequence(ctx context.Context, year int) (string, error)

	// LinkInvoice links an accounting move (customer invoice) to a sale order.
	LinkInvoice(ctx context.Context, orderID int64, moveID int64) error

	// GetLinkedInvoiceIDs returns all accounting move IDs linked to a sale order.
	GetLinkedInvoiceIDs(ctx context.Context, orderID int64) ([]int64, error)
}
