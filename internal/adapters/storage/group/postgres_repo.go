package groupstorage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domaingroup "cashflow_backend/internal/domain/group"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedGroupFilterFields = map[string]string{
	"id":       "id",
	"name":     "name",
	"category": "category",
	"active":   "active",
}

// PostgresRepo stores groups and permissions in PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Create(ctx context.Context, g *domaingroup.Group) error {
	if err := g.Validate(); err != nil {
		return err
	}
	query := `INSERT INTO res_groups (name, category, active, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id, created_at, updated_at`
	if err := r.pool.QueryRow(ctx, query, g.Name, g.Category, true).Scan(&g.ID, &g.Audit.CreatedAt, &g.Audit.UpdatedAt); err != nil {
		return platformerrors.Internal("failed to create group", err)
	}
	g.Active = true
	return nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id int64) (*domaingroup.Group, error) {
	query := `SELECT id, name, category, active, created_at, updated_at FROM res_groups WHERE id = $1 AND active = true`
	var g domaingroup.Group
	if err := r.pool.QueryRow(ctx, query, id).Scan(&g.ID, &g.Name, &g.Category, &g.Active, &g.Audit.CreatedAt, &g.Audit.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("group not found")
		}
		return nil, platformerrors.Internal("failed to get group", err)
	}
	return &g, nil
}

func (r *PostgresRepo) GetByName(ctx context.Context, name string) (*domaingroup.Group, error) {
	query := `SELECT id, name, category, active, created_at, updated_at FROM res_groups WHERE LOWER(name) = LOWER($1) AND active = true LIMIT 1`
	var g domaingroup.Group
	if err := r.pool.QueryRow(ctx, query, strings.TrimSpace(name)).Scan(&g.ID, &g.Name, &g.Category, &g.Active, &g.Audit.CreatedAt, &g.Audit.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("group not found")
		}
		return nil, platformerrors.Internal("failed to get group by name", err)
	}
	return &g, nil
}

func (r *PostgresRepo) Update(ctx context.Context, g *domaingroup.Group) error {
	if err := g.Validate(); err != nil {
		return err
	}
	query := `UPDATE res_groups SET name = $1, category = $2, updated_at = NOW() WHERE id = $3 AND active = true RETURNING updated_at`
	if err := r.pool.QueryRow(ctx, query, g.Name, g.Category, g.ID).Scan(&g.Audit.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound("group not found")
		}
		return platformerrors.Internal("failed to update group", err)
	}
	return nil
}

