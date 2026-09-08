package fleetstorage

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"cashflow_backend/internal/domain/fleet"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is a thread-safe in-memory fleet.Repository for tests and development.
type MemoryRepo struct {
	mu          sync.RWMutex
	brands      map[int64]*fleet.VehicleBrand
	modelCats   map[int64]*fleet.VehicleModelCategory
	models      map[int64]*fleet.VehicleModel
	tags        map[int64]*fleet.VehicleTag
	states      map[int64]*fleet.VehicleState
	serviceTypes map[int64]*fleet.ServiceType
	vehicles    map[int64]*fleet.Vehicle
	vehicleTags map[int64][]int64
	assignLogs  map[int64]*fleet.VehicleAssignationLog
	odometers   map[int64]*fleet.VehicleOdometer
	services    map[int64]*fleet.VehicleLogService
	contracts   map[int64]*fleet.VehicleLogContract
	next struct {
		brand, modelCat, model, tag, state, serviceType, vehicle, assign, odometer, service, contract int64
	}
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		brands:       make(map[int64]*fleet.VehicleBrand),
		modelCats:    make(map[int64]*fleet.VehicleModelCategory),
		models:       make(map[int64]*fleet.VehicleModel),
		tags:         make(map[int64]*fleet.VehicleTag),
		states:       make(map[int64]*fleet.VehicleState),
		serviceTypes: make(map[int64]*fleet.ServiceType),
		vehicles:     make(map[int64]*fleet.Vehicle),
		vehicleTags:  make(map[int64][]int64),
		assignLogs:   make(map[int64]*fleet.VehicleAssignationLog),
		odometers:    make(map[int64]*fleet.VehicleOdometer),
		services:     make(map[int64]*fleet.VehicleLogService),
		contracts:    make(map[int64]*fleet.VehicleLogContract),
	}
}

func notFound(kind string, id int64) error {
	return platformerrors.NotFound(fmt.Sprintf("%s with ID %d not found", kind, id))
}

// ── Brands ──────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateBrand(_ context.Context, v *fleet.VehicleBrand) error { r.mu.Lock(); defer r.mu.Unlock(); r.next.brand++; v.ID = r.next.brand; v.CreatedAt = now(); clone := *v; r.brands[v.ID] = &clone; return nil }
func (r *MemoryRepo) GetBrandByID(_ context.Context, id int64) (*fleet.VehicleBrand, error) { r.mu.RLock(); defer r.mu.RUnlock(); v, ok := r.brands[id]; if !ok { return nil, notFound("fleet brand", id) }; clone := *v; return &clone, nil }
func (r *MemoryRepo) UpdateBrand(_ context.Context, v *fleet.VehicleBrand) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.brands[v.ID]; !ok { return notFound("fleet brand", v.ID) }; clone := *v; r.brands[v.ID] = &clone; return nil }
func (r *MemoryRepo) DeleteBrand(_ context.Context, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.brands[id]; !ok { return notFound("fleet brand", id) }; for _, m := range r.models { if m.BrandID == id { return platformerrors.Conflict("fleet brand is in use") } }; delete(r.brands, id); return nil }
func (r *MemoryRepo) ListBrands(_ context.Context) ([]fleet.VehicleBrand, error) { r.mu.RLock(); defer r.mu.RUnlock(); result := make([]fleet.VehicleBrand, 0, len(r.brands)); for _, v := range r.brands { result = append(result, *v) }; sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil }

// ── Model categories ────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateModelCategory(_ context.Context, v *fleet.VehicleModelCategory) error { r.mu.Lock(); defer r.mu.Unlock(); r.next.modelCat++; v.ID = r.next.modelCat; v.CreatedAt = now(); clone := *v; r.modelCats[v.ID] = &clone; return nil }
func (r *MemoryRepo) GetModelCategoryByID(_ context.Context, id int64) (*fleet.VehicleModelCategory, error) { r.mu.RLock(); defer r.mu.RUnlock(); v, ok := r.modelCats[id]; if !ok { return nil, notFound("fleet model category", id) }; clone := *v; return &clone, nil }
func (r *MemoryRepo) UpdateModelCategory(_ context.Context, v *fleet.VehicleModelCategory) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.modelCats[v.ID]; !ok { return notFound("fleet model category", v.ID) }; clone := *v; r.modelCats[v.ID] = &clone; return nil }
func (r *MemoryRepo) DeleteModelCategory(_ context.Context, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.modelCats[id]; !ok { return notFound("fleet model category", id) }; for _, m := range r.models { if m.CategoryID != nil && *m.CategoryID == id { return platformerrors.Conflict("fleet model category is in use") } }; delete(r.modelCats, id); return nil }
func (r *MemoryRepo) ListModelCategories(_ context.Context) ([]fleet.VehicleModelCategory, error) { r.mu.RLock(); defer r.mu.RUnlock(); result := make([]fleet.VehicleModelCategory, 0, len(r.modelCats)); for _, v := range r.modelCats { result = append(result, *v) }; sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil }

