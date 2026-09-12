package repairstorage

import (
	"context"
	"sort"
	"sync"

	"cashflow_backend/internal/domain/repair"
)

type MemoryRepo struct {
	mu     sync.RWMutex
	orders map[int64]*repair.RepairOrder
	lines  map[int64]*repair.RepairOrderLine
	nextID int64
}

func (r *MemoryRepo) allocateID() int64 {
	r.nextID++
	return r.nextID
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
	if order.ID == 0 {
		order.ID = r.allocateID()
	}
	r.orders[order.ID] = order
	return nil
}

func (r *MemoryRepo) GetOrderByID(ctx context.Context, id int64) (*repair.RepairOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, ok := r.orders[id]
	if !ok {
		return nil, nil
	}
	return order, nil
}

func (r *MemoryRepo) GetOrderByName(ctx context.Context, name string) (*repair.RepairOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, order := range r.orders {
		if order.Name == name {
			return order, nil
		}
	}
	return nil, nil
}

func (r *MemoryRepo) UpdateOrder(ctx context.Context, order *repair.RepairOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if order.ID == 0 {
		order.ID = r.allocateID()
	}
	r.orders[order.ID] = order
	return nil
}

func (r *MemoryRepo) DeleteOrder(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.orders, id)
	return nil
}

func (r *MemoryRepo) ListOrders(ctx context.Context, companyID int64) ([]repair.RepairOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]repair.RepairOrder, 0, len(r.orders))
	for _, order := range r.orders {
		if companyID == 0 || order.CompanyID == companyID {
			result = append(result, *order)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryRepo) CreateOrderLine(ctx context.Context, line *repair.RepairOrderLine) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if line.ID == 0 {
		line.ID = r.allocateID()
	}
	r.lines[line.ID] = line
	return nil
}

func (r *MemoryRepo) GetLinesByRepairID(ctx context.Context, repairID int64) ([]repair.RepairOrderLine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]repair.RepairOrderLine, 0)
	for _, line := range r.lines {
		if line.RepairID == repairID {
			result = append(result, *line)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *MemoryRepo) DeleteOrderLine(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lines, id)
	return nil
}
