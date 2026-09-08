package stockstorage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/stock"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo provides a thread-safe, high-fidelity in-memory implementation of stock.Repository.
type MemoryRepo struct {
	mu             sync.RWMutex
	locations      map[int64]*stock.StockLocation
	warehouses     map[int64]*stock.Warehouse
	pickings       map[int64]*stock.StockPicking
	moves          map[int64]*stock.StockMove
	moveLines      map[int64]*stock.StockMoveLine
	lots           map[int64]*stock.StockLot
	quants         map[string]*stock.StockQuant // key: "productID:locationID"
	productValues  map[int64]*stock.ProductValue
	periods        map[int64]*stock.AccountingPeriod
	orderpoints    map[int64]*stock.Orderpoint
	landedCosts    map[int64]*stock.LandedCost
	procurements   map[int64]*stock.ProcurementGroup
	scraps         map[int64]*stock.StockScrap
	seqInCounters  map[int]int64
	seqOutCounters map[int]int64
	seqIntCounters map[int]int64
	opCounters     map[int]int64
	lcCounters     map[int]int64
	routes         map[int64]*stock.StockRoute
	rules          map[int64]*stock.StockRule
	lastLocID      int64
	lastWhID       int64
	lastPickingID  int64
	lastMoveID     int64
	lastMoveLineID int64
	lastLotID      int64
	lastQuantID    int64
	lastValueID    int64
	lastPeriodID   int64
	lastOpID       int64
	lastLCID       int64
	lastPGID       int64
	lastRouteID    int64
	lastRuleID     int64
	lastScrapID    int64
}