// ── Models ──────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateModel(_ context.Context, v *fleet.VehicleModel) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if _, ok := r.brands[v.BrandID]; !ok { return platformerrors.Validation("fleet model brand not found", nil) }
	r.next.model++; v.ID = r.next.model; v.CreatedAt = now(); clone := *v; r.models[v.ID] = &clone; return nil
}
func (r *MemoryRepo) GetModelByID(_ context.Context, id int64) (*fleet.VehicleModel, error) { r.mu.RLock(); defer r.mu.RUnlock(); v, ok := r.models[id]; if !ok { return nil, notFound("fleet model", id) }; clone := *v; return &clone, nil }
func (r *MemoryRepo) UpdateModel(_ context.Context, v *fleet.VehicleModel) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.models[v.ID]; !ok { return notFound("fleet model", v.ID) }; clone := *v; r.models[v.ID] = &clone; return nil }
func (r *MemoryRepo) DeleteModel(_ context.Context, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.models[id]; !ok { return notFound("fleet model", id) }; for _, veh := range r.vehicles { if veh.ModelID == id { return platformerrors.Conflict("fleet model is in use") } }; delete(r.models, id); return nil }
func (r *MemoryRepo) ListModels(_ context.Context, brandID *int64) ([]fleet.VehicleModel, error) { r.mu.RLock(); defer r.mu.RUnlock(); result := make([]fleet.VehicleModel, 0); for _, v := range r.models { if brandID == nil || v.BrandID == *brandID { result = append(result, *v) } }; sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil }

// ── Tags ────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateTag(_ context.Context, v *fleet.VehicleTag) error { r.mu.Lock(); defer r.mu.Unlock(); r.next.tag++; v.ID = r.next.tag; v.CreatedAt = now(); clone := *v; r.tags[v.ID] = &clone; return nil }
func (r *MemoryRepo) GetTagByID(_ context.Context, id int64) (*fleet.VehicleTag, error) { r.mu.RLock(); defer r.mu.RUnlock(); v, ok := r.tags[id]; if !ok { return nil, notFound("fleet tag", id) }; clone := *v; return &clone, nil }
func (r *MemoryRepo) UpdateTag(_ context.Context, v *fleet.VehicleTag) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.tags[v.ID]; !ok { return notFound("fleet tag", v.ID) }; clone := *v; r.tags[v.ID] = &clone; return nil }
func (r *MemoryRepo) DeleteTag(_ context.Context, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.tags[id]; !ok { return notFound("fleet tag", id) }; delete(r.tags, id); for vehID, ids := range r.vehicleTags { r.vehicleTags[vehID] = removeInt(ids, id) }; return nil }
func (r *MemoryRepo) ListTags(_ context.Context) ([]fleet.VehicleTag, error) { r.mu.RLock(); defer r.mu.RUnlock(); result := make([]fleet.VehicleTag, 0, len(r.tags)); for _, v := range r.tags { result = append(result, *v) }; sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil }

// ── States ──────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateState(_ context.Context, v *fleet.VehicleState) error { r.mu.Lock(); defer r.mu.Unlock(); if err := v.Validate(); err != nil { return err }; r.next.state++; v.ID = r.next.state; clone := *v; r.states[v.ID] = &clone; return nil }
func (r *MemoryRepo) GetStateByID(_ context.Context, id int64) (*fleet.VehicleState, error) { r.mu.RLock(); defer r.mu.RUnlock(); v, ok := r.states[id]; if !ok { return nil, notFound("fleet vehicle state", id) }; clone := *v; return &clone, nil }
func (r *MemoryRepo) UpdateState(_ context.Context, v *fleet.VehicleState) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.states[v.ID]; !ok { return notFound("fleet vehicle state", v.ID) }; if err := v.Validate(); err != nil { return err }; clone := *v; r.states[v.ID] = &clone; return nil }
func (r *MemoryRepo) DeleteState(_ context.Context, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.states[id]; !ok { return notFound("fleet vehicle state", id) }; for _, veh := range r.vehicles { if veh.StateID != nil && *veh.StateID == id { return platformerrors.Conflict("fleet vehicle state is in use") } }; delete(r.states, id); return nil }
func (r *MemoryRepo) ListStates(_ context.Context) ([]fleet.VehicleState, error) { r.mu.RLock(); defer r.mu.RUnlock(); result := make([]fleet.VehicleState, 0, len(r.states)); for _, v := range r.states { result = append(result, *v) }; sort.Slice(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence }); return result, nil }

