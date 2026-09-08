package maintenancestorage

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"cashflow_backend/internal/domain/maintenance"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is a thread-safe in-memory maintenance.Repository for tests and development.
type MemoryRepo struct {
	mu              sync.RWMutex
	categories      map[int64]*maintenance.EquipmentCategory
	stages          map[int64]*maintenance.EquipmentStage
	teams           map[int64]*maintenance.Team
	teamMembers     map[int64][]int64
	equipment       map[int64]*maintenance.Equipment
	requests        map[int64]*maintenance.MaintenanceRequest
	nextCategoryID  int64
	nextStageID     int64
	nextTeamID      int64
	nextEquipmentID int64
	nextRequestID   int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		categories:  make(map[int64]*maintenance.EquipmentCategory),
		stages:      make(map[int64]*maintenance.EquipmentStage),
		teams:       make(map[int64]*maintenance.Team),
		teamMembers: make(map[int64][]int64),
		equipment:   make(map[int64]*maintenance.Equipment),
		requests:    make(map[int64]*maintenance.MaintenanceRequest),
	}
}

func notFound(kind string, id int64) error {
	return platformerrors.NotFound(fmt.Sprintf("%s with ID %d not found", kind, id))
}
func companyAllowed(scope, companyID int64) bool { return scope <= 0 || companyID == scope }
func companyAllowedInt(scope int64, companyID *int64) bool {
	return scope <= 0 || companyID == nil || *companyID == scope
}
func companyAllowedPtr(scope *int64, companyID *int64) bool {
	return scope == nil || companyID == nil || *scope == *companyID
}

// ── Categories ──────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateCategory(ctx context.Context, v *maintenance.EquipmentCategory) error {
	r.mu.Lock(); defer r.mu.Unlock()
	r.nextCategoryID++; v.ID = r.nextCategoryID; v.Active = true; v.Audit = audit.NewFields(ctx)
	clone := *v; r.categories[v.ID] = &clone; return nil
}
func (r *MemoryRepo) GetCategoryByID(_ context.Context, companyID *int64, id int64) (*maintenance.EquipmentCategory, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	v, ok := r.categories[id]; if !ok || !companyAllowedInt(v.CompanyID, companyID) { return nil, notFound("maintenance category", id) }
	clone := *v; return &clone, nil
}
func (r *MemoryRepo) UpdateCategory(ctx context.Context, v *maintenance.EquipmentCategory) error {
	r.mu.Lock(); defer r.mu.Unlock()
	old, ok := r.categories[v.ID]; if !ok { return notFound("maintenance category", v.ID) }
	v.Audit = old.Audit; v.Audit.Touch(ctx)
	clone := *v; r.categories[v.ID] = &clone; return nil
}
func (r *MemoryRepo) DeleteCategory(_ context.Context, companyID *int64, id int64) error {
	r.mu.Lock(); defer r.mu.Unlock()
	v, ok := r.categories[id]; if !ok || !companyAllowedInt(v.CompanyID, companyID) { return notFound("maintenance category", id) }
	for _, e := range r.equipment { if e.CategoryID != nil && *e.CategoryID == id { return platformerrors.Conflict("maintenance category is in use") } }
	delete(r.categories, id); return nil
}
func (r *MemoryRepo) ListCategories(_ context.Context, companyID *int64) ([]maintenance.EquipmentCategory, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	result := make([]maintenance.EquipmentCategory, 0)
	for _, v := range r.categories { if companyAllowedInt(v.CompanyID, companyID) { result = append(result, *v) } }
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil
}