// NewMemoryRepo initializes a MemoryRepo pre-seeded with standard locations and warehouse.
func NewMemoryRepo() *MemoryRepo {
	repo := &MemoryRepo{
		locations:      make(map[int64]*stock.StockLocation),
		warehouses:     make(map[int64]*stock.Warehouse),
		pickings:       make(map[int64]*stock.StockPicking),
		moves:          make(map[int64]*stock.StockMove),
		moveLines:      make(map[int64]*stock.StockMoveLine),
		lots:           make(map[int64]*stock.StockLot),
		quants:         make(map[string]*stock.StockQuant),
		productValues:  make(map[int64]*stock.ProductValue),
		periods:        make(map[int64]*stock.AccountingPeriod),
		orderpoints:    make(map[int64]*stock.Orderpoint),
		landedCosts:    make(map[int64]*stock.LandedCost),
		procurements:   make(map[int64]*stock.ProcurementGroup),
		routes:         make(map[int64]*stock.StockRoute),
		rules:          make(map[int64]*stock.StockRule),
		scraps:         make(map[int64]*stock.StockScrap),
		seqInCounters:  make(map[int]int64),
		seqOutCounters: make(map[int]int64),
		seqIntCounters: make(map[int]int64),
		opCounters:     make(map[int]int64),
		lcCounters:     make(map[int]int64),
	}

	now := time.Now().UTC()

	// Seed Standard Locations
	p1 := int64(1)
	p4 := int64(4)
	p7 := int64(7)

	seeds := []stock.StockLocation{
		{ID: 1, Name: "Partner Locations", CompleteName: "Partner Locations", Usage: stock.LocationUsageView, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Vendors", CompleteName: "Partner Locations/Vendors", Usage: stock.LocationUsageSupplier, ParentID: &p1, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "Customers", CompleteName: "Partner Locations/Customers", Usage: stock.LocationUsageCustomer, ParentID: &p1, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "Virtual Locations", CompleteName: "Virtual Locations", Usage: stock.LocationUsageView, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 5, Name: "Inventory adjustment", CompleteName: "Virtual Locations/Inventory adjustment", Usage: stock.LocationUsageInventory, ParentID: &p4, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 6, Name: "Scrap", CompleteName: "Virtual Locations/Scrap", Usage: stock.LocationUsageInventory, ScrapLocation: true, ParentID: &p4, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 7, Name: "WH", CompleteName: "WH", Usage: stock.LocationUsageView, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 8, Name: "Stock", CompleteName: "WH/Stock", Usage: stock.LocationUsageInternal, ParentID: &p7, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 9, Name: "Input", CompleteName: "WH/Input", Usage: stock.LocationUsageInternal, ParentID: &p7, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 10, Name: "Output", CompleteName: "WH/Output", Usage: stock.LocationUsageInternal, ParentID: &p7, Active: true, CreatedAt: now, UpdatedAt: now},
	}

	for _, loc := range seeds {
		clone := loc
		repo.locations[loc.ID] = &clone
	}
	repo.lastLocID = 10

	// Seed Main Warehouse
	mainWh := stock.Warehouse{
		ID:             1,
		Name:           "Main Warehouse",
		Code:           "WH",
		ViewLocationID: &p7,
		LotStockID:     8,
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repo.warehouses[1] = &mainWh
	repo.lastWhID = 1

	return repo
}

// ─────────────────────────────────────────────────────────────────────────────
// Locations
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateLocation(ctx context.Context, loc *stock.StockLocation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastLocID++
	loc.ID = r.lastLocID
	now := time.Now().UTC()
	loc.CreatedAt = now
	loc.UpdatedAt = now
	loc.Active = true

	clone := *loc
	r.locations[loc.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetLocationByID(ctx context.Context, id int64) (*stock.StockLocation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	loc, ok := r.locations[id]
	if !ok || !loc.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("stock location #%d not found", id))
	}
	clone := *loc
	return &clone, nil
}

func (r *MemoryRepo) GetLocationByName(ctx context.Context, name string) (*stock.StockLocation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, loc := range r.locations {
		if loc.Active && strings.EqualFold(string(loc.Name), name) {
			clone := *loc
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound(fmt.Sprintf("stock location '%s' not found", name))
}

func (r *MemoryRepo) GetLocationByUsage(ctx context.Context, usage stock.LocationUsage) (*stock.StockLocation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, loc := range r.locations {
		if loc.Active && loc.Usage == usage {
			clone := *loc
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound(fmt.Sprintf("stock location for usage '%s' not found", usage))
}

func (r *MemoryRepo) UpdateLocation(ctx context.Context, loc *stock.StockLocation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.locations[loc.ID]
	if !ok || !existing.Active {
		return platformerrors.NotFound(fmt.Sprintf("stock location #%d not found", loc.ID))
	}

	loc.UpdatedAt = time.Now().UTC()
	clone := *loc
	r.locations[loc.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteLocation(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	loc, ok := r.locations[id]
	if !ok || !loc.Active {
		return platformerrors.NotFound(fmt.Sprintf("stock location #%d not found", id))
	}

	// Soft delete
	loc.Active = false
	loc.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListLocations(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.StockLocation], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []stock.StockLocation
	for _, loc := range r.locations {
		if !loc.Active {
			continue
		}
		if f != nil && len(f.Criteria) > 0 {
			match := true
			for _, crit := range f.Criteria {
				switch crit.Field {
				case "usage":
					if string(loc.Usage) != fmt.Sprintf("%v", crit.Value) {
						match = false
					}
				case "name", "search":
					s := strings.ToLower(fmt.Sprintf("%v", crit.Value))
					if !strings.Contains(strings.ToLower(string(loc.Name)), s) && !strings.Contains(strings.ToLower(string(loc.CompleteName)), s) {
						match = false
					}
				}
			}
			if !match {
				continue
			}
		}
		result = append(result, *loc)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	total := int64(len(result))
	offset := page.Offset()
	limit := page.LimitClamped()

	if offset >= int(total) {
		return pagination.NewPageResult([]stock.StockLocation{}, total, page), nil
	}

	end := offset + limit
	if end > int(total) {
		end = int(total)
	}

	return pagination.NewPageResult(result[offset:end], total, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Warehouses
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateWarehouse(ctx context.Context, wh *stock.Warehouse) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, w := range r.warehouses {
		if w.Active && strings.EqualFold(w.Code, wh.Code) {
			return platformerrors.Conflict(fmt.Sprintf("warehouse with code '%s' already exists", wh.Code))
		}
	}

	r.lastWhID++
	wh.ID = r.lastWhID
	now := time.Now().UTC()
	wh.CreatedAt = now
	wh.UpdatedAt = now
	wh.Active = true

	clone := *wh
	r.warehouses[wh.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetWarehouseByID(ctx context.Context, id int64) (*stock.Warehouse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wh, ok := r.warehouses[id]
	if !ok || !wh.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("warehouse #%d not found", id))
	}
	clone := *wh
	return &clone, nil
}

func (r *MemoryRepo) GetWarehouseByCode(ctx context.Context, code string) (*stock.Warehouse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, wh := range r.warehouses {
		if wh.Active && strings.EqualFold(wh.Code, code) {
			clone := *wh
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound(fmt.Sprintf("warehouse with code '%s' not found", code))
}

func (r *MemoryRepo) UpdateWarehouse(ctx context.Context, wh *stock.Warehouse) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.warehouses[wh.ID]
	if !ok || !existing.Active {
		return platformerrors.NotFound(fmt.Sprintf("warehouse #%d not found", wh.ID))
	}

	for _, w := range r.warehouses {
		if w.ID != wh.ID && w.Active && strings.EqualFold(w.Code, wh.Code) {
			return platformerrors.Conflict(fmt.Sprintf("warehouse with code '%s' already exists", wh.Code))
		}
	}

	wh.UpdatedAt = time.Now().UTC()
	clone := *wh
	r.warehouses[wh.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteWarehouse(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	wh, ok := r.warehouses[id]
	if !ok || !wh.Active {
		return platformerrors.NotFound(fmt.Sprintf("warehouse #%d not found", id))
	}

	wh.Active = false
	wh.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListWarehouses(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.Warehouse], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []stock.Warehouse
	for _, wh := range r.warehouses {
		if !wh.Active {
			continue
		}
		if f != nil && len(f.Criteria) > 0 {
			match := true
			for _, crit := range f.Criteria {
				switch crit.Field {
				case "code":
					if !strings.EqualFold(wh.Code, fmt.Sprintf("%v", crit.Value)) {
						match = false
					}
				case "name", "search":
					s := strings.ToLower(fmt.Sprintf("%v", crit.Value))
					if !strings.Contains(strings.ToLower(string(wh.Name)), s) && !strings.Contains(strings.ToLower(wh.Code), s) {
						match = false
					}
				}
			}
			if !match {
				continue
			}
		}
		result = append(result, *wh)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	total := int64(len(result))
	offset := page.Offset()
	limit := page.LimitClamped()

	if offset >= int(total) {
		return pagination.NewPageResult([]stock.Warehouse{}, total, page), nil
	}

	end := offset + limit
	if end > int(total) {
		end = int(total)
	}

	return pagination.NewPageResult(result[offset:end], total, page), nil
}

func (r *MemoryRepo) GetDefaultWarehouse(ctx context.Context) (*stock.Warehouse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if wh, ok := r.warehouses[1]; ok && wh.Active {
		clone := *wh
		return &clone, nil
	}

	for _, wh := range r.warehouses {
		if wh.Active {
			clone := *wh
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound("no default warehouse found")
}

// ─────────────────────────────────────────────────────────────────────────────
// Sequences & Pickings
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) NextSequence(ctx context.Context, pickingType stock.PickingType, year int) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch pickingType {
	case stock.PickingTypeIncoming:
		r.seqInCounters[year]++
		return fmt.Sprintf("WH/IN/%d/%05d", year, r.seqInCounters[year]), nil
	case stock.PickingTypeOutgoing:
		r.seqOutCounters[year]++
		return fmt.Sprintf("WH/OUT/%d/%05d", year, r.seqOutCounters[year]), nil
	case stock.PickingTypeInternal:
		r.seqIntCounters[year]++
		return fmt.Sprintf("WH/INT/%d/%05d", year, r.seqIntCounters[year]), nil
	default:
		return "", platformerrors.Validation("unknown picking type for sequence", nil)
	}
}

func (r *MemoryRepo) CreatePicking(ctx context.Context, picking *stock.StockPicking) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if picking.Name != "" && picking.Name != "/" {
		for _, p := range r.pickings {
			if p.Active && strings.EqualFold(p.Name, picking.Name) {
				return platformerrors.Conflict(fmt.Sprintf("picking with name '%s' already exists", picking.Name))
			}
		}
	}

	r.lastPickingID++
	picking.ID = r.lastPickingID
	now := time.Now().UTC()
	picking.CreatedAt = now
	picking.UpdatedAt = now
	picking.Active = true

	storedMoves := make([]stock.StockMove, len(picking.Moves))
	for i, m := range picking.Moves {
		r.lastMoveID++
		mClone := m
		mClone.ID = r.lastMoveID
		mClone.PickingID = &picking.ID
		mClone.CreatedAt = now
		mClone.UpdatedAt = now
		if mClone.Date.IsZero() {
			mClone.Date = now
		}
		r.moves[mClone.ID] = &mClone
		storedMoves[i] = mClone
	}
	picking.Moves = storedMoves

	clone := *picking
	r.pickings[picking.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetPickingByID(ctx context.Context, id int64) (*stock.StockPicking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	picking, ok := r.pickings[id]
	if !ok || !picking.Active {
		return nil, platformerrors.NotFound(fmt.Sprintf("stock picking #%d not found", id))
	}

	clone := *picking
	var moves []stock.StockMove
	for _, m := range r.moves {
		if m.PickingID != nil && *m.PickingID == id {
			moves = append(moves, *m)
		}
	}
	sort.Slice(moves, func(i, j int) bool {
		return moves[i].Sequence < moves[j].Sequence
	})
	clone.Moves = moves
	return &clone, nil
}

func (r *MemoryRepo) GetPickingByName(ctx context.Context, name string) (*stock.StockPicking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.pickings {
		if p.Active && strings.EqualFold(p.Name, name) {
			clone := *p
			var moves []stock.StockMove
			for _, m := range r.moves {
				if m.PickingID != nil && *m.PickingID == p.ID {
					moves = append(moves, *m)
				}
			}
			sort.Slice(moves, func(i, j int) bool {
				return moves[i].Sequence < moves[j].Sequence
			})
			clone.Moves = moves
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound(fmt.Sprintf("stock picking '%s' not found", name))
}

func (r *MemoryRepo) UpdatePicking(ctx context.Context, picking *stock.StockPicking) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.pickings[picking.ID]
	if !ok || !existing.Active {
		return platformerrors.NotFound(fmt.Sprintf("stock picking #%d not found", picking.ID))
	}

	now := time.Now().UTC()
	picking.UpdatedAt = now

	// Replace moves
	for mID, m := range r.moves {
		if m.PickingID != nil && *m.PickingID == picking.ID {
			delete(r.moves, mID)
		}
	}

	storedMoves := make([]stock.StockMove, len(picking.Moves))
	for i, m := range picking.Moves {
		if m.ID <= 0 {
			r.lastMoveID++
			m.ID = r.lastMoveID
		}
		mClone := m
		mClone.PickingID = &picking.ID
		mClone.UpdatedAt = now
		if mClone.CreatedAt.IsZero() {
			mClone.CreatedAt = now
		}
		if mClone.Date.IsZero() {
			mClone.Date = now
		}
		r.moves[mClone.ID] = &mClone
		storedMoves[i] = mClone
	}
	picking.Moves = storedMoves

	clone := *picking
	r.pickings[picking.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeletePicking(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.pickings[id]
	if !ok || !p.Active {
		return platformerrors.NotFound(fmt.Sprintf("stock picking #%d not found", id))
	}

	p.Active = false
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListPickings(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.StockPicking], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []stock.StockPicking
	for _, p := range r.pickings {
		if !p.Active {
			continue
		}
		if f != nil && len(f.Criteria) > 0 {
			match := true
			for _, crit := range f.Criteria {
				switch crit.Field {
				case "picking_type":
					if string(p.PickingType) != fmt.Sprintf("%v", crit.Value) {
						match = false
					}
				case "state":
					if string(p.State) != fmt.Sprintf("%v", crit.Value) {
						match = false
					}
				case "partner_id":
					var pid int64
					_, _ = fmt.Sscanf(fmt.Sprintf("%v", crit.Value), "%d", &pid)
					if p.PartnerID == nil || *p.PartnerID != pid {
						match = false
					}
				case "origin":
					if !strings.Contains(strings.ToLower(p.Origin), strings.ToLower(fmt.Sprintf("%v", crit.Value))) {
						match = false
					}
				case "name", "search":
					s := strings.ToLower(fmt.Sprintf("%v", crit.Value))
					if !strings.Contains(strings.ToLower(p.Name), s) && !strings.Contains(strings.ToLower(p.Origin), s) {
						match = false
					}
				}
			}
			if !match {
				continue
			}
		}

		clone := *p
		var moves []stock.StockMove
		for _, m := range r.moves {
			if m.PickingID != nil && *m.PickingID == p.ID {
				moves = append(moves, *m)
			}
		}
		clone.Moves = moves
		result = append(result, clone)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID > result[j].ID
	})

	total := int64(len(result))
	offset := page.Offset()
	limit := page.LimitClamped()

	if offset >= int(total) {
		return pagination.NewPageResult([]stock.StockPicking{}, total, page), nil
	}

	end := offset + limit
	if end > int(total) {
		end = int(total)
	}

	return pagination.NewPageResult(result[offset:end], total, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Moves
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateMove(ctx context.Context, move *stock.StockMove) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastMoveID++
	move.ID = r.lastMoveID
	now := time.Now().UTC()
	move.CreatedAt = now
	move.UpdatedAt = now
	if move.Date.IsZero() {
		move.Date = now
	}

	clone := *move
	r.moves[move.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetMoveByID(ctx context.Context, id int64) (*stock.StockMove, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.moves[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("stock move #%d not found", id))
	}
	clone := *m
	for _, line := range r.moveLines {
		if line.MoveID == id {
			clone.MoveLines = append(clone.MoveLines, *line)
		}
	}
	sort.Slice(clone.MoveLines, func(i, j int) bool { return clone.MoveLines[i].ID < clone.MoveLines[j].ID })
	return &clone, nil
}

func (r *MemoryRepo) UpdateMove(ctx context.Context, move *stock.StockMove) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.moves[move.ID]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("stock move #%d not found", move.ID))
	}

	move.UpdatedAt = time.Now().UTC()
	clone := *move
	r.moves[move.ID] = &clone
	_ = existing
	return nil
}

func (r *MemoryRepo) ReserveMove(ctx context.Context, move *stock.StockMove) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.moves[move.ID]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("stock move #%d not found", move.ID))
	}
	remaining := move.ProductQty - move.ReservedQuantity
	if remaining <= 0 {
		return nil
	}
	quant, ok := r.quants[quantKey(move.ProductID, move.LocationID)]
	if !ok || quant.Quantity-quant.ReservedQuantity <= 0 {
		return nil
	}
	available := quant.Quantity - quant.ReservedQuantity
	reserved := remaining
	if reserved > available {
		reserved = available
	}
	quant.ReservedQuantity += reserved
	quant.UpdatedAt = time.Now().UTC()

	move.ReservedQuantity += reserved
	if move.ReservedQuantity >= move.ProductQty {
		move.State = stock.MoveStateAssigned
	}
	move.UpdatedAt = time.Now().UTC()
	clone := *move
	r.moves[move.ID] = &clone

	r.lastMoveLineID++
	line := &stock.StockMoveLine{
		ID:               r.lastMoveLineID,
		MoveID:           move.ID,
		ProductID:        move.ProductID,
		ProductUom:       move.ProductUom,
		LocationID:       move.LocationID,
		LocationDestID:   move.LocationDestID,
		ReservedQuantity: reserved,
		Date:             move.Date,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	r.moveLines[line.ID] = line
	move.MoveLines = append(move.MoveLines, *line)
	_ = existing
	return nil
}

func (r *MemoryRepo) CreateMoveLine(ctx context.Context, line *stock.StockMoveLine) error {
	if err := line.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.moves[line.MoveID]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("stock move #%d not found", line.MoveID))
	}
	if line.LotID != nil {
		lot, ok := r.lots[*line.LotID]
		if !ok || !lot.Active {
			return platformerrors.NotFound(fmt.Sprintf("stock lot #%d not found", *line.LotID))
		}
		if lot.ProductID != line.ProductID {
			return platformerrors.Validation("lot product does not match move line product", nil)
		}
		if lot.TrackingMode == stock.TrackingSerial && (line.ReservedQuantity > 1 || line.QuantityDone > 1) {
			return platformerrors.Validation("serial move line quantity cannot exceed one", nil)
		}
	}
	r.lastMoveLineID++
	line.ID = r.lastMoveLineID
	now := time.Now().UTC()
	line.CreatedAt = now
	line.UpdatedAt = now
	if line.Date.IsZero() {
		line.Date = now
	}
	clone := *line
	r.moveLines[line.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetMoveLineByID(ctx context.Context, id int64) (*stock.StockMoveLine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	line, ok := r.moveLines[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("stock move line #%d not found", id))
	}
	clone := *line
	return &clone, nil
}

func (r *MemoryRepo) ListMoveLinesByMoveID(ctx context.Context, moveID int64) ([]stock.StockMoveLine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var lines []stock.StockMoveLine
	for _, line := range r.moveLines {
		if line.MoveID == moveID {
			lines = append(lines, *line)
		}
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].ID < lines[j].ID })
	return lines, nil
}

func (r *MemoryRepo) UpdateMoveLine(ctx context.Context, line *stock.StockMoveLine) error {
	if err := line.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.moveLines[line.ID]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("stock move line #%d not found", line.ID))
	}
	line.UpdatedAt = time.Now().UTC()
	clone := *line
	r.moveLines[line.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteMoveLine(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.moveLines[id]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("stock move line #%d not found", id))
	}
	delete(r.moveLines, id)
	return nil
}

func (r *MemoryRepo) CreateLot(ctx context.Context, lot *stock.StockLot) error {
	if err := lot.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.lots {
		if existing.ProductID == lot.ProductID && existing.Name == lot.Name {
			return platformerrors.Conflict("lot name already exists for product", nil)
		}
	}
	r.lastLotID++
	lot.ID = r.lastLotID
	now := time.Now().UTC()
	lot.CreatedAt = now
	lot.UpdatedAt = now
	lot.Active = true
	clone := *lot
	r.lots[lot.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetLotByID(ctx context.Context, id int64) (*stock.StockLot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lot, ok := r.lots[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("stock lot #%d not found", id))
	}
	clone := *lot
	return &clone, nil
}

func (r *MemoryRepo) ListLotsByProduct(ctx context.Context, productID int64) ([]stock.StockLot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var lots []stock.StockLot
	for _, lot := range r.lots {
		if lot.ProductID == productID && lot.Active {
			lots = append(lots, *lot)
		}
	}
	sort.Slice(lots, func(i, j int) bool { return lots[i].ID < lots[j].ID })
	return lots, nil
}

func (r *MemoryRepo) UpdateLot(ctx context.Context, lot *stock.StockLot) error {
	if err := lot.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.lots[lot.ID]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("stock lot #%d not found", lot.ID))
	}
	lot.UpdatedAt = time.Now().UTC()
	clone := *lot
	r.lots[lot.ID] = &clone
	return nil
}

func (r *MemoryRepo) ListMoves(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.StockMove], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []stock.StockMove
	for _, m := range r.moves {
		if f != nil && len(f.Criteria) > 0 {
			match := true
			for _, crit := range f.Criteria {
				switch crit.Field {
				case "product_id":
					var pid int64
					_, _ = fmt.Sscanf(fmt.Sprintf("%v", crit.Value), "%d", &pid)
					if m.ProductID != pid {
						match = false
					}
				case "picking_id":
					var pickID int64
					_, _ = fmt.Sscanf(fmt.Sprintf("%v", crit.Value), "%d", &pickID)
					if m.PickingID == nil || *m.PickingID != pickID {
						match = false
					}
				case "state":
					if string(m.State) != fmt.Sprintf("%v", crit.Value) {
						match = false
					}
				}
			}
			if !match {
				continue
			}
		}
		result = append(result, *m)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID > result[j].ID
	})

	total := int64(len(result))
	offset := page.Offset()
	limit := page.LimitClamped()

	if offset >= int(total) {
		return pagination.NewPageResult([]stock.StockMove{}, total, page), nil
	}

	end := offset + limit
	if end > int(total) {
		end = int(total)
	}

	return pagination.NewPageResult(result[offset:end], total, page), nil
}

func (r *MemoryRepo) GetMovesByPickingID(ctx context.Context, pickingID int64) ([]stock.StockMove, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var moves []stock.StockMove
	for _, m := range r.moves {
		if m.PickingID != nil && *m.PickingID == pickingID {
			moves = append(moves, *m)
		}
	}
	sort.Slice(moves, func(i, j int) bool {
		return moves[i].Sequence < moves[j].Sequence
	})
	return moves, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Quants & Balances
// ─────────────────────────────────────────────────────────────────────────────

func quantKey(productID, locationID int64) string {
	return fmt.Sprintf("%d:%d", productID, locationID)
}

func (r *MemoryRepo) GetQuant(ctx context.Context, productID, locationID int64) (*stock.StockQuant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q, ok := r.quants[quantKey(productID, locationID)]
	if !ok {
		return &stock.StockQuant{
			ProductID:  productID,
			LocationID: locationID,
			Quantity:   0,
		}, nil
	}
	clone := *q
	return &clone, nil
}

func (r *MemoryRepo) ReserveQuantity(ctx context.Context, productID, locationID int64, quantity float64) (float64, error) {
	if quantity <= 0 {
		return 0, platformerrors.Validation("reservation quantity must be positive", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	q, ok := r.quants[quantKey(productID, locationID)]
	if !ok {
		return 0, nil
	}
	available := q.Quantity - q.ReservedQuantity
	if available <= 0 {
		return 0, nil
	}
	reserved := quantity
	if reserved > available {
		reserved = available
	}
	q.ReservedQuantity += reserved
	q.UpdatedAt = time.Now().UTC()
	return reserved, nil
}

func (r *MemoryRepo) UpdateQuantQuantity(ctx context.Context, productID, locationID int64, deltaQty float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	k := quantKey(productID, locationID)
	now := time.Now().UTC()
	q, ok := r.quants[k]
	if !ok {
		r.lastQuantID++
		q = &stock.StockQuant{
			ID:         r.lastQuantID,
			ProductID:  productID,
			LocationID: locationID,
			Quantity:   deltaQty,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		r.quants[k] = q
		return nil
	}

	q.Quantity += deltaQty
	q.UpdatedAt = now
	return nil
}

func (r *MemoryRepo) SetQuantQuantity(ctx context.Context, productID, locationID int64, newQty float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	k := quantKey(productID, locationID)
	now := time.Now().UTC()
	q, ok := r.quants[k]
	if !ok {
		r.lastQuantID++
		q = &stock.StockQuant{
			ID:         r.lastQuantID,
			ProductID:  productID,
			LocationID: locationID,
			Quantity:   newQty,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		r.quants[k] = q
		return nil
	}

	q.Quantity = newQty
	q.UpdatedAt = now
	return nil
}

func (r *MemoryRepo) ListQuants(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.StockQuant], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []stock.StockQuant
	for _, q := range r.quants {
		if f != nil && len(f.Criteria) > 0 {
			match := true
			for _, crit := range f.Criteria {
				switch crit.Field {
				case "product_id":
					var pid int64
					_, _ = fmt.Sscanf(fmt.Sprintf("%v", crit.Value), "%d", &pid)
					if q.ProductID != pid {
						match = false
					}
				case "location_id":
					var locID int64
					_, _ = fmt.Sscanf(fmt.Sprintf("%v", crit.Value), "%d", &locID)
					if q.LocationID != locID {
						match = false
					}
				}
			}
			if !match {
				continue
			}
		}
		result = append(result, *q)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ProductID < result[j].ProductID
	})

	total := int64(len(result))
	offset := page.Offset()
	limit := page.LimitClamped()

	if offset >= int(total) {
		return pagination.NewPageResult([]stock.StockQuant{}, total, page), nil
	}

	end := offset + limit
	if end > int(total) {
		end = int(total)
	}

	return pagination.NewPageResult(result[offset:end], total, page), nil
}

func (r *MemoryRepo) GetOnHandStock(ctx context.Context, productID *int64, locationID *int64, warehouseID *int64) ([]stock.StockOnHandItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var items []stock.StockOnHandItem

	// Filter quants
	for _, q := range r.quants {
		if productID != nil && q.ProductID != *productID {
			continue
		}
		if locationID != nil && q.LocationID != *locationID {
			continue
		}

		loc, ok := r.locations[q.LocationID]
		if !ok || !loc.Active {
			continue
		}

		// If no specific location requested, only include internal storage locations
		if locationID == nil && loc.Usage != stock.LocationUsageInternal {
			continue
		}

		var whID *int64
		var whName string
		for _, w := range r.warehouses {
			if w.Active && w.LotStockID == loc.ID {
				whID = &w.ID
				whName = string(w.Name)
				break
			}
		}

		if warehouseID != nil && (whID == nil || *whID != *warehouseID) {
			continue
		}

		item := stock.StockOnHandItem{
			ProductID:         q.ProductID,
			LocationID:        q.LocationID,
			LocationName:      string(loc.CompleteName),
			WarehouseID:       whID,
			WarehouseName:     whName,
			Quantity:          q.Quantity,
			ReservedQuantity:  q.ReservedQuantity,
			AvailableQuantity: q.AvailableQuantity(),
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].ProductID == items[j].ProductID {
			return items[i].LocationID < items[j].LocationID
		}
		return items[i].ProductID < items[j].ProductID
	})

	return items, nil
}

// ValidateMovesTx performs atomic move validation and double-entry quant movement in memory.
func (r *MemoryRepo) ValidateMovesTx(ctx context.Context, moves []stock.StockMove) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	for i := range moves {
		m := &moves[i]
		qty := m.QuantityDone
		if qty <= 0 {
			qty = m.ProductQty
		}
		if qty <= 0 || qty > m.ProductQty {
			return platformerrors.Validation("move quantity is outside the requested quantity", nil)
		}
		reservedToConsume := m.ReservedQuantity
		if reservedToConsume > qty {
			reservedToConsume = qty
		}
		m.ReservedQuantity -= reservedToConsume
		m.QuantityDone = qty
		m.State = stock.MoveStateDone
		m.UpdatedAt = now

		// Update move in repository map
		r.moves[m.ID] = m

		// Decrease Source Location Quant
		srcKey := quantKey(m.ProductID, m.LocationID)
		if q, ok := r.quants[srcKey]; ok {
			q.Quantity -= qty
			q.ReservedQuantity -= reservedToConsume
			if q.ReservedQuantity < 0 {
				q.ReservedQuantity = 0
			}
			q.UpdatedAt = now
		} else {
			r.lastQuantID++
			r.quants[srcKey] = &stock.StockQuant{
				ID:         r.lastQuantID,
				ProductID:  m.ProductID,
				LocationID: m.LocationID,
				Quantity:   -qty,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
		}

		remainingDone := qty
		for _, line := range r.moveLines {
			if line.MoveID != m.ID || remainingDone <= 0 {
				continue
			}
			lineDone := line.ReservedQuantity
			if lineDone > remainingDone {
				lineDone = remainingDone
			}
			line.ReservedQuantity -= lineDone
			line.QuantityDone += lineDone
			line.UpdatedAt = now
			remainingDone -= lineDone
		}

		// Increase Destination Location Quant
		destKey := quantKey(m.ProductID, m.LocationDestID)
		if q, ok := r.quants[destKey]; ok {
			q.Quantity += qty
			q.UpdatedAt = now
		} else {
			r.lastQuantID++
			r.quants[destKey] = &stock.StockQuant{
				ID:         r.lastQuantID,
				ProductID:  m.ProductID,
				LocationID: m.LocationDestID,
				Quantity:   qty,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
		}
	}
	return nil
}

// ValidatePickingTx performs atomic picking validation and double-entry quant movement in memory.
func (r *MemoryRepo) ValidatePickingTx(ctx context.Context, picking *stock.StockPicking) error {
	if err := r.ValidateMovesTx(ctx, picking.Moves); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	picking.State = stock.PickingStateDone
	picking.DateDone = &now
	picking.UpdatedAt = now

	clone := *picking
	r.pickings[picking.ID] = &clone
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Valuations (Phase 12 — stock-account integration)
// ─────────────────────────────────────────────────────────────────────────────

// UpdateMoveValue persists the valuation fields of a stock move.
func (r *MemoryRepo) UpdateMoveValue(ctx context.Context, move *stock.StockMove) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.moves[move.ID]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("stock move #%d not found", move.ID))
	}
	clone := *move
	clone.UpdatedAt = time.Now().UTC()
	r.moves[move.ID] = &clone
	*move = clone
	return nil
}

// GetFIFOStack returns the remaining incoming layers for a product, ordered oldest first.
func (r *MemoryRepo) GetFIFOStack(ctx context.Context, productID, companyID int64) (stock.FIFOStack, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stack, err := r.fifoStackLocked(productID)
	if err != nil {
		return nil, err
	}
	return stack, nil
}

// CreateProductValue inserts a history record of a product value update.
func (r *MemoryRepo) CreateProductValue(ctx context.Context, pv *stock.ProductValue) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastValueID++
	pv.ID = r.lastValueID
	pv.CreatedAt = time.Now().UTC()
	if pv.Date.IsZero() {
		pv.Date = pv.CreatedAt
	}
	clone := *pv
	r.productValues[pv.ID] = &clone
	return nil
}

// ListProductValues returns the history of product value records.
func (r *MemoryRepo) ListProductValues(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.ProductValue], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var values []stock.ProductValue
	for _, pv := range r.productValues {
		if !matchFilterByString(pv.ProductID, "product_id", f) || !matchFilterByInt64Ptr(pv.MoveID, "move_id", f) {
			continue
		}
		values = append(values, *pv)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID > values[j].ID })
	total := int64(len(values))
	start := int64(page.Offset())
	end := start + int64(page.LimitClamped())
	if start >= total {
		start, end = 0, 0
	}
	if end > total {
		end = total
	}
	return pagination.NewPageResult(values[start:end], total, page), nil
}

// ComputeTotalValuation aggregates the current valuation by product (and optional location).
func (r *MemoryRepo) ComputeTotalValuation(ctx context.Context, productID *int64, locationID *int64) ([]stock.ValuationSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	type aggKey struct {
		product int64
		loc     int64
	}
	agg := make(map[aggKey]*stock.ValuationSummary)

	for _, q := range r.quants {
		if productID != nil && q.ProductID != *productID {
			continue
		}
		if locationID != nil && q.LocationID != *locationID {
			continue
		}
		loc, ok := r.locations[q.LocationID]
		if !ok || !loc.Active {
			continue
		}
		if locationID == nil && loc.Usage != stock.LocationUsageInternal {
			continue
		}
		if q.Quantity == 0 {
			continue
		}

		stack, err := r.fifoStackLocked(q.ProductID)
		if err != nil {
			return nil, err
		}
		unitCost := stack.UnitPrice()

		key := aggKey{product: q.ProductID, loc: q.LocationID}
		if s, ok := agg[key]; ok {
			s.Quantity += q.Quantity
			s.Value += q.Quantity * unitCost
			s.UnitCost = unitCost
		} else {
			locID := q.LocationID
			s := &stock.ValuationSummary{
				ProductID:   q.ProductID,
				ProductName: r.productNameLocked(q.ProductID),
				LocationID:  &locID,
				Location:    string(loc.CompleteName),
				Quantity:    q.Quantity,
				UnitCost:    unitCost,
				Value:       q.Quantity * unitCost,
			}
			agg[key] = s
		}
	}

	var summaries []stock.ValuationSummary
	for _, s := range agg {
		summaries = append(summaries, *s)
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].ProductID == summaries[j].ProductID {
			return summaries[i].Location < summaries[j].Location
		}
		return summaries[i].ProductID < summaries[j].ProductID
	})
	return summaries, nil
}

// CreateAccountingPeriod inserts a new period.
func (r *MemoryRepo) CreateAccountingPeriod(ctx context.Context, p *stock.AccountingPeriod) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastPeriodID++
	p.ID = r.lastPeriodID
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.State == "" {
		p.State = "open"
	}
	clone := *p
	r.periods[p.ID] = &clone
	return nil
}

// GetAccountingPeriodByID fetches a single period.
func (r *MemoryRepo) GetAccountingPeriodByID(ctx context.Context, id int64) (*stock.AccountingPeriod, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.periods[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("accounting period #%d not found", id))
	}
	clone := *p
	return &clone, nil
}