// ── Service types ───────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateServiceType(_ context.Context, v *fleet.ServiceType) error { r.mu.Lock(); defer r.mu.Unlock(); if err := v.Validate(); err != nil { return err }; r.next.serviceType++; v.ID = r.next.serviceType; clone := *v; r.serviceTypes[v.ID] = &clone; return nil }
func (r *MemoryRepo) GetServiceTypeByID(_ context.Context, id int64) (*fleet.ServiceType, error) { r.mu.RLock(); defer r.mu.RUnlock(); v, ok := r.serviceTypes[id]; if !ok { return nil, notFound("fleet service type", id) }; clone := *v; return &clone, nil }
func (r *MemoryRepo) UpdateServiceType(_ context.Context, v *fleet.ServiceType) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.serviceTypes[v.ID]; !ok { return notFound("fleet service type", v.ID) }; if err := v.Validate(); err != nil { return err }; clone := *v; r.serviceTypes[v.ID] = &clone; return nil }
func (r *MemoryRepo) DeleteServiceType(_ context.Context, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.serviceTypes[id]; !ok { return notFound("fleet service type", id) }; delete(r.serviceTypes, id); return nil }
func (r *MemoryRepo) ListServiceTypes(_ context.Context) ([]fleet.ServiceType, error) { r.mu.RLock(); defer r.mu.RUnlock(); result := make([]fleet.ServiceType, 0, len(r.serviceTypes)); for _, v := range r.serviceTypes { result = append(result, *v) }; sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil }

