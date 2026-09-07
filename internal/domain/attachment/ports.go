package attachment

import (
	"context"

	"cashflow_backend/internal/platform/pagination"
)

// Repository defines the contract for persistent storage of Attachment entities.
type Repository interface {
	Create(ctx context.Context, a *Attachment) error
	GetByID(ctx context.Context, id int64) (*Attachment, error)
	Update(ctx context.Context, a *Attachment) error
	Delete(ctx context.Context, id int64) error
	ListByModel(ctx context.Context, resModel string, resID int64, page pagination.PageRequest) (pagination.PageResult[Attachment], error)
}