// ListAccountingPeriods returns a paginated list of periods.
func (r *MemoryRepo) ListAccountingPeriods(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.AccountingPeriod], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var periods []stock.AccountingPeriod
	for _, p := range r.periods {
		if !matchPeriodFilter(p, f) {
			continue
		}
		periods = append(periods, *p)
	}
	sort.Slice(periods, func(i, j int) bool { return periods[i].DateFrom.After(periods[j].DateFrom) })
	total := int64(len(periods))
	start := int64(page.Offset())
	end := start + int64(page.LimitClamped())
	if start >= total {
		start, end = 0, 0
	}
	if end > total {
		end = total
	}
	return pagination.NewPageResult(periods[start:end], total, page), nil
}

// CloseAccountingPeriod marks a period as closed and links the closing entry.
func (r *MemoryRepo) CloseAccountingPeriod(ctx context.Context, p *stock.AccountingPeriod) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.periods[p.ID]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("accounting period #%d not found", p.ID))
	}
	if existing.State == "closed" {
		return platformerrors.Conflict("accounting period is already closed")
	}
	existing.State = "closed"
	existing.AccountMoveID = p.AccountMoveID
	existing.UpdatedAt = time.Now().UTC()
	p.State = existing.State
	p.AccountMoveID = existing.AccountMoveID
	p.UpdatedAt = existing.UpdatedAt
	return nil
}