// ── Stages ──────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateStage(_ context.Context, v *maintenance.EquipmentStage) error {
	r.mu.Lock(); defer r.mu.Unlock(); r.nextStageID++; v.ID = r.nextStageID
	clone := *v; r.stages[v.ID] = &clone; return nil
}
func (r *MemoryRepo) GetStageByID(_ context.Context, id int64) (*maintenance.EquipmentStage, error) {
	r.mu.RLock(); defer r.mu.RUnlock(); v, ok := r.stages[id]; if !ok { return nil, notFound("maintenance stage", id) }
	clone := *v; return &clone, nil
}
func (r *MemoryRepo) UpdateStage(_ context.Context, v *maintenance.EquipmentStage) error {
	r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.stages[v.ID]; !ok { return notFound("maintenance stage", v.ID) }
	clone := *v; r.stages[v.ID] = &clone; return nil
}
func (r *MemoryRepo) DeleteStage(_ context.Context, id int64) error {
	r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.stages[id]; !ok { return notFound("maintenance stage", id) }
	for _, req := range r.requests { if req.StageID != nil && *req.StageID == id { return platformerrors.Conflict("maintenance stage is in use") } }
	delete(r.stages, id); return nil
}
func (r *MemoryRepo) ListStages(_ context.Context) ([]maintenance.EquipmentStage, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	result := make([]maintenance.EquipmentStage, 0, len(r.stages))
	for _, v := range r.stages { result = append(result, *v) }
	sort.Slice(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence }); return result, nil
}

// ── Teams ───────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateTeam(ctx context.Context, v *maintenance.Team) error {
	r.mu.Lock(); defer r.mu.Unlock(); if err := v.Validate(); err != nil { return err }
	r.nextTeamID++; v.ID = r.nextTeamID; v.Active = true; v.Audit = audit.NewFields(ctx)
	clone := *v; clone.MemberIDs = append([]int64(nil), v.MemberIDs...)
	r.teams[v.ID] = &clone; r.teamMembers[v.ID] = append([]int64(nil), v.MemberIDs...); return nil
}
func (r *MemoryRepo) GetTeamByID(_ context.Context, companyID *int64, id int64) (*maintenance.Team, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	v, ok := r.teams[id]; if !ok || !companyAllowedPtr(v.CompanyID, companyID) { return nil, notFound("maintenance team", id) }
	clone := *v; clone.MemberIDs = append([]int64(nil), r.teamMembers[id]...); return &clone, nil
}
func (r *MemoryRepo) UpdateTeam(ctx context.Context, v *maintenance.Team) error {
	r.mu.Lock(); defer r.mu.Unlock()
	old, ok := r.teams[v.ID]; if !ok { return notFound("maintenance team", v.ID) }
	if err := v.Validate(); err != nil { return err }
	v.Active = old.Active; v.Audit = old.Audit; v.Audit.Touch(ctx)
	clone := *v; clone.MemberIDs = append([]int64(nil), v.MemberIDs...)
	r.teams[v.ID] = &clone; r.teamMembers[v.ID] = append([]int64(nil), v.MemberIDs...); return nil
}
func (r *MemoryRepo) DeleteTeam(_ context.Context, companyID *int64, id int64) error {
	r.mu.Lock(); defer r.mu.Unlock()
	v, ok := r.teams[id]; if !ok || !companyAllowedPtr(v.CompanyID, companyID) { return notFound("maintenance team", id) }
	for _, e := range r.equipment { if e.TeamID != nil && *e.TeamID == id { return platformerrors.Conflict("maintenance team is in use") } }
	delete(r.teams, id); delete(r.teamMembers, id); return nil
}
func (r *MemoryRepo) ListTeams(_ context.Context, companyID *int64) ([]maintenance.Team, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	result := make([]maintenance.Team, 0)
	for _, v := range r.teams {
		if companyAllowedPtr(v.CompanyID, companyID) {
			clone := *v; clone.MemberIDs = append([]int64(nil), r.teamMembers[v.ID]...)
			result = append(result, clone)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil
}
func (r *MemoryRepo) SetTeamMembers(_ context.Context, teamID int64, memberIDs []int64) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if _, ok := r.teams[teamID]; !ok { return notFound("maintenance team", teamID) }
	r.teamMembers[teamID] = append([]int64(nil), memberIDs...); return nil
}

// ── Equipment ───────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateEquipment(ctx context.Context, v *maintenance.Equipment) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if err := v.Validate(); err != nil { return err }
	r.nextEquipmentID++; v.ID = r.nextEquipmentID; v.Active = true; v.Audit = audit.NewFields(ctx)
	clone := *v; r.equipment[v.ID] = &clone; return nil
}
func (r *MemoryRepo) GetEquipmentByID(_ context.Context, companyID, id int64) (*maintenance.Equipment, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	v, ok := r.equipment[id]; if !ok || !companyAllowed(companyID, v.CompanyID) { return nil, notFound("maintenance equipment", id) }
	clone := *v; return &clone, nil
}
func (r *MemoryRepo) UpdateEquipment(ctx context.Context, v *maintenance.Equipment) error {
	r.mu.Lock(); defer r.mu.Unlock()
	old, ok := r.equipment[v.ID]; if !ok { return notFound("maintenance equipment", v.ID) }
	if err := v.Validate(); err != nil { return err }
	if v.CompanyID != old.CompanyID { return platformerrors.Conflict("equipment company cannot be changed") }
	v.Audit = old.Audit; v.Audit.Touch(ctx)
	clone := *v; r.equipment[v.ID] = &clone; return nil
}
func (r *MemoryRepo) DeleteEquipment(_ context.Context, companyID, id int64) error {
	r.mu.Lock(); defer r.mu.Unlock()
	v, ok := r.equipment[id]; if !ok || !companyAllowed(companyID, v.CompanyID) { return notFound("maintenance equipment", id) }
	for _, req := range r.requests { if req.EquipmentID != nil && *req.EquipmentID == id { return platformerrors.Conflict("equipment has maintenance requests") } }
	delete(r.equipment, id); return nil
}
func (r *MemoryRepo) ListEquipments(_ context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[maintenance.Equipment], error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	result := make([]maintenance.Equipment, 0)
	for _, v := range r.equipment { if companyAllowed(companyID, v.CompanyID) { result = append(result, *v) } }
	result = applyEquipmentFilter(result, f)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return paginated(result, page), nil
}