// ── Vehicles ────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateVehicle(ctx context.Context, v *fleet.Vehicle) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if err := v.Validate(); err != nil { return err }
	if _, ok := r.models[v.ModelID]; !ok { return platformerrors.Validation("fleet vehicle model not found", nil) }
	for _, existing := range r.vehicles { if existing.LicensePlate == v.LicensePlate && existing.CompanyID == v.CompanyID { return platformerrors.Conflict("license plate already in use") } }
	r.next.vehicle++; v.ID = r.next.vehicle; v.Active = true; v.Audit = audit.NewFields(ctx)
	clone := *v; r.vehicles[v.ID] = &clone; return nil
}
func (r *MemoryRepo) GetVehicleByID(_ context.Context, companyID, id int64) (*fleet.Vehicle, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	v, ok := r.vehicles[id]; if !ok || v.CompanyID != companyID { return nil, notFound("fleet vehicle", id) }
	clone := *v; return &clone, nil
}
func (r *MemoryRepo) UpdateVehicle(ctx context.Context, v *fleet.Vehicle) error {
	r.mu.Lock(); defer r.mu.Unlock()
	old, ok := r.vehicles[v.ID]; if !ok { return notFound("fleet vehicle", v.ID) }
	if err := v.Validate(); err != nil { return err }
	if v.CompanyID != old.CompanyID { return platformerrors.Conflict("fleet vehicle company cannot be changed") }
	for id, existing := range r.vehicles { if id != v.ID && existing.LicensePlate == v.LicensePlate && existing.CompanyID == v.CompanyID { return platformerrors.Conflict("license plate already in use") } }
	v.Active = old.Active; v.Audit = old.Audit; v.Audit.Touch(ctx)
	clone := *v; r.vehicles[v.ID] = &clone; return nil
}
func (r *MemoryRepo) DeleteVehicle(_ context.Context, companyID, id int64) error {
	r.mu.Lock(); defer r.mu.Unlock()
	v, ok := r.vehicles[id]; if !ok || v.CompanyID != companyID { return notFound("fleet vehicle", id) }
	for _, s := range r.services { if s.VehicleID == id { return platformerrors.Conflict("fleet vehicle has service logs") } }
	for _, c := range r.contracts { if c.VehicleID == id { return platformerrors.Conflict("fleet vehicle has contracts") } }
	delete(r.vehicles, id); delete(r.vehicleTags, id); return nil
}
func (r *MemoryRepo) ListVehicles(_ context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[fleet.Vehicle], error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	result := make([]fleet.Vehicle, 0)
	for _, v := range r.vehicles {
		if v.CompanyID != companyID || !vehicleMatches(*v, f) { continue }
		result = append(result, *v)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return fleetPaginated(result, page), nil
}
func (r *MemoryRepo) SetVehicleTags(_ context.Context, companyID, vehicleID int64, tagIDs []int64) error {
	r.mu.Lock(); defer r.mu.Unlock()
	v, ok := r.vehicles[vehicleID]; if !ok || v.CompanyID != companyID { return notFound("fleet vehicle", vehicleID) }
	for _, id := range tagIDs { if _, ok := r.tags[id]; !ok { return notFound("fleet tag", id) } }
	r.vehicleTags[vehicleID] = append([]int64(nil), tagIDs...)
	return nil
}
func (r *MemoryRepo) ListVehicleTagIDs(_ context.Context, companyID, vehicleID int64) ([]int64, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	v, ok := r.vehicles[vehicleID]; if !ok || v.CompanyID != companyID { return nil, notFound("fleet vehicle", vehicleID) }
	return append([]int64(nil), r.vehicleTags[vehicleID]...), nil
}

// ── Assignation logs ────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateAssignationLog(_ context.Context, v *fleet.VehicleAssignationLog) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if _, ok := r.vehicles[v.VehicleID]; !ok { return platformerrors.Validation("fleet vehicle not found", nil) }
	if overlap, err := r.overlappingLocked(v.VehicleID, v.DateStart, v.DateEnd, 0); err != nil { return err } else if overlap { return platformerrors.Conflict("assignation dates overlap an existing assignation for this vehicle") }
	r.next.assign++; v.ID = r.next.assign; v.CreatedAt = now(); clone := *v; r.assignLogs[v.ID] = &clone; return nil
}
func (r *MemoryRepo) GetAssignationLogByID(_ context.Context, id int64) (*fleet.VehicleAssignationLog, error) { r.mu.RLock(); defer r.mu.RUnlock(); v, ok := r.assignLogs[id]; if !ok { return nil, notFound("fleet assignation log", id) }; clone := *v; return &clone, nil }
func (r *MemoryRepo) UpdateAssignationLog(_ context.Context, v *fleet.VehicleAssignationLog) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if _, ok := r.assignLogs[v.ID]; !ok { return notFound("fleet assignation log", v.ID) }
	if overlap, err := r.overlappingLocked(v.VehicleID, v.DateStart, v.DateEnd, v.ID); err != nil { return err } else if overlap { return platformerrors.Conflict("assignation dates overlap an existing assignation for this vehicle") }
	v.CreatedAt = r.assignLogs[v.ID].CreatedAt; clone := *v; r.assignLogs[v.ID] = &clone; return nil
}
func (r *MemoryRepo) DeleteAssignationLog(_ context.Context, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.assignLogs[id]; !ok { return notFound("fleet assignation log", id) }; delete(r.assignLogs, id); return nil }
func (r *MemoryRepo) ListAssignationLogs(_ context.Context, companyID, vehicleID int64) ([]fleet.VehicleAssignationLog, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	v, ok := r.vehicles[vehicleID]; if !ok || v.CompanyID != companyID { return nil, notFound("fleet vehicle", vehicleID) }
	result := make([]fleet.VehicleAssignationLog, 0)
	for _, l := range r.assignLogs { if l.VehicleID == vehicleID { result = append(result, *l) } }
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil
}
func (r *MemoryRepo) HasOverlappingAssignation(_ context.Context, vehicleID int64, start, end *time.Time, excludeID int64) (bool, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	return r.overlappingLocked(vehicleID, start, end, excludeID)
}
func (r *MemoryRepo) overlappingLocked(vehicleID int64, start, end *time.Time, excludeID int64) (bool, error) {
	if start == nil {
		return true, platformerrors.Validation("assignation requires a start date", nil)
	}
	for _, l := range r.assignLogs {
		if l.VehicleID != vehicleID || l.ID == excludeID || l.DateStart == nil {
			continue
		}
		// Existing assignation ends before the new one starts: no overlap.
		if l.DateEnd != nil && l.DateEnd.Before(*start) {
			continue
		}
		// New assignation ends before the existing one starts: no overlap.
		if end != nil && end.Before(*l.DateStart) {
			continue
		}
		return true, nil
	}
	return false, nil
}

