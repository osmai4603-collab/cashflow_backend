package maintenancestorage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/maintenance"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedEquipmentFilterFields = map[string]string{
	"category_id": "category_id",
	"team_id":     "team_id",
	"active":      "active",
	"assign_to":   "assign_to",
	"company_id":  "company_id",
}

var allowedRequestFilterFields = map[string]string{
	"equipment_id":         "equipment_id",
	"team_id":              "team_id",
	"stage_id":             "stage_id",
	"maintenance_type":     "maintenance_type",
	"priority":             "priority",
	"kanban_state":         "kanban_state",
	"technician_user_id":   "technician_user_id",
	"recurring_maintenance": "recurring_maintenance",
	"archived":             "archived",
	"close_date":           "close_date",
	"company_id":           "company_id",
}

const maintenanceEquipmentColumns = `
	id, name, category_id, team_id, technician_user_id, owner_user_id, employee_id,
	department_id, assign_to, partner_id, partner_ref, location_id, serial_no, model,
	warranty_date, effective_date, next_action_date, period, cost, notes, assign_date,
	scrap_date, active, company_id, created_at, updated_at, created_by, updated_by
`

const maintenanceRequestColumns = `
	id, name, equipment_id, team_id, request_date, close_date, schedule_date, schedule_end,
	maintenance_type, priority, stage_id, kanban_state, technician_user_id, owner_user_id,
	employee_id, department_id, duration, description, recurring_maintenance, repeat_interval,
	repeat_unit, repeat_type, repeat_until, archived, company_id, created_at, updated_at, created_by, updated_by
`

// PostgresRepo implements maintenance.Repository on PostgreSQL.
type PostgresRepo struct{ pool *pgxpool.Pool }

// NewPostgresRepo creates a Postgres maintenance repository.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo { return &PostgresRepo{pool: pool} }

