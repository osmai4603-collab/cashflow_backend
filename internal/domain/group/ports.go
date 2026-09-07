package group

import (
	"context"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository manages group assignment and group metadata.
type Repository interface {
	Create(ctx context.Context, g *Group) error
	GetByID(ctx context.Context, id int64) (*Group, error)
	GetByName(ctx context.Context, name string) (*Group, error)
	Update(ctx context.Context, g *Group) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Group], error)
	GetByUserID(ctx context.Context, userID int64) ([]Group, error)
	GetEffectiveGroupIDs(ctx context.Context, userID int64) ([]int64, error)
	AssignUserToGroup(ctx context.Context, userID, groupID int64) error
	RemoveUserFromGroup(ctx context.Context, userID, groupID int64) error
	AddImpliedGroup(ctx context.Context, groupID, impliedGroupID int64) error
	RemoveImpliedGroup(ctx context.Context, groupID, impliedGroupID int64) error
}

// PermissionRepository manages RBAC permission checks.
type PermissionRepository interface {
	CreatePermission(ctx context.Context, p *Permission) error
	GetByGroupID(ctx context.Context, groupID int64) ([]Permission, error)
	GetByGroupIDs(ctx context.Context, groupIDs []int64) ([]Permission, error)
	SetForGroup(ctx context.Context, groupID int64, perms []Permission) error
	CheckAccess(ctx context.Context, userID int64, model, action string) (bool, error)
}
