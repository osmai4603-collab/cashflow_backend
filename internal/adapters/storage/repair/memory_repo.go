package repairstorage

import (
	"context"
	"cashflow_backend/internal/domain/repair"
	"sync"
)

type MemoryRepo struct {
	mu sync.RWMutex
	orders map[int64]*repair.RepairOrder
	lines  map[int64]*repair.RepairOrderLine
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		orders: make(map[int64]*repair.RepairOrder),
		lines:  make(map[int64]*repair.RepairOrderLine),
	}
}

func (r *MemoryRepo) CreateOrder(ctx context.Context, order *repair.RepairOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}
func (r *MemoryRepo) GetOrderByID(ctx context.Context, id int64) (*repair.RepairOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.orders[id], nil
}
func (r *MemoryRepo) GetOrderByName(ctx context.Context, name string) (*repair.RepairOrder, error) { return nil, nil }
func (r *MemoryRepo) UpdateOrder(ctx context.Context, order *repair.RepairOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}
func (r *MemoryRepo) DeleteOrder(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.orders, id)
	return nil
}
func (r *MemoryRepo) ListOrders(ctx context.Context, companyID int64) ([]repair.RepairOrder, error) { return nil, nil }
func (r *MemoryRepo) CreateOrderLine(ctx context.Context, line *repair.RepairOrderLine) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lines[line.ID] = line
	return nil
}
func (r *MemoryRepo) GetLinesByRepairID(ctx context.Context, repairID int64) ([]repair.RepairOrderLine, error) { return nil, nil }
func (r *MemoryRepo) DeleteOrderLine(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lines, id)
	return nil
}
