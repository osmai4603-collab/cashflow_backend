package sequence

import (
	"context"
)

// Repository defines the contract for persistent storage of Sequence entities.
type Repository interface {
	Create(ctx context.Context, s *Sequence) error
	GetByID(ctx context.Context, id int64) (*Sequence, error)
	GetByCode(ctx context.Context, code string) (*Sequence, error)
	Update(ctx context.Context, s *Sequence) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]Sequence, error)
	// NextValue atomically increments and returns the next number for the sequence.
	// Implementations must use SELECT ... FOR UPDATE or equivalent to prevent races.
	NextValue(ctx context.Context, id int64) (int, error)
	// Reset resets the current_number back to start_number - increment_by for the given sequence.
	Reset(ctx context.Context, id int64) error
}