// fifoStackLocked returns the FIFO stack without acquiring the lock (caller must hold r.mu).
func (r *MemoryRepo) fifoStackLocked(productID int64) (stock.FIFOStack, error) {
	var stack stock.FIFOStack
	for _, m := range r.moves {
		if m.ProductID != productID || !m.IsIn || m.State != stock.MoveStateDone || m.RemainingQty <= 0 {
			continue
		}
		stack = append(stack, stock.FIFOEntry{MoveID: m.ID, Qty: m.RemainingQty, Value: m.RemainingValue})
	}
	sort.Slice(stack, func(i, j int) bool { return stack[i].MoveID < stack[j].MoveID })
	return stack, nil
}

// productNameLocked resolves a product template name (requires r.mu).
func (r *MemoryRepo) productNameLocked(productID int64) string {
	return fmt.Sprintf("Product #%d", productID)
}

// matchFilterByString reports whether an int64 field matches the filter (empty filter → true).
func matchFilterByString(v int64, field string, f *filter.Filter) bool {
	for _, c := range filterCriteria(f, field) {
		want, ok := criteriaInt64(c)
		if !ok {
			continue
		}
		return v == want
	}
	return true
}

// matchFilterByInt64Ptr reports whether an *int64 field matches the filter (empty filter → true).
func matchFilterByInt64Ptr(v *int64, field string, f *filter.Filter) bool {
	for _, c := range filterCriteria(f, field) {
		want, ok := criteriaInt64(c)
		if !ok {
			continue
		}
		return v != nil && *v == want
	}
	return true
}

