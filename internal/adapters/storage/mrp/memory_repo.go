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
	mu             sync.RWMutex
	workcenters    map[int64]*mrp.Workcenter
	boms           map[int64]*mrp.BillOfMaterials
	productions    map[int64]*mrp.ProductionOrder
	workorders     map[int64]*mrp.Workorder
	timeLogs       map[int64][]mrp.WorkorderTimeLog
	calendars      map[int64][]mrp.WorkcenterCalendar
	wcNextID       int64
	bomNextID      int64
	moNextID       int64
	woNextID       int64
	timeLogNextID  int64
	calendarNextID int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		workcenters:    make(map[int64]*mrp.Workcenter),
		boms:           make(map[int64]*mrp.BillOfMaterials),
		productions:    make(map[int64]*mrp.ProductionOrder),
		workorders:     make(map[int64]*mrp.Workorder),
		timeLogs:       make(map[int64][]mrp.WorkorderTimeLog),
		calendars:      make(map[int64][]mrp.WorkcenterCalendar),
		wcNextID:       1,
		bomNextID:      1,
		moNextID:       1,
		woNextID:       1,
		timeLogNextID:  1,
		calendarNextID: 1,
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
	clone := *mo
	clone.Operations = append([]mrp.RoutingOperation(nil), mo.Operations...)
	r.productions[mo.ID] = &clone
	return nil
}

func (r *MemoryRepo) UpdateProduction(ctx context.Context, mo *mrp.ProductionOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.productions[mo.ID]; !ok {
		return platformerrors.NotFound("production not found", nil)
	}
	clone := *mo
	clone.Operations = append([]mrp.RoutingOperation(nil), mo.Operations...)
	r.productions[mo.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetProductionByID(ctx context.Context, id int64) (*mrp.ProductionOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	mo, ok := r.productions[id]
	if !ok {
		return nil, platformerrors.NotFound("production not found", nil)
	}
	clone := *mo
	clone.Operations = append([]mrp.RoutingOperation(nil), mo.Operations...)
	return &clone, nil
}

func (r *MemoryRepo) ListProductions(ctx context.Context, f mrp.ProductionFilter) ([]*mrp.ProductionOrder, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*mrp.ProductionOrder, 0, len(r.productions))
	for _, mo := range r.productions {
		clone := *mo
		result = append(result, &clone)
	}
	return result, int64(len(result)), nil
}

func (r *MemoryRepo) CreateWorkorder(ctx context.Context, wo *mrp.Workorder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.productions[wo.ProductionID]; !ok {
		return platformerrors.NotFound("production not found", nil)
	}
	wo.ID = r.woNextID
	r.woNextID++
	clone := *wo
	clone.TimeLogs = append([]mrp.WorkorderTimeLog(nil), wo.TimeLogs...)
	r.workorders[wo.ID] = &clone
	return nil
}

func (r *MemoryRepo) UpdateWorkorder(ctx context.Context, wo *mrp.Workorder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.workorders[wo.ID]; !ok {
		return platformerrors.NotFound("workorder not found", nil)
	}
	clone := *wo
	clone.TimeLogs = append([]mrp.WorkorderTimeLog(nil), wo.TimeLogs...)
	r.workorders[wo.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetWorkorderByID(ctx context.Context, id int64) (*mrp.Workorder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	wo, ok := r.workorders[id]
	if !ok {
		return nil, platformerrors.NotFound("workorder not found", nil)
	}
	clone := *wo
	clone.TimeLogs = append([]mrp.WorkorderTimeLog(nil), wo.TimeLogs...)
	return &clone, nil
}

func (r *MemoryRepo) ListWorkorders(ctx context.Context, productionID int64) ([]*mrp.Workorder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*mrp.Workorder, 0)
	for _, wo := range r.workorders {
		if wo.ProductionID == productionID {
			clone := *wo
			result = append(result, &clone)
		}
	}
	return result, nil
}

func (r *MemoryRepo) CreateWorkorderTimeLog(ctx context.Context, log *mrp.WorkorderTimeLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.workorders[log.WorkorderID]; !ok {
		return platformerrors.NotFound("workorder not found", nil)
	}
	log.ID = r.timeLogNextID
	r.timeLogNextID++
	r.timeLogs[log.WorkorderID] = append(r.timeLogs[log.WorkorderID], *log)
	return nil
}

func (r *MemoryRepo) ListWorkorderTimeLogs(ctx context.Context, workorderID int64) ([]mrp.WorkorderTimeLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]mrp.WorkorderTimeLog(nil), r.timeLogs[workorderID]...), nil
}

func (r *MemoryRepo) CreateWorkcenterCalendar(ctx context.Context, calendar *mrp.WorkcenterCalendar) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	calendar.ID = r.calendarNextID
	r.calendarNextID++
	r.calendars[calendar.WorkcenterID] = append(r.calendars[calendar.WorkcenterID], *calendar)
	return nil
}

func (r *MemoryRepo) ListWorkcenterCalendars(ctx context.Context, workcenterID int64) ([]mrp.WorkcenterCalendar, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]mrp.WorkcenterCalendar(nil), r.calendars[workcenterID]...), nil
}

func (r *MemoryRepo) CreateUnbuild(ctx context.Context, uo *mrp.UnbuildOrder) error {
	return nil
}

func (r *MemoryRepo) GetUnbuildByID(ctx context.Context, id int64) (*mrp.UnbuildOrder, error) {
	return &mrp.UnbuildOrder{ID: id}, nil
}