func (r *PostgresRepo) Delete(ctx context.Context, id int64) error {
	query := `UPDATE res_groups SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete group", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return platformerrors.NotFound("group not found")
	}
	return nil
}

func (r *PostgresRepo) List(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[domaingroup.Group], error) {
	if f == nil {
		f = filter.NewFilter()
	}
	f.Add("active", filter.OpEqual, true)
	whereClause, args, _, err := f.BuildWhereClause(allowedGroupFilterFields, 1)
	if err != nil {
		return pagination.PageResult[domaingroup.Group]{}, err
	}
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM res_groups %s", whereClause)
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.PageResult[domaingroup.Group]{}, platformerrors.Internal("failed to count groups", err)
	}
	query := fmt.Sprintf(`SELECT id, name, category, active, created_at, updated_at FROM res_groups %s ORDER BY id LIMIT $%d OFFSET $%d`, whereClause, len(args)+1, len(args)+2)
	args = append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return pagination.PageResult[domaingroup.Group]{}, platformerrors.Internal("failed to list groups", err)
	}
	defer rows.Close()
	items := make([]domaingroup.Group, 0)
	for rows.Next() {
		var g domaingroup.Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Category, &g.Active, &g.Audit.CreatedAt, &g.Audit.UpdatedAt); err != nil {
			return pagination.PageResult[domaingroup.Group]{}, platformerrors.Internal("failed to read group row", err)
		}
		items = append(items, g)
	}
	return pagination.NewPageResult(items, total, page), nil
}

func (r *PostgresRepo) GetByUserID(ctx context.Context, userID int64) ([]domaingroup.Group, error) {
	query := `SELECT g.id, g.name, g.category, g.active, g.created_at, g.updated_at
		FROM res_groups g
		JOIN res_groups_users_rel rel ON rel.group_id = g.id
		WHERE rel.user_id = $1 AND g.active = true`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, platformerrors.Internal("failed to load user groups", err)
	}
	defer rows.Close()
	groups := make([]domaingroup.Group, 0)
	for rows.Next() {
		var g domaingroup.Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Category, &g.Active, &g.Audit.CreatedAt, &g.Audit.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("failed to read group", err)
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (r *PostgresRepo) GetEffectiveGroupIDs(ctx context.Context, userID int64) ([]int64, error) {
	query := `WITH RECURSIVE effective_groups AS (
		SELECT rel.group_id
		FROM res_groups_users_rel rel
		JOIN res_groups g ON g.id = rel.group_id
		WHERE rel.user_id = $1 AND g.active = true
		UNION
		SELECT implied.implied_group_id
		FROM effective_groups current
		JOIN res_groups_implied_rel implied ON implied.group_id = current.group_id
		JOIN res_groups g ON g.id = implied.implied_group_id
		WHERE g.active = true
	)
	SELECT DISTINCT group_id FROM effective_groups ORDER BY group_id`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, platformerrors.Internal("failed to load effective user groups", err)
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, platformerrors.Internal("failed to read effective group", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *PostgresRepo) AssignUserToGroup(ctx context.Context, userID, groupID int64) error {
	query := `INSERT INTO res_groups_users_rel (group_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	if _, err := r.pool.Exec(ctx, query, groupID, userID); err != nil {
		return platformerrors.Internal("failed to assign user to group", err)
	}
	return nil
}

func (r *PostgresRepo) RemoveUserFromGroup(ctx context.Context, userID, groupID int64) error {
	query := `DELETE FROM res_groups_users_rel WHERE group_id = $1 AND user_id = $2`
	if _, err := r.pool.Exec(ctx, query, groupID, userID); err != nil {
		return platformerrors.Internal("failed to remove user from group", err)
	}
	return nil
}

func (r *PostgresRepo) AddImpliedGroup(ctx context.Context, groupID, impliedGroupID int64) error {
	if groupID == impliedGroupID {
		return platformerrors.BadRequest("a group cannot imply itself")
	}
	query := `INSERT INTO res_groups_implied_rel (group_id, implied_group_id)
		SELECT $1, $2
		WHERE NOT EXISTS (
			WITH RECURSIVE reachable(group_id) AS (
				SELECT $2::BIGINT
				UNION
				SELECT rel.implied_group_id FROM res_groups_implied_rel rel JOIN reachable ON reachable.group_id = rel.group_id
			)
			SELECT 1 FROM reachable WHERE group_id = $1
		)
		ON CONFLICT DO NOTHING`
	if _, err := r.pool.Exec(ctx, query, groupID, impliedGroupID); err != nil {
		return platformerrors.Internal("failed to add implied group", err)
	}
	return nil
}

func (r *PostgresRepo) RemoveImpliedGroup(ctx context.Context, groupID, impliedGroupID int64) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM res_groups_implied_rel WHERE group_id = $1 AND implied_group_id = $2`, groupID, impliedGroupID); err != nil {
		return platformerrors.Internal("failed to remove implied group", err)
	}
	return nil
}

func (r *PostgresRepo) CreatePermission(ctx context.Context, p *domaingroup.Permission) error {
	if err := p.Validate(); err != nil {
		return err
	}
	query := `INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (group_id, model) DO UPDATE SET can_read = EXCLUDED.can_read, can_create = EXCLUDED.can_create, can_update = EXCLUDED.can_update, can_delete = EXCLUDED.can_delete RETURNING id`
	if err := r.pool.QueryRow(ctx, query, p.GroupID, p.Model, p.CanRead, p.CanCreate, p.CanUpdate, p.CanDelete).Scan(&p.ID); err != nil {
		return platformerrors.Internal("failed to create permission", err)
	}
	return nil
}

func (r *PostgresRepo) GetByGroupID(ctx context.Context, groupID int64) ([]domaingroup.Permission, error) {
	query := `SELECT id, group_id, model, can_read, can_create, can_update, can_delete FROM res_group_permissions WHERE group_id = $1`
	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, platformerrors.Internal("failed to get group permissions", err)
	}
	defer rows.Close()
	perms := make([]domaingroup.Permission, 0)
	for rows.Next() {
		var p domaingroup.Permission
		if err := rows.Scan(&p.ID, &p.GroupID, &p.Model, &p.CanRead, &p.CanCreate, &p.CanUpdate, &p.CanDelete); err != nil {
			return nil, platformerrors.Internal("failed to read permission row", err)
		}
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *PostgresRepo) GetByGroupIDs(ctx context.Context, groupIDs []int64) ([]domaingroup.Permission, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	query := `SELECT id, group_id, model, can_read, can_create, can_update, can_delete FROM res_group_permissions WHERE group_id = ANY($1)`
	rows, err := r.pool.Query(ctx, query, groupIDs)
	if err != nil {
		return nil, platformerrors.Internal("failed to load permissions", err)
	}
	defer rows.Close()
	perms := make([]domaingroup.Permission, 0)
	for rows.Next() {
		var p domaingroup.Permission
		if err := rows.Scan(&p.ID, &p.GroupID, &p.Model, &p.CanRead, &p.CanCreate, &p.CanUpdate, &p.CanDelete); err != nil {
			return nil, platformerrors.Internal("failed to read permission", err)
		}
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *PostgresRepo) SetForGroup(ctx context.Context, groupID int64, perms []domaingroup.Permission) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM res_group_permissions WHERE group_id = $1`, groupID); err != nil {
		return platformerrors.Internal("failed to clear group permissions", err)
	}
	for i := range perms {
		perms[i].GroupID = groupID
		if err := perms[i].Validate(); err != nil {
			return err
		}
		if _, err := r.pool.Exec(ctx, `INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete) VALUES ($1, $2, $3, $4, $5, $6)`, groupID, perms[i].Model, perms[i].CanRead, perms[i].CanCreate, perms[i].CanUpdate, perms[i].CanDelete); err != nil {
			return platformerrors.Internal("failed to set group permissions", err)
		}
	}
	return nil
}

func (r *PostgresRepo) CheckAccess(ctx context.Context, userID int64, model, action string) (bool, error) {
	query := `WITH RECURSIVE effective_groups AS (
		SELECT rel.group_id
		FROM res_groups_users_rel rel
		JOIN res_groups g ON g.id = rel.group_id
		WHERE rel.user_id = $1 AND g.active = true
		UNION
		SELECT implied.implied_group_id
		FROM effective_groups current
		JOIN res_groups_implied_rel implied ON implied.group_id = current.group_id
		JOIN res_groups g ON g.id = implied.implied_group_id
		WHERE g.active = true
	)
	SELECT EXISTS (
		SELECT 1
		FROM effective_groups groups
		JOIN res_group_permissions p ON p.group_id = groups.group_id
		WHERE p.active = true AND LOWER(p.model) = LOWER($2) AND ((LOWER($3) = 'read' AND p.can_read) OR (LOWER($3) = 'create' AND p.can_create) OR (LOWER($3) = 'update' AND p.can_update) OR (LOWER($3) = 'delete' AND p.can_delete))
	)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, userID, model, action).Scan(&exists); err != nil {
		return false, platformerrors.Internal("failed to check access", err)
	}
	return exists, nil
}