// ── Odometers ───────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateOdometer(ctx context.Context, v *fleet.VehicleOdometer) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if _, ok := r.vehicles[v.VehicleID]; !ok { return platformerrors.Validation("fleet vehicle not found", nil) }
	latest, ok := r.latestOdometerLocked(v.VehicleID)
	if ok && v.Value < latest.Value { return platformerrors.Conflict("odometer value cannot be less than the latest reading") }
	r.next.odometer++; v.ID = r.next.odometer; v.CreatedAt = now(); clone := *v; r.odometers[v.ID] = &clone
	r.vehicles[v.VehicleID].Odometer = v.Value
	return nil
}
func (r *MemoryRepo) GetOdometerByID(_ context.Context, id int64) (*fleet.VehicleOdometer, error) { r.mu.RLock(); defer r.mu.RUnlock(); v, ok := r.odometers[id]; if !ok { return nil, notFound("fleet odometer", id) }; clone := *v; return &clone, nil }
func (r *MemoryRepo) UpdateOdometer(_ context.Context, v *fleet.VehicleOdometer) error { r.mu.Lock(); defer r.mu.Unlock(); old, ok := r.odometers[v.ID]; if !ok { return notFound("fleet odometer", v.ID) }; v.CreatedAt = old.CreatedAt; clone := *v; r.odometers[v.ID] = &clone; return nil }
func (r *MemoryRepo) DeleteOdometer(_ context.Context, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.odometers[id]; !ok { return notFound("fleet odometer", id) }; delete(r.odometers, id); return nil }
func (r *MemoryRepo) ListOdometers(_ context.Context, companyID, vehicleID int64) ([]fleet.VehicleOdometer, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	v, ok := r.vehicles[vehicleID]; if !ok || v.CompanyID != companyID { return nil, notFound("fleet vehicle", vehicleID) }
	result := make([]fleet.VehicleOdometer, 0)
	for _, o := range r.odometers { if o.VehicleID == vehicleID { result = append(result, *o) } }
	sort.Slice(result, func(i, j int) bool { return result[i].Date.Before(result[j].Date) }); return result, nil
}
func (r *MemoryRepo) GetLatestOdometer(_ context.Context, companyID, vehicleID int64) (*fleet.VehicleOdometer, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	v, ok := r.vehicles[vehicleID]; if !ok || v.CompanyID != companyID { return nil, notFound("fleet vehicle", vehicleID) }
	latest, ok := r.latestOdometerLocked(vehicleID)
	if !ok { return nil, nil }
	clone := *latest; return &clone, nil
}
func (r *MemoryRepo) latestOdometerLocked(vehicleID int64) (*fleet.VehicleOdometer, bool) {
	var latest *fleet.VehicleOdometer
	for _, o := range r.odometers {
		if o.VehicleID != vehicleID { continue }
		if latest == nil || o.Value > latest.Value { latest = o }
	}
	return latest, latest != nil
}

// ── Service logs ────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateLogService(ctx context.Context, v *fleet.VehicleLogService) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if err := v.Validate(); err != nil { return err }
	veh, ok := r.vehicles[v.VehicleID]; if !ok || veh.CompanyID != v.CompanyID { return platformerrors.Validation("fleet vehicle not found for company", nil) }
	r.next.service++; v.ID = r.next.service; v.Audit = audit.NewFields(ctx); clone := *v; r.services[v.ID] = &clone; return nil
}
func (r *MemoryRepo) GetLogServiceByID(_ context.Context, companyID, id int64) (*fleet.VehicleLogService, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	v, ok := r.services[id]; if !ok || v.CompanyID != companyID { return nil, notFound("fleet service log", id) }
	clone := *v; return &clone, nil
}
func (r *MemoryRepo) UpdateLogService(ctx context.Context, v *fleet.VehicleLogService) error { r.mu.Lock(); defer r.mu.Unlock(); old, ok := r.services[v.ID]; if !ok { return notFound("fleet service log", v.ID) }; if err := v.Validate(); err != nil { return err }; if v.CompanyID != old.CompanyID { return platformerrors.Conflict("fleet service log company cannot be changed") }; v.Audit = old.Audit; v.Audit.Touch(ctx); clone := *v; r.services[v.ID] = &clone; return nil }
func (r *MemoryRepo) DeleteLogService(_ context.Context, companyID, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); v, ok := r.services[id]; if !ok || v.CompanyID != companyID { return notFound("fleet service log", id) }; delete(r.services, id); return nil }
func (r *MemoryRepo) ListLogServices(_ context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[fleet.VehicleLogService], error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	result := make([]fleet.VehicleLogService, 0)
	for _, v := range r.services { if v.CompanyID == companyID && serviceMatches(*v, f) { result = append(result, *v) } }
	sort.Slice(result, func(i, j int) bool { return result[i].Date.Before(result[j].Date) })
	return fleetPaginated(result, page), nil
}