// ── Categories ──────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateCategory(ctx context.Context, v *maintenance.EquipmentCategory) error {
	query := `
		INSERT INTO maintenance_equipment_categories (name, color, active, company_id, created_at, updated_at, created_by, updated_by)
		VALUES ($1, $2, $3, $4, NOW(), NOW(), $5, $5)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query, v.Name, v.Color, v.Active, v.CompanyID, auditCreatedBy(ctx)).
		Scan(&v.ID, &v.Audit.CreatedAt, &v.Audit.UpdatedAt)
}

func (r *PostgresRepo) GetCategoryByID(ctx context.Context, companyID *int64, id int64) (*maintenance.EquipmentCategory, error) {
	query := `
		SELECT id, name, color, active, company_id, created_at, updated_at, created_by, updated_by
		FROM maintenance_equipment_categories WHERE id = $1
	`
	args := []any{id}
	if companyID != nil {
		query += ` AND (company_id IS NULL OR company_id = $2)`
		args = append(args, *companyID)
	}
	var v maintenance.EquipmentCategory
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&v.ID, &v.Name, &v.Color, &v.Active, &v.CompanyID,
		&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("maintenance category", id) }
		return nil, platformerrors.Internal("failed to get maintenance category", err)
	}
	return &v, nil
}

func (r *PostgresRepo) UpdateCategory(ctx context.Context, v *maintenance.EquipmentCategory) error {
	query := `
		UPDATE maintenance_equipment_categories
		SET name = $1, color = $2, active = $3, updated_at = NOW(), updated_by = $4
		WHERE id = $5 RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query, v.Name, v.Color, v.Active, auditCreatedBy(ctx), v.ID).Scan(&v.Audit.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return notFound("maintenance category", v.ID) }
		return platformerrors.Internal("failed to update maintenance category", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteCategory(ctx context.Context, companyID *int64, id int64) error {
	query := `DELETE FROM maintenance_equipment_categories WHERE id = $1`
	args := []any{id}
	if companyID != nil {
		query += ` AND (company_id IS NULL OR company_id = $2)`
		args = append(args, *companyID)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		if isForeignKeyViolation(err) { return platformerrors.Conflict("maintenance category is in use") }
		return platformerrors.Internal("failed to delete maintenance category", err)
	}
	if tag.RowsAffected() == 0 { return notFound("maintenance category", id) }
	return nil
}

func (r *PostgresRepo) ListCategories(ctx context.Context, companyID *int64) ([]maintenance.EquipmentCategory, error) {
	query := `SELECT id, name, color, active, company_id, created_at, updated_at, created_by, updated_by
		FROM maintenance_equipment_categories`
	var args []any
	if companyID != nil {
		query += ` WHERE company_id IS NULL OR company_id = $1`
		args = append(args, *companyID)
	}
	query += ` ORDER BY id`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil { return nil, platformerrors.Internal("failed to list maintenance categories", err) }
	defer rows.Close()

	var list []maintenance.EquipmentCategory
	for rows.Next() {
		var v maintenance.EquipmentCategory
		if err := rows.Scan(&v.ID, &v.Name, &v.Color, &v.Active, &v.CompanyID,
			&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy); err != nil {
			return nil, platformerrors.Internal("failed to scan maintenance category", err)
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// ── Stages ──────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateStage(ctx context.Context, v *maintenance.EquipmentStage) error {
	query := `INSERT INTO maintenance_stages (name, sequence, fold, done) VALUES ($1, $2, $3, $4) RETURNING id`
	return r.pool.QueryRow(ctx, query, v.Name, v.Sequence, v.Fold, v.Done).Scan(&v.ID)
}

func (r *PostgresRepo) GetStageByID(ctx context.Context, id int64) (*maintenance.EquipmentStage, error) {
	query := `SELECT id, name, sequence, fold, done FROM maintenance_stages WHERE id = $1`
	var v maintenance.EquipmentStage
	if err := r.pool.QueryRow(ctx, query, id).Scan(&v.ID, &v.Name, &v.Sequence, &v.Fold, &v.Done); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("maintenance stage", id) }
		return nil, platformerrors.Internal("failed to get maintenance stage", err)
	}
	return &v, nil
}

func (r *PostgresRepo) UpdateStage(ctx context.Context, v *maintenance.EquipmentStage) error {
	query := `UPDATE maintenance_stages SET name = $1, sequence = $2, fold = $3, done = $4 WHERE id = $5`
	tag, err := r.pool.Exec(ctx, query, v.Name, v.Sequence, v.Fold, v.Done, v.ID)
	if err != nil { return platformerrors.Internal("failed to update maintenance stage", err) }
	if tag.RowsAffected() == 0 { return notFound("maintenance stage", v.ID) }
	return nil
}

func (r *PostgresRepo) DeleteStage(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM maintenance_stages WHERE id = $1`, id)
	if err != nil {
		if isForeignKeyViolation(err) { return platformerrors.Conflict("maintenance stage is in use") }
		return platformerrors.Internal("failed to delete maintenance stage", err)
	}
	if tag.RowsAffected() == 0 { return notFound("maintenance stage", id) }
	return nil
}

func (r *PostgresRepo) ListStages(ctx context.Context) ([]maintenance.EquipmentStage, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, sequence, fold, done FROM maintenance_stages ORDER BY sequence, id`)
	if err != nil { return nil, platformerrors.Internal("failed to list maintenance stages", err) }
	defer rows.Close()

	var list []maintenance.EquipmentStage
	for rows.Next() {
		var v maintenance.EquipmentStage
		if err := rows.Scan(&v.ID, &v.Name, &v.Sequence, &v.Fold, &v.Done); err != nil {
			return nil, platformerrors.Internal("failed to scan maintenance stage", err)
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// ── Teams ───────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateTeam(ctx context.Context, v *maintenance.Team) error {
	if err := v.Validate(); err != nil { return err }
	var companyID any
	if v.CompanyID != nil { companyID = *v.CompanyID }
	query := `INSERT INTO maintenance_teams (name, color, active, company_id) VALUES ($1, $2, $3, $4) RETURNING id`
	if err := r.pool.QueryRow(ctx, query, v.Name, v.Color, v.Active, companyID).Scan(&v.ID); err != nil {
		return platformerrors.Internal("failed to create maintenance team", err)
	}
	return r.SetTeamMembers(ctx, v.ID, v.MemberIDs)
}

func (r *PostgresRepo) GetTeamByID(ctx context.Context, companyID *int64, id int64) (*maintenance.Team, error) {
	query := `SELECT id, name, color, active, company_id FROM maintenance_teams WHERE id = $1`
	args := []any{id}
	if companyID != nil {
		query += ` AND (company_id IS NULL OR company_id = $2)`
		args = append(args, *companyID)
	}
	var v maintenance.Team
	err := r.pool.QueryRow(ctx, query, args...).Scan(&v.ID, &v.Name, &v.Color, &v.Active, &v.CompanyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("maintenance team", id) }
		return nil, platformerrors.Internal("failed to get maintenance team", err)
	}
	if err := r.loadTeamMembers(ctx, &v); err != nil { return nil, err }
	return &v, nil
}

func (r *PostgresRepo) UpdateTeam(ctx context.Context, v *maintenance.Team) error {
	if err := v.Validate(); err != nil { return err }
	var companyID any
	if v.CompanyID != nil { companyID = *v.CompanyID }
	query := `UPDATE maintenance_teams SET name = $1, color = $2, active = $3, company_id = $4, updated_at = NOW() WHERE id = $5`
	tag, err := r.pool.Exec(ctx, query, v.Name, v.Color, v.Active, companyID, v.ID)
	if err != nil { return platformerrors.Internal("failed to update maintenance team", err) }
	if tag.RowsAffected() == 0 { return notFound("maintenance team", v.ID) }
	return r.SetTeamMembers(ctx, v.ID, v.MemberIDs)
}

func (r *PostgresRepo) DeleteTeam(ctx context.Context, companyID *int64, id int64) error {
	query := `DELETE FROM maintenance_teams WHERE id = $1`
	args := []any{id}
	if companyID != nil {
		query += ` AND (company_id IS NULL OR company_id = $2)`
		args = append(args, *companyID)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		if isForeignKeyViolation(err) { return platformerrors.Conflict("maintenance team is in use") }
		return platformerrors.Internal("failed to delete maintenance team", err)
	}
	if tag.RowsAffected() == 0 { return notFound("maintenance team", id) }
	return nil
}

func (r *PostgresRepo) ListTeams(ctx context.Context, companyID *int64) ([]maintenance.Team, error) {
	query := `SELECT id, name, color, active, company_id FROM maintenance_teams`
	var args []any
	if companyID != nil {
		query += ` WHERE company_id IS NULL OR company_id = $1`
		args = append(args, *companyID)
	}
	query += ` ORDER BY id`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil { return nil, platformerrors.Internal("failed to list maintenance teams", err) }
	defer rows.Close()

	var list []maintenance.Team
	for rows.Next() {
		var v maintenance.Team
		if err := rows.Scan(&v.ID, &v.Name, &v.Color, &v.Active, &v.CompanyID); err != nil {
			return nil, platformerrors.Internal("failed to scan maintenance team", err)
		}
		if err := r.loadTeamMembers(ctx, &v); err != nil { return nil, err }
		list = append(list, v)
	}
	if err := rows.Err(); err != nil { return nil, err }
	sortTeams(list)
	return list, nil
}

func (r *PostgresRepo) SetTeamMembers(ctx context.Context, teamID int64, memberIDs []int64) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM maintenance_team_members WHERE team_id = $1`, teamID); err != nil {
		return platformerrors.Internal("failed to reset maintenance team members", err)
	}
	for _, memberID := range memberIDs {
		if _, err := r.pool.Exec(ctx,
			`INSERT INTO maintenance_team_members (team_id, user_id) VALUES ($1, $2)`, teamID, memberID); err != nil {
			return platformerrors.Internal("failed to add maintenance team member", err)
		}
	}
	return nil
}

func (r *PostgresRepo) loadTeamMembers(ctx context.Context, v *maintenance.Team) error {
	rows, err := r.pool.Query(ctx, `SELECT user_id FROM maintenance_team_members WHERE team_id = $1 ORDER BY user_id`, v.ID)
	if err != nil { return platformerrors.Internal("failed to load maintenance team members", err) }
	defer rows.Close()
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil { return platformerrors.Internal("failed to scan maintenance team member", err) }
		v.MemberIDs = append(v.MemberIDs, userID)
	}
	return rows.Err()
}

// ── Equipment ───────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateEquipment(ctx context.Context, v *maintenance.Equipment) error {
	query := `
		INSERT INTO maintenance_equipment (
			name, category_id, team_id, technician_user_id, owner_user_id, employee_id,
			department_id, assign_to, partner_id, partner_ref, location_id, serial_no, model,
			warranty_date, effective_date, next_action_date, period, cost, notes, assign_date,
			scrap_date, active, company_id, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23, NOW(), NOW(), $24, $24
		)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		v.Name, nullableInt(v.CategoryID), nullableInt(v.TeamID), nullableInt(v.TechnicianUserID),
		nullableInt(v.OwnerUserID), nullableInt(v.EmployeeID), nullableInt(v.DepartmentID), v.AssignTo,
		nullableInt(v.PartnerID), nullableString(v.PartnerRef), nullableInt(v.LocationID),
		nullableString(v.SerialNo), nullableString(v.Model), nullableTime(v.WarrantyDate),
		nullableTime(v.EffectiveDate), nullableTime(v.NextActionDate), v.Period, v.Cost,
		nullableString(v.Notes), nullableTime(v.AssignDate), nullableTime(v.ScrapDate),
		v.Active, v.CompanyID, auditCreatedBy(ctx),
	).Scan(&v.ID, &v.Audit.CreatedAt, &v.Audit.UpdatedAt)
}

