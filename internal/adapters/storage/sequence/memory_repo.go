package sequencestorage

import (
	"context"
	"sort"
	"strings"
	"sync"

	"cashflow_backend/internal/domain/sequence"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// MemoryRepo is a thread-safe in-memory implementation of sequence.Repository.
type MemoryRepo struct {
	mu        sync.Mutex
	sequences map[int64]*sequence.Sequence
	lastID    int64
}

// NewMemoryRepo creates an initialized MemoryRepo.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		sequences: make(map[int64]*sequence.Sequence),
	}
}

func (r *MemoryRepo) Create(ctx context.Context, s *sequence.Sequence) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastID++
	s.ID = r.lastID
	s.Active = true

	clone := *s
	r.sequences[s.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetByID(ctx context.Context, id int64) (*sequence.Sequence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, exists := r.sequences[id]
	if !exists || !s.Active {
		return nil, platformerrors.NotFound("sequence not found")
	}

	clone := *s
	return &clone, nil
}

func (r *MemoryRepo) GetByCode(ctx context.Context, code string) (*sequence.Sequence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, s := range r.sequences {
		if s.Active && strings.EqualFold(s.Code, code) {
			clone := *s
			return &clone, nil
		}
	}

	return nil, platformerrors.NotFound("sequence not found")
}

func (r *MemoryRepo) Update(ctx context.Context, s *sequence.Sequence) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.sequences[s.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound("sequence not found")
	}

	clone := *s
	r.sequences[s.ID] = &clone
	return nil
}

func (r *MemoryRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.sequences[id]
	if !exists || !existing.Active {
		return platformerrors.NotFound("sequence not found")
	}

	existing.Active = false
	return nil
}

func (r *MemoryRepo) List(ctx context.Context) ([]sequence.Sequence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var items []sequence.Sequence
	for _, s := range r.sequences {
		if !s.Active {
			continue
		}
		items = append(items, *s)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return items, nil
}

func (r *MemoryRepo) NextValue(ctx context.Context, id int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, exists := r.sequences[id]
	if !exists || !s.Active {
		return 0, platformerrors.NotFound("sequence not found")
	}

	next := s.NextNumber()
	s.CurrentNumber = next
	return next, nil
}

func (r *MemoryRepo) Reset(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, exists := r.sequences[id]
	if !exists || !s.Active {
		return platformerrors.NotFound("sequence not found")
	}

	s.CurrentNumber = s.StartNumber - s.IncrementBy
	return nil
}
