package user

import (
	"context"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines the contract for persistent storage of User entities.
type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByLogin(ctx context.Context, login string) (*User, error)
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[User], error)
	UpdateLastLogin(ctx context.Context, userID int64) error
}