// ── Contracts ───────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateLogContract(ctx context.Context, v *fleet.VehicleLogContract) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if err := v.Validate(); err != nil { return err }
	veh, ok := r.vehicles[v.VehicleID]; if !ok || veh.CompanyID != v.CompanyID { return platformerrors.Validation("fleet vehicle not found for company", nil) }
	r.next.contract++; v.ID = r.next.contract; v.Audit = audit.NewFields(ctx); clone := *v; r.contracts[v.ID] = &clone; return nil
}
func (r *MemoryRepo) GetLogContractByID(_ context.Context, companyID, id int64) (*fleet.VehicleLogContract, error) { r.mu.RLock(); defer r.mu.RUnlock(); v, ok := r.contracts[id]; if !ok || v.CompanyID != companyID { return nil, notFound("fleet contract", id) }; clone := *v; return &clone, nil }
func (r *MemoryRepo) UpdateLogContract(ctx context.Context, v *fleet.VehicleLogContract) error { r.mu.Lock(); defer r.mu.Unlock(); old, ok := r.contracts[v.ID]; if !ok { return notFound("fleet contract", v.ID) }; if err := v.Validate(); err != nil { return err }; if v.CompanyID != old.CompanyID { return platformerrors.Conflict("fleet contract company cannot be changed") }; v.Audit = old.Audit; v.Audit.Touch(ctx); clone := *v; r.contracts[v.ID] = &clone; return nil }
func (r *MemoryRepo) DeleteLogContract(_ context.Context, companyID, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); v, ok := r.contracts[id]; if !ok || v.CompanyID != companyID { return notFound("fleet contract", id) }; delete(r.contracts, id); return nil }
func (r *MemoryRepo) ListLogContracts(_ context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[fleet.VehicleLogContract], error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	result := make([]fleet.VehicleLogContract, 0)
	for _, v := range r.contracts { if v.CompanyID == companyID && contractMatches(*v, f) { result = append(result, *v) } }
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return fleetPaginated(result, page), nil
}
func (r *MemoryRepo) ListContractsForRefresh(_ context.Context) ([]fleet.VehicleLogContract, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	result := make([]fleet.VehicleLogContract, 0)
	for _, v := range r.contracts { if v.State == fleet.ContractStateOpen { result = append(result, *v) } }
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil
}