// ── Requests ────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateRequest(ctx context.Context, v *maintenance.MaintenanceRequest) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if err := v.Validate(); err != nil { return err }
	r.nextRequestID++; v.ID = r.nextRequestID; v.Audit = audit.NewFields(ctx)
	clone := *v; r.requests[v.ID] = &clone; return nil
}
func (r *MemoryRepo) GetRequestByID(_ context.Context, companyID, id int64) (*maintenance.MaintenanceRequest, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	v, ok := r.requests[id]; if !ok || !companyAllowed(companyID, v.CompanyID) { return nil, notFound("maintenance request", id) }
	clone := *v; return &clone, nil
}
func (r *MemoryRepo) UpdateRequest(ctx context.Context, v *maintenance.MaintenanceRequest) error {
	r.mu.Lock(); defer r.mu.Unlock()
	old, ok := r.requests[v.ID]; if !ok { return notFound("maintenance request", v.ID) }
	if err := v.Validate(); err != nil { return err }
	if v.CompanyID != old.CompanyID { return platformerrors.Conflict("request company cannot be changed") }
	v.Audit = old.Audit; v.Audit.Touch(ctx)
	clone := *v; r.requests[v.ID] = &clone; return nil
}
func (r *MemoryRepo) DeleteRequest(_ context.Context, companyID, id int64) error {
	r.mu.Lock(); defer r.mu.Unlock()
	v, ok := r.requests[id]; if !ok || !companyAllowed(companyID, v.CompanyID) { return notFound("maintenance request", id) }
	delete(r.requests, id); return nil
}
func (r *MemoryRepo) ListRequests(_ context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[maintenance.MaintenanceRequest], error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	result := make([]maintenance.MaintenanceRequest, 0)
	for _, v := range r.requests { if companyAllowed(companyID, v.CompanyID) { result = append(result, *v) } }
	result = applyRequestFilter(result, f)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return paginated(result, page), nil
}
func (r *MemoryRepo) ListRecurringOpenRequests(_ context.Context) ([]maintenance.MaintenanceRequest, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	result := make([]maintenance.MaintenanceRequest, 0)
	for _, v := range r.requests {
		if v.RecurringMaintenance && !v.Archived && v.CloseDate == nil { result = append(result, *v) }
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

func paginated[T any](items []T, page pagination.PageRequest) pagination.PageResult[T] {
	res := pagination.NewPageResult(items, int64(len(items)), page)
	if res.Page <= 0 || res.Limit <= 0 { return res }
	start := (res.Page - 1) * res.Limit
	if start >= len(items) { res.Items = make([]T, 0) } else if start+res.Limit <= len(items) { res.Items = items[start : start+res.Limit] } else { res.Items = items[start:] }
	return res
}

func applyEquipmentFilter(items []maintenance.Equipment, f *filter.Filter) []maintenance.Equipment {
	if f == nil || len(f.Criteria) == 0 { return items }
	result := items[:0]
	for _, v := range items {
		if equipmentMatches(v, f) { result = append(result, v) }
	}
	return result
}

func equipmentMatches(v maintenance.Equipment, f *filter.Filter) bool {
	for _, c := range f.Criteria {
		switch c.Field {
		case "category_id":
			if !matchIntPtr(v.CategoryID, c.Operator, c.Value) { return false }
		case "team_id":
			if !matchIntPtr(v.TeamID, c.Operator, c.Value) { return false }
		case "active":
			if !matchBool(v.Active, c.Operator, c.Value) { return false }
		case "assign_to":
			if !matchString(v.AssignTo, c.Operator, c.Value) { return false }
		default:
			return false
		}
	}
	return true
}

func applyRequestFilter(items []maintenance.MaintenanceRequest, f *filter.Filter) []maintenance.MaintenanceRequest {
	if f == nil || len(f.Criteria) == 0 { return items }
	result := items[:0]
	for _, v := range items {
		if requestMatches(v, f) { result = append(result, v) }
	}
	return result
}

func requestMatches(v maintenance.MaintenanceRequest, f *filter.Filter) bool {
	for _, c := range f.Criteria {
		switch c.Field {
		case "equipment_id":
			if !matchIntPtr(v.EquipmentID, c.Operator, c.Value) { return false }
		case "team_id":
			if !matchIntPtr(v.TeamID, c.Operator, c.Value) { return false }
		case "stage_id":
			if !matchIntPtr(v.StageID, c.Operator, c.Value) { return false }
		case "maintenance_type":
			if !matchString(v.MaintenanceType, c.Operator, c.Value) { return false }
		case "priority":
			if !matchString(v.Priority, c.Operator, c.Value) { return false }
		case "kanban_state":
			if !matchString(v.KanbanState, c.Operator, c.Value) { return false }
		case "technician_user_id":
			if !matchIntPtr(v.TechnicianUserID, c.Operator, c.Value) { return false }
		case "recurring_maintenance":
			if !matchBool(v.RecurringMaintenance, c.Operator, c.Value) { return false }
		case "archived":
			if !matchBool(v.Archived, c.Operator, c.Value) { return false }
		case "close_date":
			null := c.Operator == "is_null" || c.Operator == filter.OpIsNull
			if null != (v.CloseDate == nil) { return false }
		default:
			return false
		}
	}
	return true
}

func matchIntPtr(ptr *int64, op string, val any) bool {
	if ptr == nil { return matchNil(op) }
	target, ok := toInt64(val)
	if !ok { return false }
	switch op {
	case filter.OpEqual, "=": return *ptr == target
	case filter.OpNotEqual, "!=", "<>": return *ptr != target
	case filter.OpGreaterThan, ">": return *ptr > target
	case filter.OpGreaterThanOrEqual, ">=": return *ptr >= target
	case filter.OpLessThan, "<": return *ptr < target
	case filter.OpLessThanOrEqual, "<=": return *ptr <= target
	default: return false
	}
}

func toInt64(val any) (int64, bool) {
	switch t := val.(type) {
	case int: return int64(t), true
	case int64: return t, true
	case float64: return int64(t), true
	case string:
		var out int64
		if _, err := fmt.Sscanf(t, "%d", &out); err != nil { return 0, false }
		return out, true
	default: return 0, false
	}
}

func matchString(value, op string, val any) bool {
	target, ok := val.(string)
	if !ok { return false }
	switch op {
	case filter.OpEqual, "=": return value == target
	case filter.OpNotEqual, "!=", "<>": return value != target
	case filter.OpLike, filter.OpILike: return containsFold(value, target)
	default: return false
	}
}

func matchBool(value bool, op string, val any) bool {
	target, ok := val.(bool)
	if !ok {
		if s, ok := val.(string); ok { target = s == "true" } else { return false }
	}
	switch op {
	case filter.OpEqual, "=": return value == target
	case filter.OpNotEqual, "!=": return value != target
	default: return false
	}
}

func matchNil(op string) bool { return op == "is_null" || op == filter.OpIsNull }

func containsFold(haystack, needle string) bool {
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