package groupstorage

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	domaingroup "cashflow_backend/internal/domain/group"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is an in-memory RBAC repository used for local dev and tests.
type MemoryRepo struct {
	mu          sync.RWMutex
	groups      map[int64]*domaingroup.Group
	permissions map[int64][]domaingroup.Permission
	userGroups  map[int64]map[int64]bool
	lastGroupID int64
	lastPermID  int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		groups:      make(map[int64]*domaingroup.Group),
		permissions: make(map[int64][]domaingroup.Permission),
		userGroups:  make(map[int64]map[int64]bool),
	}
}

func (r *MemoryRepo) Create(ctx context.Context, g *domaingroup.Group) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if g == nil {
		return platformerrors.BadRequest("group cannot be nil")
	}
	if err := g.Validate(); err != nil {
		return err
	}
	r.lastGroupID++
	g.ID = r.lastGroupID
	g.Active = true
	if g.Audit.CreatedAt.IsZero() {
		g.Audit.CreatedAt = time.Now().UTC()
	}
	if g.Audit.UpdatedAt.IsZero() {
		g.Audit.UpdatedAt = g.Audit.CreatedAt
	}
	clone := *g
	r.groups[g.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetByID(ctx context.Context, id int64) (*domaingroup.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.groups[id]
	if !ok || !g.Active {
		return nil, platformerrors.NotFound("group not found")
	}
	clone := *g
	return &clone, nil
}

func (r *MemoryRepo) GetByName(ctx context.Context, name string) (*domaingroup.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, g := range r.groups {
		if g.Active && strings.EqualFold(g.Name, strings.TrimSpace(name)) {
			clone := *g
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound("group not found")
}

func (r *MemoryRepo) Update(ctx context.Context, g *domaingroup.Group) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := g.Validate(); err != nil {
		return err
	}
	current, ok := r.groups[g.ID]
	if !ok || !current.Active {
		return platformerrors.NotFound("group not found")
	}
	g.Audit.UpdatedAt = time.Now().UTC()
	clone := *g
	r.groups[g.ID] = &clone
	return nil
}

func (r *MemoryRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.groups[id]
	if !ok || !g.Active {
		return platformerrors.NotFound("group not found")
	}
	g.Active = false
	g.Audit.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[domaingroup.Group], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var filtered []*domaingroup.Group
	for _, g := range r.groups {
		if !g.Active {
			continue
		}
		if f != nil && len(f.Criteria) > 0 {
			match := true
			for _, c := range f.Criteria {
				switch strings.ToLower(c.Field) {
				case "name":
					if val, ok := c.Value.(string); ok {
						if !strings.Contains(strings.ToLower(g.Name), strings.ToLower(val)) {
							match = false
						}
					}
				case "category":
					if val, ok := c.Value.(string); ok && !strings.EqualFold(g.Category, val) {
						match = false
					}
				case "active":
					if val, ok := c.Value.(bool); ok && g.Active != val {
						match = false
					}
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, g)
	}
	if page.SortBy == "" {
		page.SortBy = "id"
	}
	sort.Slice(filtered, func(i, j int) bool {
		if page.OrderDirection() == "DESC" {
			return filtered[i].ID > filtered[j].ID
		}
		return filtered[i].ID < filtered[j].ID
	})
	items := make([]domaingroup.Group, 0, len(filtered))
	for _, g := range filtered {
		clone := *g
		items = append(items, clone)
	}
	return pagination.NewPageResult(items, int64(len(items)), page), nil
}

func (r *MemoryRepo) GetByUserID(ctx context.Context, userID int64) ([]domaingroup.Group, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	groups := make([]domaingroup.Group, 0)
	for groupID := range r.userGroups[userID] {
		if g, ok := r.groups[groupID]; ok && g.Active {
			clone := *g
			groups = append(groups, clone)
		}
	}
	return groups, nil
}

func (r *MemoryRepo) GetEffectiveGroupIDs(ctx context.Context, userID int64) ([]int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	effective := make(map[int64]bool)
	queue := make([]int64, 0, len(r.userGroups[userID]))
	for groupID := range r.userGroups[userID] {
		queue = append(queue, groupID)
	}
	for len(queue) > 0 {
		groupID := queue[0]
		queue = queue[1:]
		if effective[groupID] {
			continue
		}
		group, ok := r.groups[groupID]
		if !ok || !group.Active {
			continue
		}
		effective[groupID] = true
		for _, impliedGroupID := range group.ImpliedGroupIDs {
			queue = append(queue, impliedGroupID)
		}
	}

	ids := make([]int64, 0, len(effective))
	for groupID := range effective {
		ids = append(ids, groupID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func (r *MemoryRepo) AssignUserToGroup(ctx context.Context, userID, groupID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.groups[groupID]; !ok {
		return platformerrors.NotFound("group not found")
	}
	if r.userGroups[userID] == nil {
		r.userGroups[userID] = map[int64]bool{}
	}
	r.userGroups[userID][groupID] = true
	return nil
}

func (r *MemoryRepo) RemoveUserFromGroup(ctx context.Context, userID, groupID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.userGroups[userID] != nil {
		delete(r.userGroups[userID], groupID)
	}
	return nil
}

func (r *MemoryRepo) AddImpliedGroup(ctx context.Context, groupID, impliedGroupID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if groupID == impliedGroupID {
		return platformerrors.BadRequest("a group cannot imply itself")
	}
	if _, ok := r.groups[groupID]; !ok {
		return platformerrors.NotFound("group not found")
	}
	if _, ok := r.groups[impliedGroupID]; !ok {
		return platformerrors.NotFound("implied group not found")
	}
	if r.reachesLocked(impliedGroupID, groupID) {
		return platformerrors.BadRequest("implied group relationship would create a cycle")
	}
	group := r.groups[groupID]
	for _, existingID := range group.ImpliedGroupIDs {
		if existingID == impliedGroupID {
			return nil
		}
	}
	group.ImpliedGroupIDs = append(group.ImpliedGroupIDs, impliedGroupID)
	return nil
}

func (r *MemoryRepo) RemoveImpliedGroup(ctx context.Context, groupID, impliedGroupID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	group, ok := r.groups[groupID]
	if !ok {
		return platformerrors.NotFound("group not found")
	}
	filtered := group.ImpliedGroupIDs[:0]
	for _, existingID := range group.ImpliedGroupIDs {
		if existingID != impliedGroupID {
			filtered = append(filtered, existingID)
		}
	}
	group.ImpliedGroupIDs = filtered
	return nil
}

func (r *MemoryRepo) reachesLocked(startID, targetID int64) bool {
	visited := map[int64]bool{}
	queue := []int64{startID}
	for len(queue) > 0 {
		groupID := queue[0]
		queue = queue[1:]
		if groupID == targetID {
			return true
		}
		if visited[groupID] {
			continue
		}
		visited[groupID] = true
		if group, ok := r.groups[groupID]; ok {
			queue = append(queue, group.ImpliedGroupIDs...)
		}
	}
	return false
}

func (r *MemoryRepo) CreatePermission(ctx context.Context, p *domaingroup.Permission) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p == nil {
		return platformerrors.BadRequest("permission cannot be nil")
	}
	if err := p.Validate(); err != nil {
		return err
	}
	r.lastPermID++
	p.ID = r.lastPermID
	groupPermissions := r.permissions[p.GroupID]
	for i := range groupPermissions {
		if groupPermissions[i].Model == p.Model {
			groupPermissions[i] = *p
			r.permissions[p.GroupID] = groupPermissions
			return nil
		}
	}
	groupPermissions = append(groupPermissions, *p)
	r.permissions[p.GroupID] = groupPermissions
	return nil
}

func (r *MemoryRepo) GetByGroupID(ctx context.Context, groupID int64) ([]domaingroup.Permission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	perms := make([]domaingroup.Permission, len(r.permissions[groupID]))
	copy(perms, r.permissions[groupID])
	return perms, nil
}

func (r *MemoryRepo) GetByGroupIDs(ctx context.Context, groupIDs []int64) ([]domaingroup.Permission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domaingroup.Permission, 0)
	seen := map[string]bool{}
	for _, groupID := range groupIDs {
		for _, perm := range r.permissions[groupID] {
			key := fmtKey(groupID, perm.Model)
			if !seen[key] {
				seen[key] = true
				out = append(out, perm)
			}
		}
	}
	return out, nil
}

func (r *MemoryRepo) SetForGroup(ctx context.Context, groupID int64, perms []domaingroup.Permission) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	newPerms := make([]domaingroup.Permission, 0, len(perms))
	for i := range perms {
		perms[i].GroupID = groupID
		if err := perms[i].Validate(); err != nil {
			return err
		}
		r.lastPermID++
		perms[i].ID = r.lastPermID
		newPerms = append(newPerms, perms[i])
	}
	r.permissions[groupID] = newPerms
	return nil
}

func (r *MemoryRepo) CheckAccess(ctx context.Context, userID int64, model, action string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if model == "" || action == "" {
		return false, nil
	}
	groupIDs := r.effectiveGroupIDsLocked(userID)
	for _, groupID := range groupIDs {
		for _, perm := range r.permissions[groupID] {
			if strings.EqualFold(perm.Model, model) && perm.Allows(action) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (r *MemoryRepo) effectiveGroupIDsLocked(userID int64) []int64 {
	effective := make(map[int64]bool)
	queue := make([]int64, 0, len(r.userGroups[userID]))
	for groupID := range r.userGroups[userID] {
		queue = append(queue, groupID)
	}
	for len(queue) > 0 {
		groupID := queue[0]
		queue = queue[1:]
		if effective[groupID] {
			continue
		}
		group, ok := r.groups[groupID]
		if !ok || !group.Active {
			continue
		}
		effective[groupID] = true
		queue = append(queue, group.ImpliedGroupIDs...)
	}
	ids := make([]int64, 0, len(effective))
	for groupID := range effective {
		ids = append(ids, groupID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func fmtKey(groupID int64, model string) string {
	return strings.ToLower(strings.TrimSpace(model)) + ":" + strconv.FormatInt(groupID, 10)
}
