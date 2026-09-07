package mrpstorage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cashflow_backend/internal/domain/mrp"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type MemoryRepo struct {
	mu          sync.RWMutex
	workcenters map[int64]*mrp.Workcenter
	boms        map[int64]*mrp.BillOfMaterials
	wcNextID    int64
	bomNextID   int64
	moNextID    int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		workcenters: make(map[int64]*mrp.Workcenter),
		boms:        make(map[int64]*mrp.BillOfMaterials),
		wcNextID:    1,
		bomNextID:   1,
		moNextID:    1,
	}
}

func (r *MemoryRepo) CreateWorkcenter(ctx context.Context, wc *mrp.Workcenter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	wc.ID = r.wcNextID
	r.wcNextID++
	r.workcenters[wc.ID] = wc
	return nil
}

func (r *MemoryRepo) UpdateWorkcenter(ctx context.Context, wc *mrp.Workcenter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.workcenters[wc.ID]; !ok {
		return platformerrors.NotFound("workcenter not found", nil)
	}
	r.workcenters[wc.ID] = wc
	return nil
}

func (r *MemoryRepo) GetWorkcenterByID(ctx context.Context, id int64) (*mrp.Workcenter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	wc, ok := r.workcenters[id]
	if !ok {
		return nil, platformerrors.NotFound("workcenter not found", nil)
	}
	return wc, nil
}

func (r *MemoryRepo) ListWorkcenters(ctx context.Context, f mrp.WorkcenterFilter) ([]*mrp.Workcenter, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []*mrp.Workcenter
	for _, wc := range r.workcenters {
		res = append(res, wc)
	}
	return res, int64(len(res)), nil
}

func (r *MemoryRepo) DeleteWorkcenter(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.workcenters, id)
	return nil
}

func (r *MemoryRepo) CreateBoM(ctx context.Context, bom *mrp.BillOfMaterials) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	bom.ID = r.bomNextID
	r.bomNextID++
	r.boms[bom.ID] = bom
	return nil
}

func (r *MemoryRepo) UpdateBoM(ctx context.Context, bom *mrp.BillOfMaterials) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.boms[bom.ID]; !ok {
		return platformerrors.NotFound("bom not found", nil)
	}
	r.boms[bom.ID] = bom
	return nil
}

func (r *MemoryRepo) GetBoMByID(ctx context.Context, id int64) (*mrp.BillOfMaterials, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	bom, ok := r.boms[id]
	if !ok {
		return nil, platformerrors.NotFound("bom not found", nil)
	}
	return bom, nil
}

func (r *MemoryRepo) ListBoMs(ctx context.Context, f mrp.BoMFilter) ([]*mrp.BillOfMaterials, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []*mrp.BillOfMaterials
	for _, bom := range r.boms {
		res = append(res, bom)
	}
	return res, int64(len(res)), nil
}

func (r *MemoryRepo) DeleteBoM(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.boms, id)
	return nil
}

func (r *MemoryRepo) AddBomLine(ctx context.Context, line *mrp.BomLine) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	bom, ok := r.boms[line.BomID]
	if !ok {
		return fmt.Errorf("bom not found")
	}
	line.ID = time.Now().UnixNano()
	bom.Lines = append(bom.Lines, *line)
	return nil
}

func (r *MemoryRepo) RemoveBomLine(ctx context.Context, id int64) error {
	return nil // Mock
}

func (r *MemoryRepo) AddRoutingOperation(ctx context.Context, op *mrp.RoutingOperation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	bom, ok := r.boms[op.BomID]
	if !ok {
		return fmt.Errorf("bom not found")
	}
	op.ID = time.Now().UnixNano()
	bom.Operations = append(bom.Operations, *op)
	return nil
}

func (r *MemoryRepo) RemoveRoutingOperation(ctx context.Context, id int64) error {
	return nil // Mock
}

func (r *MemoryRepo) CreateProduction(ctx context.Context, mo *mrp.ProductionOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	mo.ID = r.moNextID
	r.moNextID++
	return nil
}

func (r *MemoryRepo) UpdateProduction(ctx context.Context, mo *mrp.ProductionOrder) error {
	return nil
}

func (r *MemoryRepo) GetProductionByID(ctx context.Context, id int64) (*mrp.ProductionOrder, error) {
	return &mrp.ProductionOrder{ID: id}, nil
}

func (r *MemoryRepo) ListProductions(ctx context.Context, f mrp.ProductionFilter) ([]*mrp.ProductionOrder, int64, error) {
	return nil, 0, nil
}

func (r *MemoryRepo) CreateWorkorder(ctx context.Context, wo *mrp.Workorder) error {
	return nil
}

func (r *MemoryRepo) UpdateWorkorder(ctx context.Context, wo *mrp.Workorder) error {
	return nil
}

func (r *MemoryRepo) GetWorkorderByID(ctx context.Context, id int64) (*mrp.Workorder, error) {
	return &mrp.Workorder{ID: id}, nil
}

func (r *MemoryRepo) ListWorkorders(ctx context.Context, productionID int64) ([]*mrp.Workorder, error) {
	return nil, nil
}

func (r *MemoryRepo) CreateUnbuild(ctx context.Context, uo *mrp.UnbuildOrder) error {
	return nil
}

func (r *MemoryRepo) GetUnbuildByID(ctx context.Context, id int64) (*mrp.UnbuildOrder, error) {
	return &mrp.UnbuildOrder{ID: id}, nil
}