// ── Reports ─────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CostByVehicle(_ context.Context, companyID int64) ([]fleet.VehicleCost, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	byID := make(map[int64]*fleet.VehicleCost)
	order := make([]int64, 0)
	for _, v := range r.vehicles {
		if v.CompanyID != companyID { continue }
		byID[v.ID] = &fleet.VehicleCost{VehicleID: v.ID, VehicleName: v.Name, LicensePlate: v.LicensePlate}
		order = append(order, v.ID)
	}
	for _, s := range r.services {
		if s.CompanyID != companyID { continue }
		if c, ok := byID[s.VehicleID]; ok {
			c.TotalAmount += s.Amount
			c.ServiceCount++
			if c.LastServiceDate == nil {
				date := s.Date.Format("2006-01-02")
				c.LastServiceDate = &date
			}
		}
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	result := make([]fleet.VehicleCost, 0, len(order))
	for _, id := range order { result = append(result, *byID[id]) }
	return result, nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

func now() time.Time { return time.Now().UTC() }

func vehicleMatches(v fleet.Vehicle, f *filter.Filter) bool {
	if f == nil || len(f.Criteria) == 0 { return true }
	for _, c := range f.Criteria {
		switch c.Field {
		case "model_id":
			if !fleetMatchInt(v.ModelID, c.Operator, c.Value) { return false }
		case "state":
			if !fleetMatchString(v.State, c.Operator, c.Value) { return false }
		case "state_id":
			if v.StateID == nil { return c.Operator == filter.OpIsNull }
			if !fleetMatchInt(*v.StateID, c.Operator, c.Value) { return false }
		case "driver_id":
			if v.DriverID == nil { return c.Operator == filter.OpIsNull }
			if !fleetMatchInt(*v.DriverID, c.Operator, c.Value) { return false }
		case "active":
			if !fleetMatchBool(v.Active, c.Operator, c.Value) { return false }
		case "license_plate":
			if !fleetMatchString(v.LicensePlate, c.Operator, c.Value) { return false }
		default:
			return false
		}
	}
	return true
}

func serviceMatches(v fleet.VehicleLogService, f *filter.Filter) bool {
	if f == nil || len(f.Criteria) == 0 { return true }
	for _, c := range f.Criteria {
		switch c.Field {
		case "vehicle_id":
			if !fleetMatchInt(v.VehicleID, c.Operator, c.Value) { return false }
		case "service_type_id":
			if v.ServiceTypeID == nil { return c.Operator == filter.OpIsNull }
			if !fleetMatchInt(*v.ServiceTypeID, c.Operator, c.Value) { return false }
		case "state":
			if !fleetMatchString(v.State, c.Operator, c.Value) { return false }
		default:
			return false
		}
	}
	return true
}

func contractMatches(v fleet.VehicleLogContract, f *filter.Filter) bool {
	if f == nil || len(f.Criteria) == 0 { return true }
	for _, c := range f.Criteria {
		switch c.Field {
		case "vehicle_id":
			if !fleetMatchInt(v.VehicleID, c.Operator, c.Value) { return false }
		case "state":
			if !fleetMatchString(v.State, c.Operator, c.Value) { return false }
		case "cost_frequency":
			if !fleetMatchString(v.CostFrequency, c.Operator, c.Value) { return false }
		default:
			return false
		}
	}
	return true
}

func fleetMatchInt(value int64, op string, val any) bool {
	var target int64
	switch t := val.(type) {
	case int: target = int64(t)
	case int64: target = t
	case float64: target = int64(t)
	case string:
		var out int64
		if _, err := fmt.Sscanf(t, "%d", &out); err != nil { return false }
		target = out
	default: return false
	}
	switch op {
	case filter.OpEqual, "=": return value == target
	case filter.OpNotEqual, "!=", "<>": return value != target
	case filter.OpGreaterThan, ">": return value > target
	case filter.OpGreaterThanOrEqual, ">=": return value >= target
	case filter.OpLessThan, "<": return value < target
	case filter.OpLessThanOrEqual, "<=": return value <= target
	default: return false
	}
}
func fleetMatchString(value, op string, val any) bool {
	target, ok := val.(string)
	if !ok { return false }
	switch op {
	case filter.OpEqual, "=": return value == target
	case filter.OpNotEqual, "!=": return value != target
	case filter.OpLike, filter.OpILike: return fleetContainsFold(value, target)
	default: return false
	}
}
func fleetMatchBool(value bool, op string, val any) bool {
	target, ok := val.(bool)
	if !ok { if s, ok := val.(string); ok { target = s == "true" } else { return false } }
	switch op {
	case filter.OpEqual, "=": return value == target
	case filter.OpNotEqual, "!=": return value != target
	default: return false
	}
}
func fleetContainsFold(haystack, needle string) bool {
	if needle == "" { return true }
	needle = lower(needle)
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if lower(haystack[i:i+len(needle)]) == needle { return true }
	}
	return false
}
func lower(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' { c += 'a' - 'A' }
		out[i] = c
	}
	return string(out)
}
func removeInt(values []int64, target int64) []int64 {
	result := values[:0]
	for _, v := range values { if v != target { result = append(result, v) } }
	return result
}

func fleetPaginated[T any](items []T, page pagination.PageRequest) pagination.PageResult[T] {
	res := pagination.NewPageResult(items, int64(len(items)), page)
	if res.Page <= 0 || res.Limit <= 0 { return res }
	start := (res.Page - 1) * res.Limit
	if start >= len(items) { res.Items = make([]T, 0) } else if start+res.Limit <= len(items) { res.Items = items[start : start+res.Limit] } else { res.Items = items[start:] }
	return res
}