// matchPeriodFilter reports whether a period matches the filter (empty filter → true).
func matchPeriodFilter(p *stock.AccountingPeriod, f *filter.Filter) bool {
	for _, c := range filterCriteria(f, "state") {
		if state, ok := c.Value.(string); ok && state != p.State {
			return false
		}
	}
	for _, c := range filterCriteria(f, "journal_id") {
		want, ok := criteriaInt64(c)
		if !ok {
			continue
		}
		return want == p.JournalID
	}
	return true
}

// filterCriteria returns all filter criteria for the given field.
func filterCriteria(f *filter.Filter, field string) []filter.Criterion {
	if f == nil {
		return nil
	}
	var out []filter.Criterion
	for _, c := range f.Criteria {
		if c.Field == field {
			out = append(out, c)
		}
	}
	return out
}

// criteriaInt64 coerces a criteria value to int64.
func criteriaInt64(c filter.Criterion) (int64, bool) {
	var want int64
	if _, err := fmt.Sscanf(fmt.Sprintf("%v", c.Value), "%d", &want); err != nil {
		return 0, false
	}
	return want, true
}

// ─────────────────────────────────────────────────────────────────────────────
// Reorder Rules (Phase 13 — stock.orderpoint)
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateOrderpoint(ctx context.Context, op *stock.Orderpoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.orderpoints {
		if existing.Active && existing.ProductID == op.ProductID && existing.LocationID == op.LocationID && existing.CompanyID == op.CompanyID {
			return platformerrors.Conflict("an orderpoint already exists for this product in this location")
		}
	}

	r.lastOpID++
	op.ID = r.lastOpID
	if op.Name == "" {
		year := time.Now().UTC().Year()
		r.opCounters[year]++
		op.Name = fmt.Sprintf("ROP/%d/%05d", year, r.opCounters[year])
	}
	now := time.Now().UTC()
	op.CreatedAt = now
	op.UpdatedAt = now
	op.Active = true
	clone := *op
	r.orderpoints[op.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetOrderpointByID(ctx context.Context, id int64) (*stock.Orderpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	op, ok := r.orderpoints[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("stock orderpoint #%d not found", id))
	}
	clone := *op
	return &clone, nil
}

func (r *MemoryRepo) UpdateOrderpoint(ctx context.Context, op *stock.Orderpoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.orderpoints[op.ID]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("stock orderpoint #%d not found", op.ID))
	}
	op.UpdatedAt = time.Now().UTC()
	clone := *op
	r.orderpoints[op.ID] = &clone
	_ = existing
	return nil
}

