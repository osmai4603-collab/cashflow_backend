package delivery

import (
	"context"
	"sync"

	"cashflow_backend/internal/domain/delivery"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	carriers map[int64]*delivery.DeliveryCarrier
	nextID   int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		carriers: make(map[int64]*delivery.DeliveryCarrier),
		nextID:   1,
	}
}

func (r *MemoryRepository) CreateCarrier(ctx context.Context, c *delivery.DeliveryCarrier) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c.ID = r.nextID
	r.nextID++
	r.carriers[c.ID] = c
	return nil
}

func (r *MemoryRepository) GetCarrierByID(ctx context.Context, id int64) (*delivery.DeliveryCarrier, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.carriers[id]
	if !ok {
		return nil, platformerrors.NotFound("carrier not found")
	}
	return c, nil
}

func (r *MemoryRepository) ListCarriers(ctx context.Context, filters map[string]interface{}) ([]delivery.DeliveryCarrier, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []delivery.DeliveryCarrier
	for _, c := range r.carriers {
		if c.Active {
			list = append(list, *c)
		}
	}
	return list, nil
}

func (r *MemoryRepository) UpdateCarrier(ctx context.Context, c *delivery.DeliveryCarrier) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.carriers[c.ID] = c
	return nil
}

func (r *MemoryRepository) DeleteCarrier(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.carriers, id)
	return nil
}

func (r *MemoryRepository) CreatePriceRule(ctx context.Context, rule *delivery.DeliveryPriceRule) error {
	return nil
}

func (r *MemoryRepository) DeletePriceRule(ctx context.Context, id int64) error {
	return nil
}

func (r *MemoryRepository) GetZipPrefixes(ctx context.Context, ids []int64) ([]delivery.DeliveryZipPrefix, error) {
	return nil, nil
}

func (r *MemoryRepository) CreateZipPrefix(ctx context.Context, name string) (*delivery.DeliveryZipPrefix, error) {
	return &delivery.DeliveryZipPrefix{ID: 1, Name: name}, nil
}
