package partner

import (
	"context"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines the contract for persistent storage of Partner entities.
type Repository interface {
	// Create stores a new partner entity.
	Create(ctx context.Context, p *Partner) error

	// GetByID retrieves an active partner by its primary key identifier.
	GetByID(ctx context.Context, id int64) (*Partner, error)

	// Update modifies an existing partner entity.
	Update(ctx context.Context, p *Partner) error

	// Delete performs a soft-delete by marking active=false.
	Delete(ctx context.Context, id int64) error

	// List returns paginated partners matching optional domain filters.
	List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Partner], error)

	// ListCustomers returns paginated partners where is_customer is true and active is true.
	ListCustomers(ctx context.Context, page pagination.PageRequest) (pagination.PageResult[Partner], error)

	// ListSuppliers returns paginated partners where is_supplier is true and active is true.
	ListSuppliers(ctx context.Context, page pagination.PageRequest) (pagination.PageResult[Partner], error)
}