func (r *PostgresRepo) GetEquipmentByID(ctx context.Context, companyID, id int64) (*maintenance.Equipment, error) {
	query := `SELECT ` + maintenanceEquipmentColumns + ` FROM maintenance_equipment WHERE id = $1 AND company_id = $2`
	var v maintenance.Equipment
	if err := r.pool.QueryRow(ctx, query, id, companyID).Scan(
		&v.ID, &v.Name, &v.CategoryID, &v.TeamID, &v.TechnicianUserID, &v.OwnerUserID, &v.EmployeeID,
		&v.DepartmentID, &v.AssignTo, &v.PartnerID, &v.PartnerRef, &v.LocationID, &v.SerialNo, &v.Model,
		&v.WarrantyDate, &v.EffectiveDate, &v.NextActionDate, &v.Period, &v.Cost, &v.Notes,
		&v.AssignDate, &v.ScrapDate, &v.Active, &v.CompanyID,
		&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("maintenance equipment", id) }
		return nil, platformerrors.Internal("failed to get maintenance equipment", err)
	}
	return &v, nil
}

func (r *PostgresRepo) UpdateEquipment(ctx context.Context, v *maintenance.Equipment) error {
	query := `
		UPDATE maintenance_equipment SET
			name = $1, category_id = $2, team_id = $3, technician_user_id = $4, owner_user_id = $5,
			employee_id = $6, department_id = $7, assign_to = $8, partner_id = $9, partner_ref = $10,
			location_id = $11, serial_no = $12, model = $13, warranty_date = $14, effective_date = $15,
			next_action_date = $16, period = $17, cost = $18, notes = $19, assign_date = $20,
			scrap_date = $21, active = $22, updated_at = NOW(), updated_by = $23
		WHERE id = $24 RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		v.Name, nullableInt(v.CategoryID), nullableInt(v.TeamID), nullableInt(v.TechnicianUserID),
		nullableInt(v.OwnerUserID), nullableInt(v.EmployeeID), nullableInt(v.DepartmentID), v.AssignTo,
		nullableInt(v.PartnerID), nullableString(v.PartnerRef), nullableInt(v.LocationID),
		nullableString(v.SerialNo), nullableString(v.Model), nullableTime(v.WarrantyDate),
		nullableTime(v.EffectiveDate), nullableTime(v.NextActionDate), v.Period, v.Cost,
		nullableString(v.Notes), nullableTime(v.AssignDate), nullableTime(v.ScrapDate),
		v.Active, auditCreatedBy(ctx), v.ID,
	).Scan(&v.Audit.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return notFound("maintenance equipment", v.ID) }
		return platformerrors.Internal("failed to update maintenance equipment", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteEquipment(ctx context.Context, companyID, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM maintenance_equipment WHERE id = $1 AND company_id = $2`, id, companyID)
	if err != nil {
		if isForeignKeyViolation(err) { return platformerrors.Conflict("equipment has maintenance requests") }
		return platformerrors.Internal("failed to delete maintenance equipment", err)
	}
	if tag.RowsAffected() == 0 { return notFound("maintenance equipment", id) }
	return nil
}

