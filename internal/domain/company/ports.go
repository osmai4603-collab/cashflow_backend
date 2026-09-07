package company

import (
	"context"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines the contract for persistent storage of Company entities.
type Repository interface {
	Create(ctx context.Context, c *Company) error
	GetByID(ctx context.Context, id int64) (*Company, error)
	Update(ctx context.Context, c *Company) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Company], error)
	GetDefaultCompany(ctx context.Context) (*Company, error)
}
