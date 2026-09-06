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
	quants         map[string]*stock.StockQuant // key: "productID:locationID"
	seqInCounters  map[int]int64
	seqOutCounters map[int]int64
	seqIntCounters map[int]int64
	lastLocID      int64
	lastWhID       int64
	lastPickingID  int64
	lastMoveID     int64
	lastQuantID    int64
}

// NewMemoryRepo initializes a MemoryRepo pre-seeded with standard locations and warehouse.
func NewMemoryRepo() *MemoryRepo {
	repo := &MemoryRepo{
		locations:      make(map[int64]*stock.StockLocation),
		warehouses:     make(map[int64]*stock.Warehouse),
		pickings:       make(map[int64]*stock.StockPicking),
		moves:          make(map[int64]*stock.StockMove),
		quants:         make(map[string]*stock.StockQuant),
		seqInCounters:  make(map[int]int64),
		seqOutCounters: make(map[int]int64),
		seqIntCounters: make(map[int]int64),
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
		if loc.Active && strings.EqualFold(loc.Name, name) {
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
					if !strings.Contains(strings.ToLower(loc.Name), s) && !strings.Contains(strings.ToLower(loc.CompleteName), s) {
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
					if !strings.Contains(strings.ToLower(wh.Name), s) && !strings.Contains(strings.ToLower(wh.Code), s) {
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
				whName = w.Name
				break
			}
		}

		if warehouseID != nil && (whID == nil || *whID != *warehouseID) {
			continue
		}

		item := stock.StockOnHandItem{
			ProductID:         q.ProductID,
			LocationID:        q.LocationID,
			LocationName:      loc.CompleteName,
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

// ValidatePickingTx performs atomic picking validation and double-entry quant movement in memory.
func (r *MemoryRepo) ValidatePickingTx(ctx context.Context, picking *stock.StockPicking) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	picking.State = stock.PickingStateDone
	picking.DateDone = &now
	picking.UpdatedAt = now

	// Update moves & double-entry quant movement
	for i := range picking.Moves {
		m := &picking.Moves[i]
		qty := m.QuantityDone
		if qty <= 0 {
			qty = m.ProductQty
		}
		m.QuantityDone = qty
		m.State = stock.MoveStateDone
		m.UpdatedAt = now

		// Update move in repository map
		r.moves[m.ID] = m

		// Decrease Source Location Quant
		srcKey := quantKey(m.ProductID, m.LocationID)
		if q, ok := r.quants[srcKey]; ok {
			q.Quantity -= qty
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

	clone := *picking
	r.pickings[picking.ID] = &clone
	return nil
}