func (r *MemoryRepo) DeleteOrderpoint(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	op, ok := r.orderpoints[id]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("stock orderpoint #%d not found", id))
	}
	op.Active = false
	op.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListOrderpoints(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.Orderpoint], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ops []*stock.Orderpoint
	for _, op := range r.orderpoints {
		if !op.Active {
			continue
		}
		if !matchOPFilter(op, f) {
			continue
		}
		clone := *op
		ops = append(ops, &clone)
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i].ID > ops[j].ID })

	total := int64(len(ops))
	offset := page.Offset()
	if offset >= int(total) {
		return pagination.NewPageResult([]stock.Orderpoint{}, total, page), nil
	}
	limit := page.LimitClamped()
	end := offset + limit
	if end > int(total) {
		end = int(total)
	}
	items := make([]stock.Orderpoint, 0, end-offset)
	for _, op := range ops[offset:end] {
		items = append(items, *op)
	}
	return pagination.NewPageResult(items, total, page), nil
}

func matchOPFilter(op *stock.Orderpoint, f *filter.Filter) bool {
	if !matchFilterByString(op.ProductID, "product_id", f) {
		return false
	}
	if !matchFilterByString(op.WarehouseID, "warehouse_id", f) {
		return false
	}
	if !matchFilterByString(op.LocationID, "location_id", f) {
		return false
	}
	for _, c := range filterCriteria(f, "trigger") {
		if v, ok := c.Value.(string); ok && v != string(op.Trigger) {
			return false
		}
	}
	for _, c := range filterCriteria(f, "source") {
		if v, ok := c.Value.(string); ok && v != string(op.Source) {
			return false
		}
	}
	return true
}