func (r *PostgresRepo) ListEquipments(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[maintenance.Equipment], error) {
	criteria := []filter.Criterion{{Field: "company_id", Operator: filter.OpEqual, Value: companyID}}
	if f != nil { criteria = append(criteria, f.Criteria...) }
	combined := &filter.Filter{Criteria: criteria}

	whereClause, args, nextIdx, err := combined.BuildWhereClause(allowedEquipmentFilterFields, 1)
	if err != nil {
		return pagination.PageResult[maintenance.Equipment]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}

	var totalItems int64
	if err := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM maintenance_equipment %s`, whereClause), args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[maintenance.Equipment]{}, platformerrors.Internal("failed to count maintenance equipment", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedEquipmentFilterFields[page.SortBy]; ok { sortBy = col }
	}
	limit := page.LimitClamped()
	offset := page.Offset()
	query := fmt.Sprintf(`SELECT %s FROM maintenance_equipment %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		maintenanceEquipmentColumns, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)
	rows, err := r.pool.Query(ctx, query, append(args, limit, offset)...)
	if err != nil { return pagination.PageResult[maintenance.Equipment]{}, platformerrors.Internal("failed to list maintenance equipment", err) }
	defer rows.Close()

	var list []maintenance.Equipment
	for rows.Next() {
		var v maintenance.Equipment
		if err := rows.Scan(
			&v.ID, &v.Name, &v.CategoryID, &v.TeamID, &v.TechnicianUserID, &v.OwnerUserID, &v.EmployeeID,
			&v.DepartmentID, &v.AssignTo, &v.PartnerID, &v.PartnerRef, &v.LocationID, &v.SerialNo, &v.Model,
			&v.WarrantyDate, &v.EffectiveDate, &v.NextActionDate, &v.Period, &v.Cost, &v.Notes,
			&v.AssignDate, &v.ScrapDate, &v.Active, &v.CompanyID,
			&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy,
		); err != nil {
			return pagination.PageResult[maintenance.Equipment]{}, platformerrors.Internal("failed to scan maintenance equipment", err)
		}
		list = append(list, v)
	}
	if err := rows.Err(); err != nil { return pagination.PageResult[maintenance.Equipment]{}, platformerrors.Internal("failed to iterate maintenance equipment", err) }
	return pagination.NewPageResult(list, totalItems, page), nil
}

