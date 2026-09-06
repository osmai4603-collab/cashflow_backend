package purchase

import (
	"context"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines the persistent storage contract for purchase orders and order lines.
type Repository interface {
	// CreateOrder persists a new purchase order along with its lines in a single atomic transaction.
	CreateOrder(ctx context.Context, order *PurchaseOrder) error

	// GetOrderByID retrieves an order by its ID, including all lines and linked bill IDs.
	GetOrderByID(ctx context.Context, id int64) (*PurchaseOrder, error)

	// GetOrderByName retrieves an order by its sequence name (e.g. "PO/2026/00001").
	GetOrderByName(ctx context.Context, name string) (*PurchaseOrder, error)

	// UpdateOrder updates the master order and synchronizes its lines atomically.
	UpdateOrder(ctx context.Context, order *PurchaseOrder) error

	// DeleteOrder deletes a draft or cancelled order and its lines.
	DeleteOrder(ctx context.Context, id int64) error

	// ListOrders returns a paginated list of orders matching optional filters.
	ListOrders(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[PurchaseOrder], error)

	// NextSequence generates a thread-safe permanent sequence number formatted as "PO/YYYY/NNNNN".
	NextSequence(ctx context.Context, year int) (string, error)

	// LinkBill links an accounting move (vendor bill) to a purchase order.
	LinkBill(ctx context.Context, orderID int64, moveID int64) error

	// GetLinkedBillIDs returns all accounting move IDs (vendor bills) linked to a purchase order.
	GetLinkedBillIDs(ctx context.Context, orderID int64) ([]int64, error)
}