func (r *MemoryRepo) ListOrderpointsForReplenishment(ctx context.Context, now time.Time) ([]*stock.Orderpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ops []*stock.Orderpoint
	for _, op := range r.orderpoints {
		if !op.Active {
			continue
		}
		if op.SnoozedUntil != nil && now.Before(*op.SnoozedUntil) {
			continue
		}
		if op.Trigger != stock.OrderpointTriggerAuto && op.QtyToOrderManual <= 0 {
			continue
		}
		clone := *op
		ops = append(ops, &clone)
	}
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].ProductID != ops[j].ProductID {
			return ops[i].ProductID < ops[j].ProductID
		}
		return ops[i].LocationID < ops[j].LocationID
	})
	return ops, nil
}

func (r *MemoryRepo) StockForecast(ctx context.Context, productID, locationID int64, at time.Time) (onHand, incoming, outgoing float64, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if q, ok := r.quants[quantKey(productID, locationID)]; ok {
		onHand = q.Quantity
	}
	for _, m := range r.moves {
		if m.ProductID != productID {
			continue
		}
		if !m.Date.After(at) {
			switch m.State {
			case stock.MoveStateConfirmed, stock.MoveStateAssigned:
				if m.LocationDestID == locationID {
					incoming += m.ProductQty
				}
				if m.LocationID == locationID {
					outgoing += m.ProductQty
				}
			}
		}
	}
	return onHand, incoming, outgoing, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Landed Costs (Phase 13 — stock.landed.cost)
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateLandedCost(ctx context.Context, lc *stock.LandedCost) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastLCID++
	lc.ID = r.lastLCID
	if lc.Name == "" {
		year := lc.Date.UTC().Year()
		r.lcCounters[year]++
		lc.Name = fmt.Sprintf("LC/%d/%05d", year, r.lcCounters[year])
	}
	now := time.Now().UTC()
	lc.CreatedAt = now
	lc.UpdatedAt = now
	for i := range lc.CostLines {
		lc.CostLines[i].LandedCostID = lc.ID
		lc.CostLines[i].CreatedAt = now
		lc.CostLines[i].UpdatedAt = now
	}
	for i := range lc.ValuationAdjustments {
		lc.ValuationAdjustments[i].LandedCostID = lc.ID
		lc.ValuationAdjustments[i].CreatedAt = now
		lc.ValuationAdjustments[i].UpdatedAt = now
	}
	clone := *lc
	r.landedCosts[lc.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetLandedCostByID(ctx context.Context, id int64) (*stock.LandedCost, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lc, ok := r.landedCosts[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("stock landed cost #%d not found", id))
	}
	clone := *lc
	return &clone, nil
}

func (r *MemoryRepo) UpdateLandedCost(ctx context.Context, lc *stock.LandedCost) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.landedCosts[lc.ID]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("stock landed cost #%d not found", lc.ID))
	}
	now := time.Now().UTC()
	lc.UpdatedAt = now
	lc.CreatedAt = existing.CreatedAt
	for i := range lc.CostLines {
		lc.CostLines[i].LandedCostID = lc.ID
		lc.CostLines[i].CreatedAt = now
		lc.CostLines[i].UpdatedAt = now
	}
	for i := range lc.ValuationAdjustments {
		lc.ValuationAdjustments[i].LandedCostID = lc.ID
		lc.ValuationAdjustments[i].CreatedAt = now
		lc.ValuationAdjustments[i].UpdatedAt = now
	}
	clone := *lc
	r.landedCosts[lc.ID] = &clone
	return nil
}