// ── Requests ────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateRequest(ctx context.Context, v *maintenance.MaintenanceRequest) error {
	query := `
		INSERT INTO maintenance_requests (
			name, equipment_id, team_id, request_date, close_date, schedule_date, schedule_end,
			maintenance_type, priority, stage_id, kanban_state, technician_user_id, owner_user_id,
			employee_id, department_id, duration, description, recurring_maintenance, repeat_interval,
			repeat_unit, repeat_type, repeat_until, archived, company_id, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23, $24, NOW(), NOW(), $25, $25
		)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		v.Name, nullableInt(v.EquipmentID), nullableInt(v.TeamID), v.RequestDate,
		nullableTime(v.CloseDate), nullableTime(v.ScheduleDate), nullableTime(v.ScheduleEnd),
		v.MaintenanceType, v.Priority, nullableInt(v.StageID), v.KanbanState,
		nullableInt(v.TechnicianUserID), nullableInt(v.OwnerUserID), nullableInt(v.EmployeeID),
		nullableInt(v.DepartmentID), v.Duration, v.Description, v.RecurringMaintenance,
		v.RepeatInterval, string(v.RepeatUnit), string(v.RepeatType), nullableTime(v.RepeatUntil),
		v.Archived, v.CompanyID, auditCreatedBy(ctx),
	).Scan(&v.ID, &v.Audit.CreatedAt, &v.Audit.UpdatedAt)
}

func (r *PostgresRepo) GetRequestByID(ctx context.Context, companyID, id int64) (*maintenance.MaintenanceRequest, error) {
	query := `SELECT ` + maintenanceRequestColumns + ` FROM maintenance_requests WHERE id = $1 AND company_id = $2`
	var v maintenance.MaintenanceRequest
	if err := r.pool.QueryRow(ctx, query, id, companyID).Scan(
		&v.ID, &v.Name, &v.EquipmentID, &v.TeamID, &v.RequestDate, &v.CloseDate, &v.ScheduleDate, &v.ScheduleEnd,
		&v.MaintenanceType, &v.Priority, &v.StageID, &v.KanbanState, &v.TechnicianUserID, &v.OwnerUserID,
		&v.EmployeeID, &v.DepartmentID, &v.Duration, &v.Description, &v.RecurringMaintenance,
		&v.RepeatInterval, &v.RepeatUnit, &v.RepeatType, &v.RepeatUntil, &v.Archived, &v.CompanyID,
		&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("maintenance request", id) }
		return nil, platformerrors.Internal("failed to get maintenance request", err)
	}
	return &v, nil
}

func (r *PostgresRepo) UpdateRequest(ctx context.Context, v *maintenance.MaintenanceRequest) error {
	query := `
		UPDATE maintenance_requests SET
			name = $1, equipment_id = $2, team_id = $3, request_date = $4, close_date = $5,
			schedule_date = $6, schedule_end = $7, maintenance_type = $8, priority = $9, stage_id = $10,
			kanban_state = $11, technician_user_id = $12, owner_user_id = $13, employee_id = $14,
			department_id = $15, duration = $16, description = $17, recurring_maintenance = $18,
			repeat_interval = $19, repeat_unit = $20, repeat_type = $21, repeat_until = $22,
			archived = $23, updated_at = NOW(), updated_by = $24
		WHERE id = $25 RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		v.Name, nullableInt(v.EquipmentID), nullableInt(v.TeamID), v.RequestDate,
		nullableTime(v.CloseDate), nullableTime(v.ScheduleDate), nullableTime(v.ScheduleEnd),
		v.MaintenanceType, v.Priority, nullableInt(v.StageID), v.KanbanState,
		nullableInt(v.TechnicianUserID), nullableInt(v.OwnerUserID), nullableInt(v.EmployeeID),
		nullableInt(v.DepartmentID), v.Duration, v.Description, v.RecurringMaintenance,
		v.RepeatInterval, string(v.RepeatUnit), string(v.RepeatType), nullableTime(v.RepeatUntil),
		v.Archived, auditCreatedBy(ctx), v.ID,
	).Scan(&v.Audit.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return notFound("maintenance request", v.ID) }
		return platformerrors.Internal("failed to update maintenance request", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteRequest(ctx context.Context, companyID, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM maintenance_requests WHERE id = $1 AND company_id = $2`, id, companyID)
	if err != nil { return platformerrors.Internal("failed to delete maintenance request", err) }
	if tag.RowsAffected() == 0 { return notFound("maintenance request", id) }
	return nil
}

func (r *PostgresRepo) ListRequests(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[maintenance.MaintenanceRequest], error) {
	criteria := []filter.Criterion{{Field: "company_id", Operator: filter.OpEqual, Value: companyID}}
	if f != nil { criteria = append(criteria, f.Criteria...) }
	combined := &filter.Filter{Criteria: criteria}

	whereClause, args, nextIdx, err := combined.BuildWhereClause(allowedRequestFilterFields, 1)
	if err != nil {
		return pagination.PageResult[maintenance.MaintenanceRequest]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}

	var totalItems int64
	if err := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM maintenance_requests %s`, whereClause), args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[maintenance.MaintenanceRequest]{}, platformerrors.Internal("failed to count maintenance requests", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedRequestFilterFields[page.SortBy]; ok { sortBy = col }
	}
	limit := page.LimitClamped()
	offset := page.Offset()
	query := fmt.Sprintf(`SELECT %s FROM maintenance_requests %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		maintenanceRequestColumns, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)
	rows, err := r.pool.Query(ctx, query, append(args, limit, offset)...)
	if err != nil { return pagination.PageResult[maintenance.MaintenanceRequest]{}, platformerrors.Internal("failed to list maintenance requests", err) }
	defer rows.Close()

	var list []maintenance.MaintenanceRequest
	for rows.Next() {
		var v maintenance.MaintenanceRequest
		if err := rows.Scan(
			&v.ID, &v.Name, &v.EquipmentID, &v.TeamID, &v.RequestDate, &v.CloseDate, &v.ScheduleDate, &v.ScheduleEnd,
			&v.MaintenanceType, &v.Priority, &v.StageID, &v.KanbanState, &v.TechnicianUserID, &v.OwnerUserID,
			&v.EmployeeID, &v.DepartmentID, &v.Duration, &v.Description, &v.RecurringMaintenance,
			&v.RepeatInterval, &v.RepeatUnit, &v.RepeatType, &v.RepeatUntil, &v.Archived, &v.CompanyID,
			&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy,
		); err != nil {
			return pagination.PageResult[maintenance.MaintenanceRequest]{}, platformerrors.Internal("failed to scan maintenance request", err)
		}
		list = append(list, v)
	}
	if err := rows.Err(); err != nil { return pagination.PageResult[maintenance.MaintenanceRequest]{}, platformerrors.Internal("failed to iterate maintenance requests", err) }
	return pagination.NewPageResult(list, totalItems, page), nil
}

func (r *PostgresRepo) ListRecurringOpenRequests(ctx context.Context) ([]maintenance.MaintenanceRequest, error) {
	query := `SELECT ` + maintenanceRequestColumns + ` FROM maintenance_requests
		WHERE recurring_maintenance = true AND archived = false AND close_date IS NULL ORDER BY id`
	rows, err := r.pool.Query(ctx, query)
	if err != nil { return nil, platformerrors.Internal("failed to list recurring maintenance requests", err) }
	defer rows.Close()

	var list []maintenance.MaintenanceRequest
	for rows.Next() {
		var v maintenance.MaintenanceRequest
		if err := rows.Scan(
			&v.ID, &v.Name, &v.EquipmentID, &v.TeamID, &v.RequestDate, &v.CloseDate, &v.ScheduleDate, &v.ScheduleEnd,
			&v.MaintenanceType, &v.Priority, &v.StageID, &v.KanbanState, &v.TechnicianUserID, &v.OwnerUserID,
			&v.EmployeeID, &v.DepartmentID, &v.Duration, &v.Description, &v.RecurringMaintenance,
			&v.RepeatInterval, &v.RepeatUnit, &v.RepeatType, &v.RepeatUntil, &v.Archived, &v.CompanyID,
			&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan recurring maintenance request", err)
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// ── helpers ─────────────────────────────────────────────────────────────────

func sortTeams(list []maintenance.Team) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j].ID < list[j-1].ID; j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

func nullableInt(ptr *int64) any { if ptr == nil { return nil }; return *ptr }
func nullableTime(ptr *time.Time) any { if ptr == nil { return nil }; return *ptr }
func nullableString(s string) any { if s == "" { return nil }; return s }

func auditCreatedBy(ctx context.Context) any {
	if uid := audit.UserIDFromContext(ctx); uid != nil { return *uid }
	return nil
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}