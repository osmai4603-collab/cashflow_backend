package currency

import (
	"context"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines the contract for persistent storage of Currency entities.
type Repository interface {
	Create(ctx context.Context, c *Currency) error
	GetByID(ctx context.Context, id int64) (*Currency, error)
	GetByName(ctx context.Context, name string) (*Currency, error)
	Update(ctx context.Context, c *Currency) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Currency], error)
}

// RateRepository defines the contract for persistent storage of CurrencyRate entities.
type RateRepository interface {
	Create(ctx context.Context, r *CurrencyRate) error
	GetByID(ctx context.Context, id int64) (*CurrencyRate, error)
	GetLatestRate(ctx context.Context, currencyID int64, companyID *int64) (*CurrencyRate, error)
	GetRateOnDate(ctx context.Context, currencyID int64, date string, companyID *int64) (*CurrencyRate, error)
	ListByCurrency(ctx context.Context, currencyID int64, page pagination.PageRequest) (pagination.PageResult[CurrencyRate], error)
}