func (r *MemoryRepo) ListLandedCosts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.LandedCost], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var lcs []*stock.LandedCost
	for _, lc := range r.landedCosts {
		clone := *lc
		lcs = append(lcs, &clone)
	}
	sort.Slice(lcs, func(i, j int) bool { return lcs[i].ID > lcs[j].ID })

	total := int64(len(lcs))
	offset := page.Offset()
	if offset >= int(total) {
		return pagination.NewPageResult([]stock.LandedCost{}, total, page), nil
	}
	limit := page.LimitClamped()
	end := offset + limit
	if end > int(total) {
		end = int(total)
	}
	items := make([]stock.LandedCost, 0, end-offset)
	for _, lc := range lcs[offset:end] {
		items = append(items, *lc)
	}
	return pagination.NewPageResult(items, total, page), nil
}

func (r *MemoryRepo) DeleteLandedCost(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	lc, ok := r.landedCosts[id]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("stock landed cost #%d not found", id))
	}
	if lc.State == stock.LandedCostDone {
		return platformerrors.Conflict("stock landed cost cannot be deleted once validated")
	}
	delete(r.landedCosts, id)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Procurement Groups (stock.procurement.group)
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateProcurementGroup(ctx context.Context, pg *stock.ProcurementGroup) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastPGID++
	pg.ID = r.lastPGID
	now := time.Now().UTC()
	pg.CreatedAt = now
	pg.UpdatedAt = now
	if pg.CompanyID == 0 {
		pg.CompanyID = 1
	}
	clone := *pg
	r.procurements[pg.ID] = &clone
	return nil
}

func (r *MemoryRepo) procurementMoveIDsLocked(pgID int64) []int64 {
	var ids []int64
	for _, m := range r.moves {
		if m.ProcurementGroupID != nil && *m.ProcurementGroupID == pgID {
			ids = append(ids, m.ID)
		}
	}
	return ids
}

func (r *MemoryRepo) GetProcurementGroupByID(ctx context.Context, id int64) (*stock.ProcurementGroup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pg, ok := r.procurements[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("procurement group #%d not found", id))
	}
	clone := *pg
	clone.MoveIDs = r.procurementMoveIDsLocked(id)
	return &clone, nil
}

func (r *MemoryRepo) GetProcurementGroupByName(ctx context.Context, name string) (*stock.ProcurementGroup, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, pg := range r.procurements {
		if pg.Name == name {
			clone := *pg
			clone.MoveIDs = r.procurementMoveIDsLocked(pg.ID)
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound(fmt.Sprintf("procurement group '%s' not found", name))
}

// ─────────────────────────────────────────────────────────────────────────────
// Routes & Rules (Odoo 19 Parity)
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateRoute(ctx context.Context, route *stock.StockRoute) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastRouteID++
	route.ID = r.lastRouteID
	clone := *route
	r.routes[route.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetRouteByID(ctx context.Context, id int64) (*stock.StockRoute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.routes[id]
	if !ok || !res.Active {
		return nil, platformerrors.NotFound("route not found")
	}
	clone := *res
	return &clone, nil
}

func (r *MemoryRepo) ListRoutes(ctx context.Context, companyID *int64) ([]stock.StockRoute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var items []stock.StockRoute
	for _, res := range r.routes {
		if res.Active && (companyID == nil || res.CompanyID == nil || *res.CompanyID == *companyID) {
			items = append(items, *res)
		}
	}
	return items, nil
}

func (r *MemoryRepo) CreateRule(ctx context.Context, rule *stock.StockRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastRuleID++
	rule.ID = r.lastRuleID
	clone := *rule
	r.rules[rule.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetRuleByID(ctx context.Context, id int64) (*stock.StockRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.rules[id]
	if !ok || !res.Active {
		return nil, platformerrors.NotFound("rule not found")
	}
	clone := *res
	return &clone, nil
}

func (r *MemoryRepo) ListRulesByRoute(ctx context.Context, routeID int64) ([]stock.StockRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var items []stock.StockRule
	for _, res := range r.rules {
		if res.Active && res.RouteID == routeID {
			items = append(items, *res)
		}
	}
	return items, nil
}

func (r *MemoryRepo) FindRule(ctx context.Context, routeID int64, locationDestID int64) (*stock.StockRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, res := range r.rules {
		if res.Active && res.RouteID == routeID && res.LocationDestID == locationDestID {
			clone := *res
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound("no rule found for this route and destination")
}

// ─────────────────────────────────────────────────────────────────────────────
// Scrap
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateScrap(ctx context.Context, s *stock.StockScrap) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastScrapID++
	s.ID = r.lastScrapID
	clone := *s
	r.scraps[s.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetScrapByID(ctx context.Context, id int64) (*stock.StockScrap, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.scraps[id]
	if !ok {
		return nil, platformerrors.NotFound("scrap not found")
	}
	clone := *s
	return &clone, nil
}

func (r *MemoryRepo) UpdateScrap(ctx context.Context, s *stock.StockScrap) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.scraps[s.ID]; !ok {
		return platformerrors.NotFound("scrap not found")
	}
	clone := *s
	r.scraps[s.ID] = &clone
	return nil
}
