package storage

import (
	"context"
	"errors"
	"sync"

	"cashflow_backend/internal/domain"
)

var ErrStorageClosed = errors.New("storage is closed")

// MemoryTransactionRepo is a thread-safe in-memory implementation of domain.TransactionRepository.
type MemoryTransactionRepo struct {
	mu     sync.RWMutex
	items  map[string]*domain.Transaction
	closed bool
}

// NewMemoryTransactionRepo creates a new in-memory repository instance.
func NewMemoryTransactionRepo() *MemoryTransactionRepo {
	return &MemoryTransactionRepo{
		items: make(map[string]*domain.Transaction),
	}
}

func (r *MemoryTransactionRepo) Save(ctx context.Context, tx *domain.Transaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrStorageClosed
	}

	// Store copy to prevent caller mutation race conditions
	copied := *tx
	r.items[tx.ID] = &copied
	return nil
}

func (r *MemoryTransactionRepo) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return nil, ErrStorageClosed
	}

	tx, exists := r.items[id]
	if !exists {
		return nil, domain.ErrTransactionNotFound
	}

	copied := *tx
	return &copied, nil
}

func (r *MemoryTransactionRepo) FindAll(ctx context.Context) ([]*domain.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return nil, ErrStorageClosed
	}

	result := make([]*domain.Transaction, 0, len(r.items))
	for _, tx := range r.items {
		copied := *tx
		result = append(result, &copied)
	}
	return result, nil
}

// Ping checks if the storage is accessible (used by readiness health checks).
func (r *MemoryTransactionRepo) Ping(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.closed {
		return ErrStorageClosed
	}
	return nil
}

// Close gracefully closes the storage adapter (used in phase 7 cleanup).
func (r *MemoryTransactionRepo) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closed = true
	return nil
}
