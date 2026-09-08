package fleetstorage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/fleet"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedVehicleFilterFields = map[string]string{
	"model_id":      "model_id",
	"state":         "state",
	"state_id":      "state_id",
	"driver_id":     "driver_id",
	"active":        "active",
	"license_plate": "license_plate",
	"company_id":    "company_id",
}

var allowedServiceFilterFields = map[string]string{
	"vehicle_id":      "vehicle_id",
	"service_type_id": "service_type_id",
	"state":           "state",
	"company_id":      "company_id",
}

var allowedContractFilterFields = map[string]string{
	"vehicle_id":     "vehicle_id",
	"state":          "state",
	"cost_frequency": "cost_frequency",
	"company_id":     "company_id",
}

const fleetVehicleColumns = `
	id, name, license_plate, model_id, driver_id, future_driver_id, state_id, manager_id,
	vin_sn, acquisition_date, first_contract_date, odometer, odometer_unit, fuel_type,
	horsepower, horsepower_tax, seats, doors, color, location, state, active, company_id,
	created_at, updated_at, created_by, updated_by
`

const fleetServiceColumns = `
	id, vehicle_id, description, date, amount, vendor_id, service_type_id, state, inv_ref,
	odometer, notes, company_id, created_at, updated_at, created_by, updated_by
`

const fleetContractColumns = `
	id, vehicle_id, name, user_id, date, start_date, expiration_date, cost_generated,
	cost_frequency, ins_ref, insurer_id, state, notes, company_id, created_at, updated_at,
	created_by, updated_by
`

// PostgresRepo implements fleet.Repository on PostgreSQL.
type PostgresRepo struct{ pool *pgxpool.Pool }

// NewPostgresRepo creates a Postgres fleet repository.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo { return &PostgresRepo{pool: pool} }

// ── Brands ──────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateBrand(ctx context.Context, v *fleet.VehicleBrand) error {
	return r.pool.QueryRow(ctx, `INSERT INTO fleet_vehicle_brands (name, image_128) VALUES ($1, $2) RETURNING id, created_at`,
		v.Name, v.Image128).Scan(&v.ID, &v.CreatedAt)
}
func (r *PostgresRepo) GetBrandByID(ctx context.Context, id int64) (*fleet.VehicleBrand, error) {
	var v fleet.VehicleBrand
	err := r.pool.QueryRow(ctx, `SELECT id, name, image_128, created_at FROM fleet_vehicle_brands WHERE id = $1`, id).
		Scan(&v.ID, &v.Name, &v.Image128, &v.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("fleet brand", id) }
		return nil, platformerrors.Internal("failed to get fleet brand", err)
	}
	return &v, nil
}
func (r *PostgresRepo) UpdateBrand(ctx context.Context, v *fleet.VehicleBrand) error {
	tag, err := r.pool.Exec(ctx, `UPDATE fleet_vehicle_brands SET name = $1, image_128 = $2 WHERE id = $3`, v.Name, v.Image128, v.ID)
	if err != nil { return platformerrors.Internal("failed to update fleet brand", err) }
	if tag.RowsAffected() == 0 { return notFound("fleet brand", v.ID) }
	return nil
}
func (r *PostgresRepo) DeleteBrand(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM fleet_vehicle_brands WHERE id = $1`, id)
	if err != nil {
		if isFKViolation(err) { return platformerrors.Conflict("fleet brand is in use") }
		return platformerrors.Internal("failed to delete fleet brand", err)
	}
	if tag.RowsAffected() == 0 { return notFound("fleet brand", id) }
	return nil
}
func (r *PostgresRepo) ListBrands(ctx context.Context) ([]fleet.VehicleBrand, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, image_128, created_at FROM fleet_vehicle_brands ORDER BY id`)
	if err != nil { return nil, platformerrors.Internal("failed to list fleet brands", err) }
	defer rows.Close()
	var list []fleet.VehicleBrand
	for rows.Next() {
		var v fleet.VehicleBrand
		if err := rows.Scan(&v.ID, &v.Name, &v.Image128, &v.CreatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan fleet brand", err)
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// ── Model categories ────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateModelCategory(ctx context.Context, v *fleet.VehicleModelCategory) error {
	return r.pool.QueryRow(ctx, `INSERT INTO fleet_vehicle_model_categories (name) VALUES ($1) RETURNING id, created_at`, v.Name).Scan(&v.ID, &v.CreatedAt)
}
func (r *PostgresRepo) GetModelCategoryByID(ctx context.Context, id int64) (*fleet.VehicleModelCategory, error) {
	var v fleet.VehicleModelCategory
	err := r.pool.QueryRow(ctx, `SELECT id, name, created_at FROM fleet_vehicle_model_categories WHERE id = $1`, id).Scan(&v.ID, &v.Name, &v.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("fleet model category", id) }
		return nil, platformerrors.Internal("failed to get fleet model category", err)
	}
	return &v, nil
}
func (r *PostgresRepo) UpdateModelCategory(ctx context.Context, v *fleet.VehicleModelCategory) error {
	tag, err := r.pool.Exec(ctx, `UPDATE fleet_vehicle_model_categories SET name = $1 WHERE id = $2`, v.Name, v.ID)
	if err != nil { return platformerrors.Internal("failed to update fleet model category", err) }
	if tag.RowsAffected() == 0 { return notFound("fleet model category", v.ID) }
	return nil
}
func (r *PostgresRepo) DeleteModelCategory(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM fleet_vehicle_model_categories WHERE id = $1`, id)
	if err != nil {
		if isFKViolation(err) { return platformerrors.Conflict("fleet model category is in use") }
		return platformerrors.Internal("failed to delete fleet model category", err)
	}
	if tag.RowsAffected() == 0 { return notFound("fleet model category", id) }
	return nil
}
func (r *PostgresRepo) ListModelCategories(ctx context.Context) ([]fleet.VehicleModelCategory, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, created_at FROM fleet_vehicle_model_categories ORDER BY id`)
	if err != nil { return nil, platformerrors.Internal("failed to list fleet model categories", err) }
	defer rows.Close()
	var list []fleet.VehicleModelCategory
	for rows.Next() {
		var v fleet.VehicleModelCategory
		if err := rows.Scan(&v.ID, &v.Name, &v.CreatedAt); err != nil { return nil, platformerrors.Internal("failed to scan fleet model category", err) }
		list = append(list, v)
	}
	return list, rows.Err()
}

// ── Models ──────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateModel(ctx context.Context, v *fleet.VehicleModel) error {
	return r.pool.QueryRow(ctx, `INSERT INTO fleet_vehicle_models (name, brand_id, category_id) VALUES ($1, $2, $3) RETURNING id, created_at`,
		v.Name, v.BrandID, nullableInt(v.CategoryID)).Scan(&v.ID, &v.CreatedAt)
}
func (r *PostgresRepo) GetModelByID(ctx context.Context, id int64) (*fleet.VehicleModel, error) {
	var v fleet.VehicleModel
	err := r.pool.QueryRow(ctx, `SELECT id, name, brand_id, category_id, created_at FROM fleet_vehicle_models WHERE id = $1`, id).
		Scan(&v.ID, &v.Name, &v.BrandID, &v.CategoryID, &v.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("fleet model", id) }
		return nil, platformerrors.Internal("failed to get fleet model", err)
	}
	return &v, nil
}
func (r *PostgresRepo) UpdateModel(ctx context.Context, v *fleet.VehicleModel) error {
	tag, err := r.pool.Exec(ctx, `UPDATE fleet_vehicle_models SET name = $1, brand_id = $2, category_id = $3 WHERE id = $4`,
		v.Name, v.BrandID, nullableInt(v.CategoryID), v.ID)
	if err != nil { return platformerrors.Internal("failed to update fleet model", err) }
	if tag.RowsAffected() == 0 { return notFound("fleet model", v.ID) }
	return nil
}
func (r *PostgresRepo) DeleteModel(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM fleet_vehicle_models WHERE id = $1`, id)
	if err != nil {
		if isFKViolation(err) { return platformerrors.Conflict("fleet model is in use") }
		return platformerrors.Internal("failed to delete fleet model", err)
	}
	if tag.RowsAffected() == 0 { return notFound("fleet model", id) }
	return nil
}
func (r *PostgresRepo) ListModels(ctx context.Context, brandID *int64) ([]fleet.VehicleModel, error) {
	query := `SELECT id, name, brand_id, category_id, created_at FROM fleet_vehicle_models`
	var args []any
	if brandID != nil {
		query += ` WHERE brand_id = $1`
		args = append(args, *brandID)
	}
	query += ` ORDER BY id`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil { return nil, platformerrors.Internal("failed to list fleet models", err) }
	defer rows.Close()
	var list []fleet.VehicleModel
	for rows.Next() {
		var v fleet.VehicleModel
		if err := rows.Scan(&v.ID, &v.Name, &v.BrandID, &v.CategoryID, &v.CreatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan fleet model", err)
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// ── Tags ────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateTag(ctx context.Context, v *fleet.VehicleTag) error {
	return r.pool.QueryRow(ctx, `INSERT INTO fleet_vehicle_tags (name, color) VALUES ($1, $2) RETURNING id, created_at`, v.Name, v.Color).Scan(&v.ID, &v.CreatedAt)
}
func (r *PostgresRepo) GetTagByID(ctx context.Context, id int64) (*fleet.VehicleTag, error) {
	var v fleet.VehicleTag
	err := r.pool.QueryRow(ctx, `SELECT id, name, color, created_at FROM fleet_vehicle_tags WHERE id = $1`, id).Scan(&v.ID, &v.Name, &v.Color, &v.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("fleet tag", id) }
		return nil, platformerrors.Internal("failed to get fleet tag", err)
	}
	return &v, nil
}
func (r *PostgresRepo) UpdateTag(ctx context.Context, v *fleet.VehicleTag) error {
	tag, err := r.pool.Exec(ctx, `UPDATE fleet_vehicle_tags SET name = $1, color = $2 WHERE id = $3`, v.Name, v.Color, v.ID)
	if err != nil { return platformerrors.Internal("failed to update fleet tag", err) }
	if tag.RowsAffected() == 0 { return notFound("fleet tag", v.ID) }
	return nil
}
func (r *PostgresRepo) DeleteTag(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM fleet_vehicle_tags WHERE id = $1`, id)
	if err != nil {
		if isFKViolation(err) { return platformerrors.Conflict("fleet tag is in use") }
		return platformerrors.Internal("failed to delete fleet tag", err)
	}
	if tag.RowsAffected() == 0 { return notFound("fleet tag", id) }
	return nil
}
func (r *PostgresRepo) ListTags(ctx context.Context) ([]fleet.VehicleTag, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, color, created_at FROM fleet_vehicle_tags ORDER BY id`)
	if err != nil { return nil, platformerrors.Internal("failed to list fleet tags", err) }
	defer rows.Close()
	var list []fleet.VehicleTag
	for rows.Next() {
		var v fleet.VehicleTag
		if err := rows.Scan(&v.ID, &v.Name, &v.Color, &v.CreatedAt); err != nil { return nil, platformerrors.Internal("failed to scan fleet tag", err) }
		list = append(list, v)
	}
	return list, rows.Err()
}

// ── Vehicle states ──────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateState(ctx context.Context, v *fleet.VehicleState) error {
	return r.pool.QueryRow(ctx, `INSERT INTO fleet_vehicle_states (name, sequence, fold) VALUES ($1, $2, $3) RETURNING id`, v.Name, v.Sequence, v.Fold).Scan(&v.ID)
}
func (r *PostgresRepo) GetStateByID(ctx context.Context, id int64) (*fleet.VehicleState, error) {
	var v fleet.VehicleState
	err := r.pool.QueryRow(ctx, `SELECT id, name, sequence, fold FROM fleet_vehicle_states WHERE id = $1`, id).Scan(&v.ID, &v.Name, &v.Sequence, &v.Fold)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("fleet vehicle state", id) }
		return nil, platformerrors.Internal("failed to get fleet vehicle state", err)
	}
	return &v, nil
}
func (r *PostgresRepo) UpdateState(ctx context.Context, v *fleet.VehicleState) error {
	tag, err := r.pool.Exec(ctx, `UPDATE fleet_vehicle_states SET name = $1, sequence = $2, fold = $3 WHERE id = $4`, v.Name, v.Sequence, v.Fold, v.ID)
	if err != nil { return platformerrors.Internal("failed to update fleet vehicle state", err) }
	if tag.RowsAffected() == 0 { return notFound("fleet vehicle state", v.ID) }
	return nil
}
func (r *PostgresRepo) DeleteState(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM fleet_vehicle_states WHERE id = $1`, id)
	if err != nil {
		if isFKViolation(err) { return platformerrors.Conflict("fleet vehicle state is in use") }
		return platformerrors.Internal("failed to delete fleet vehicle state", err)
	}
	if tag.RowsAffected() == 0 { return notFound("fleet vehicle state", id) }
	return nil
}
func (r *PostgresRepo) ListStates(ctx context.Context) ([]fleet.VehicleState, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, sequence, fold FROM fleet_vehicle_states ORDER BY sequence, id`)
	if err != nil { return nil, platformerrors.Internal("failed to list fleet vehicle states", err) }
	defer rows.Close()
	var list []fleet.VehicleState
	for rows.Next() {
		var v fleet.VehicleState
		if err := rows.Scan(&v.ID, &v.Name, &v.Sequence, &v.Fold); err != nil { return nil, platformerrors.Internal("failed to scan fleet vehicle state", err) }
		list = append(list, v)
	}
	return list, rows.Err()
}

// ── Service types ───────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateServiceType(ctx context.Context, v *fleet.ServiceType) error {
	return r.pool.QueryRow(ctx, `INSERT INTO fleet_service_types (name, category) VALUES ($1, $2) RETURNING id`, v.Name, string(v.Category)).Scan(&v.ID)
}
func (r *PostgresRepo) GetServiceTypeByID(ctx context.Context, id int64) (*fleet.ServiceType, error) {
	var v fleet.ServiceType
	err := r.pool.QueryRow(ctx, `SELECT id, name, category FROM fleet_service_types WHERE id = $1`, id).
		Scan(&v.ID, &v.Name, &v.Category)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("fleet service type", id) }
		return nil, platformerrors.Internal("failed to get fleet service type", err)
	}
	return &v, nil
}
func (r *PostgresRepo) UpdateServiceType(ctx context.Context, v *fleet.ServiceType) error {
	tag, err := r.pool.Exec(ctx, `UPDATE fleet_service_types SET name = $1, category = $2 WHERE id = $3`, v.Name, string(v.Category), v.ID)
	if err != nil { return platformerrors.Internal("failed to update fleet service type", err) }
	if tag.RowsAffected() == 0 { return notFound("fleet service type", v.ID) }
	return nil
}
func (r *PostgresRepo) DeleteServiceType(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM fleet_service_types WHERE id = $1`, id)
	if err != nil {
		if isFKViolation(err) { return platformerrors.Conflict("fleet service type is in use") }
		return platformerrors.Internal("failed to delete fleet service type", err)
	}
	if tag.RowsAffected() == 0 { return notFound("fleet service type", id) }
	return nil
}
func (r *PostgresRepo) ListServiceTypes(ctx context.Context) ([]fleet.ServiceType, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, category FROM fleet_service_types ORDER BY id`)
	if err != nil { return nil, platformerrors.Internal("failed to list fleet service types", err) }
	defer rows.Close()
	var list []fleet.ServiceType
	for rows.Next() {
		var v fleet.ServiceType
		if err := rows.Scan(&v.ID, &v.Name, &v.Category); err != nil { return nil, platformerrors.Internal("failed to scan fleet service type", err) }
		list = append(list, v)
	}
	return list, rows.Err()
}

// ── Vehicles ────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateVehicle(ctx context.Context, v *fleet.Vehicle) error {
	query := `
		INSERT INTO fleet_vehicles (
			name, license_plate, model_id, driver_id, future_driver_id, state_id, manager_id,
			vin_sn, acquisition_date, first_contract_date, odometer, odometer_unit, fuel_type,
			horsepower, horsepower_tax, seats, doors, color, location, state, active, company_id,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, NOW(), NOW(), $23, $23
		)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		nullableString(v.Name), v.LicensePlate, v.ModelID, nullableInt(v.DriverID),
		nullableInt(v.FutureDriverID), nullableInt(v.StateID), nullableInt(v.ManagerID),
		nullableString(v.VIN), nullableTime(v.AcquisitionDate), nullableTime(v.FirstContractDate),
		v.Odometer, v.OdometerUnit, nullableString(v.FuelType), v.Horsepower, v.HorsepowerTax,
		v.Seats, v.Doors, nullableString(v.Color), nullableString(v.Location), v.State, v.Active,
		v.CompanyID, auditCreatedBy(ctx),
	).Scan(&v.ID, &v.Audit.CreatedAt, &v.Audit.UpdatedAt)
}
func (r *PostgresRepo) GetVehicleByID(ctx context.Context, companyID, id int64) (*fleet.Vehicle, error) {
	query := `SELECT ` + fleetVehicleColumns + ` FROM fleet_vehicles WHERE id = $1 AND company_id = $2`
	var v fleet.Vehicle
	if err := r.pool.QueryRow(ctx, query, id, companyID).Scan(
		&v.ID, &v.Name, &v.LicensePlate, &v.ModelID, &v.DriverID, &v.FutureDriverID, &v.StateID, &v.ManagerID,
		&v.VIN, &v.AcquisitionDate, &v.FirstContractDate, &v.Odometer, &v.OdometerUnit, &v.FuelType,
		&v.Horsepower, &v.HorsepowerTax, &v.Seats, &v.Doors, &v.Color, &v.Location, &v.State, &v.Active, &v.CompanyID,
		&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("fleet vehicle", id) }
		return nil, platformerrors.Internal("failed to get fleet vehicle", err)
	}
	return &v, nil
}
func (r *PostgresRepo) UpdateVehicle(ctx context.Context, v *fleet.Vehicle) error {
	query := `
		UPDATE fleet_vehicles SET
			name = $1, license_plate = $2, model_id = $3, driver_id = $4, future_driver_id = $5,
			state_id = $6, manager_id = $7, vin_sn = $8, acquisition_date = $9,
			first_contract_date = $10, odometer = $11, odometer_unit = $12, fuel_type = $13,
			horsepower = $14, horsepower_tax = $15, seats = $16, doors = $17, color = $18,
			location = $19, state = $20, active = $21, updated_at = NOW(), updated_by = $22
		WHERE id = $23 RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		nullableString(v.Name), v.LicensePlate, v.ModelID, nullableInt(v.DriverID),
		nullableInt(v.FutureDriverID), nullableInt(v.StateID), nullableInt(v.ManagerID),
		nullableString(v.VIN), nullableTime(v.AcquisitionDate), nullableTime(v.FirstContractDate),
		v.Odometer, v.OdometerUnit, nullableString(v.FuelType), v.Horsepower, v.HorsepowerTax,
		v.Seats, v.Doors, nullableString(v.Color), nullableString(v.Location), v.State, v.Active,
		auditCreatedBy(ctx), v.ID,
	).Scan(&v.Audit.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return notFound("fleet vehicle", v.ID) }
		return platformerrors.Internal("failed to update fleet vehicle", err)
	}
	return nil
}
func (r *PostgresRepo) DeleteVehicle(ctx context.Context, companyID, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM fleet_vehicles WHERE id = $1 AND company_id = $2`, id, companyID)
	if err != nil {
		if isFKViolation(err) { return platformerrors.Conflict("fleet vehicle is in use") }
		return platformerrors.Internal("failed to delete fleet vehicle", err)
	}
	if tag.RowsAffected() == 0 { return notFound("fleet vehicle", id) }
	return nil
}
func (r *PostgresRepo) ListVehicles(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[fleet.Vehicle], error) {
	criteria := []filter.Criterion{{Field: "company_id", Operator: filter.OpEqual, Value: companyID}}
	if f != nil { criteria = append(criteria, f.Criteria...) }
	combined := &filter.Filter{Criteria: criteria}

	whereClause, args, nextIdx, err := combined.BuildWhereClause(allowedVehicleFilterFields, 1)
	if err != nil {
		return pagination.PageResult[fleet.Vehicle]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}
	var totalItems int64
	if err := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM fleet_vehicles %s`, whereClause), args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[fleet.Vehicle]{}, platformerrors.Internal("failed to count fleet vehicles", err)
	}
	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedVehicleFilterFields[page.SortBy]; ok { sortBy = col }
	}
	limit := page.LimitClamped()
	offset := page.Offset()
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`SELECT %s FROM fleet_vehicles %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		fleetVehicleColumns, whereClause, sortBy, orderDir, nextIdx, nextIdx+1), append(args, limit, offset)...)
	if err != nil { return pagination.PageResult[fleet.Vehicle]{}, platformerrors.Internal("failed to list fleet vehicles", err) }
	defer rows.Close()

	var list []fleet.Vehicle
	for rows.Next() {
		var v fleet.Vehicle
		if err := rows.Scan(
			&v.ID, &v.Name, &v.LicensePlate, &v.ModelID, &v.DriverID, &v.FutureDriverID, &v.StateID, &v.ManagerID,
			&v.VIN, &v.AcquisitionDate, &v.FirstContractDate, &v.Odometer, &v.OdometerUnit, &v.FuelType,
			&v.Horsepower, &v.HorsepowerTax, &v.Seats, &v.Doors, &v.Color, &v.Location, &v.State, &v.Active, &v.CompanyID,
			&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy,
		); err != nil {
			return pagination.PageResult[fleet.Vehicle]{}, platformerrors.Internal("failed to scan fleet vehicle", err)
		}
		list = append(list, v)
	}
	if err := rows.Err(); err != nil { return pagination.PageResult[fleet.Vehicle]{}, platformerrors.Internal("failed to iterate fleet vehicles", err) }
	return pagination.NewPageResult(list, totalItems, page), nil
}
func (r *PostgresRepo) SetVehicleTags(ctx context.Context, companyID, vehicleID int64, tagIDs []int64) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM fleet_vehicle_tag_rel WHERE vehicle_id = $1`, vehicleID); err != nil {
		return platformerrors.Internal("failed to reset fleet vehicle tags", err)
	}
	for _, tagID := range tagIDs {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fleet_vehicle_tag_rel (vehicle_id, tag_id) VALUES ($1, $2)`, vehicleID, tagID); err != nil {
			return platformerrors.Internal("failed to add fleet vehicle tag", err)
		}
	}
	return nil
}
func (r *PostgresRepo) ListVehicleTagIDs(ctx context.Context, companyID, vehicleID int64) ([]int64, error) {
	if _, err := r.GetVehicleByID(ctx, companyID, vehicleID); err != nil { return nil, err }
	rows, err := r.pool.Query(ctx, `SELECT tag_id FROM fleet_vehicle_tag_rel WHERE vehicle_id = $1 ORDER BY tag_id`, vehicleID)
	if err != nil { return nil, platformerrors.Internal("failed to list fleet vehicle tags", err) }
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil { return nil, platformerrors.Internal("failed to scan fleet vehicle tag", err) }
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ── Assignation logs ────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateAssignationLog(ctx context.Context, v *fleet.VehicleAssignationLog) error {
	query := `INSERT INTO fleet_vehicle_assignation_logs (vehicle_id, driver_id, date_start, date_end) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query, v.VehicleID, v.DriverID, nullableTime(v.DateStart), nullableTime(v.DateEnd)).Scan(&v.ID, &v.CreatedAt)
}
func (r *PostgresRepo) GetAssignationLogByID(ctx context.Context, id int64) (*fleet.VehicleAssignationLog, error) {
	var v fleet.VehicleAssignationLog
	err := r.pool.QueryRow(ctx, `SELECT id, vehicle_id, driver_id, date_start, date_end, created_at FROM fleet_vehicle_assignation_logs WHERE id = $1`, id).
		Scan(&v.ID, &v.VehicleID, &v.DriverID, &v.DateStart, &v.DateEnd, &v.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("fleet assignation log", id) }
		return nil, platformerrors.Internal("failed to get fleet assignation log", err)
	}
	return &v, nil
}
func (r *PostgresRepo) UpdateAssignationLog(ctx context.Context, v *fleet.VehicleAssignationLog) error {
	tag, err := r.pool.Exec(ctx, `UPDATE fleet_vehicle_assignation_logs SET vehicle_id = $1, driver_id = $2, date_start = $3, date_end = $4 WHERE id = $5`,
		v.VehicleID, v.DriverID, nullableTime(v.DateStart), nullableTime(v.DateEnd), v.ID)
	if err != nil { return platformerrors.Internal("failed to update fleet assignation log", err) }
	if tag.RowsAffected() == 0 { return notFound("fleet assignation log", v.ID) }
	return nil
}
func (r *PostgresRepo) DeleteAssignationLog(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM fleet_vehicle_assignation_logs WHERE id = $1`, id)
	if err != nil { return platformerrors.Internal("failed to delete fleet assignation log", err) }
	if tag.RowsAffected() == 0 { return notFound("fleet assignation log", id) }
	return nil
}
func (r *PostgresRepo) ListAssignationLogs(ctx context.Context, companyID, vehicleID int64) ([]fleet.VehicleAssignationLog, error) {
	if _, err := r.GetVehicleByID(ctx, companyID, vehicleID); err != nil { return nil, err }
	rows, err := r.pool.Query(ctx, `SELECT id, vehicle_id, driver_id, date_start, date_end, created_at FROM fleet_vehicle_assignation_logs WHERE vehicle_id = $1 ORDER BY id`, vehicleID)
	if err != nil { return nil, platformerrors.Internal("failed to list fleet assignation logs", err) }
	defer rows.Close()
	var list []fleet.VehicleAssignationLog
	for rows.Next() {
		var v fleet.VehicleAssignationLog
		if err := rows.Scan(&v.ID, &v.VehicleID, &v.DriverID, &v.DateStart, &v.DateEnd, &v.CreatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan fleet assignation log", err)
		}
		list = append(list, v)
	}
	return list, rows.Err()
}
func (r *PostgresRepo) HasOverlappingAssignation(ctx context.Context, vehicleID int64, start, end *time.Time, excludeID int64) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM fleet_vehicle_assignation_logs
			WHERE vehicle_id = $1 AND id <> $2
			  AND date_start IS NOT NULL
			  AND (date_end IS NULL OR date_end >= $3)
			  AND ($4 IS NULL OR $4 >= date_start)
		)`
	var overlap bool
	if err := r.pool.QueryRow(ctx, query, vehicleID, excludeID, start, end).Scan(&overlap); err != nil {
		return false, platformerrors.Internal("failed to check fleet assignation overlap", err)
	}
	return overlap, nil
}

// ── Odometers ───────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateOdometer(ctx context.Context, v *fleet.VehicleOdometer) error {
	return r.pool.QueryRow(ctx, `INSERT INTO fleet_vehicle_odometers (vehicle_id, date, value, unit) VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		v.VehicleID, v.Date, v.Value, v.Unit).Scan(&v.ID, &v.CreatedAt)
}
func (r *PostgresRepo) GetOdometerByID(ctx context.Context, id int64) (*fleet.VehicleOdometer, error) {
	var v fleet.VehicleOdometer
	err := r.pool.QueryRow(ctx, `SELECT id, vehicle_id, date, value, unit, created_at FROM fleet_vehicle_odometers WHERE id = $1`, id).
		Scan(&v.ID, &v.VehicleID, &v.Date, &v.Value, &v.Unit, &v.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("fleet odometer", id) }
		return nil, platformerrors.Internal("failed to get fleet odometer", err)
	}
	return &v, nil
}
func (r *PostgresRepo) UpdateOdometer(ctx context.Context, v *fleet.VehicleOdometer) error {
	tag, err := r.pool.Exec(ctx, `UPDATE fleet_vehicle_odometers SET vehicle_id = $1, date = $2, value = $3, unit = $4 WHERE id = $5`,
		v.VehicleID, v.Date, v.Value, v.Unit, v.ID)
	if err != nil {
		if isFKViolation(err) { return platformerrors.Conflict("fleet odometer references a missing vehicle") }
		return platformerrors.Internal("failed to update fleet odometer", err)
	}
	if tag.RowsAffected() == 0 { return notFound("fleet odometer", v.ID) }
	return nil
}
func (r *PostgresRepo) DeleteOdometer(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM fleet_vehicle_odometers WHERE id = $1`, id)
	if err != nil { return platformerrors.Internal("failed to delete fleet odometer", err) }
	if tag.RowsAffected() == 0 { return notFound("fleet odometer", id) }
	return nil
}
func (r *PostgresRepo) ListOdometers(ctx context.Context, companyID, vehicleID int64) ([]fleet.VehicleOdometer, error) {
	if _, err := r.GetVehicleByID(ctx, companyID, vehicleID); err != nil { return nil, err }
	rows, err := r.pool.Query(ctx, `SELECT id, vehicle_id, date, value, unit, created_at FROM fleet_vehicle_odometers WHERE vehicle_id = $1 ORDER BY date, id`, vehicleID)
	if err != nil { return nil, platformerrors.Internal("failed to list fleet odometers", err) }
	defer rows.Close()
	var list []fleet.VehicleOdometer
	for rows.Next() {
		var v fleet.VehicleOdometer
		if err := rows.Scan(&v.ID, &v.VehicleID, &v.Date, &v.Value, &v.Unit, &v.CreatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan fleet odometer", err)
		}
		list = append(list, v)
	}
	return list, rows.Err()
}
func (r *PostgresRepo) GetLatestOdometer(ctx context.Context, companyID, vehicleID int64) (*fleet.VehicleOdometer, error) {
	if _, err := r.GetVehicleByID(ctx, companyID, vehicleID); err != nil { return nil, err }
	var v fleet.VehicleOdometer
	err := r.pool.QueryRow(ctx, `SELECT id, vehicle_id, date, value, unit, created_at FROM fleet_vehicle_odometers WHERE vehicle_id = $1 ORDER BY value DESC, id DESC LIMIT 1`, vehicleID).
		Scan(&v.ID, &v.VehicleID, &v.Date, &v.Value, &v.Unit, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) { return nil, nil }
	if err != nil { return nil, platformerrors.Internal("failed to get latest fleet odometer", err) }
	return &v, nil
}

// ── Service logs ────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateLogService(ctx context.Context, v *fleet.VehicleLogService) error {
	query := `
		INSERT INTO fleet_vehicle_log_services (
			vehicle_id, description, date, amount, vendor_id, service_type_id, state, inv_ref,
			odometer, notes, company_id, created_at, updated_at, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW(), $12, $12)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		v.VehicleID, v.Description, v.Date, v.Amount, nullableInt(v.VendorID),
		nullableInt(v.ServiceTypeID), v.State, nullableString(v.InvRef), nullableFloat(v.Odometer),
		nullableString(v.Notes), v.CompanyID, auditCreatedBy(ctx),
	).Scan(&v.ID, &v.Audit.CreatedAt, &v.Audit.UpdatedAt)
}
func (r *PostgresRepo) GetLogServiceByID(ctx context.Context, companyID, id int64) (*fleet.VehicleLogService, error) {
	query := `SELECT ` + fleetServiceColumns + ` FROM fleet_vehicle_log_services WHERE id = $1 AND company_id = $2`
	var v fleet.VehicleLogService
	if err := r.pool.QueryRow(ctx, query, id, companyID).Scan(
		&v.ID, &v.VehicleID, &v.Description, &v.Date, &v.Amount, &v.VendorID, &v.ServiceTypeID, &v.State,
		&v.InvRef, &v.Odometer, &v.Notes, &v.CompanyID,
		&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("fleet service log", id) }
		return nil, platformerrors.Internal("failed to get fleet service log", err)
	}
	return &v, nil
}
func (r *PostgresRepo) UpdateLogService(ctx context.Context, v *fleet.VehicleLogService) error {
	query := `
		UPDATE fleet_vehicle_log_services SET
			vehicle_id = $1, description = $2, date = $3, amount = $4, vendor_id = $5,
			service_type_id = $6, state = $7, inv_ref = $8, odometer = $9, notes = $10,
			updated_at = NOW(), updated_by = $11
		WHERE id = $12 RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		v.VehicleID, v.Description, v.Date, v.Amount, nullableInt(v.VendorID),
		nullableInt(v.ServiceTypeID), v.State, nullableString(v.InvRef), nullableFloat(v.Odometer),
		nullableString(v.Notes), auditCreatedBy(ctx), v.ID,
	).Scan(&v.Audit.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return notFound("fleet service log", v.ID) }
		return platformerrors.Internal("failed to update fleet service log", err)
	}
	return nil
}
func (r *PostgresRepo) DeleteLogService(ctx context.Context, companyID, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM fleet_vehicle_log_services WHERE id = $1 AND company_id = $2`, id, companyID)
	if err != nil { return platformerrors.Internal("failed to delete fleet service log", err) }
	if tag.RowsAffected() == 0 { return notFound("fleet service log", id) }
	return nil
}
func (r *PostgresRepo) ListLogServices(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[fleet.VehicleLogService], error) {
	criteria := []filter.Criterion{{Field: "company_id", Operator: filter.OpEqual, Value: companyID}}
	if f != nil { criteria = append(criteria, f.Criteria...) }
	combined := &filter.Filter{Criteria: criteria}

	whereClause, args, nextIdx, err := combined.BuildWhereClause(allowedServiceFilterFields, 1)
	if err != nil {
		return pagination.PageResult[fleet.VehicleLogService]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}
	var totalItems int64
	if err := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM fleet_vehicle_log_services %s`, whereClause), args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[fleet.VehicleLogService]{}, platformerrors.Internal("failed to count fleet service logs", err)
	}
	orderDir := page.OrderDirection()
	sortBy := "date"
	if page.SortBy != "" {
		if col, ok := allowedServiceFilterFields[page.SortBy]; ok { sortBy = col }
	}
	limit := page.LimitClamped()
	offset := page.Offset()
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`SELECT %s FROM fleet_vehicle_log_services %s ORDER BY %s %s, id LIMIT $%d OFFSET $%d`,
		fleetServiceColumns, whereClause, sortBy, orderDir, nextIdx, nextIdx+1), append(args, limit, offset)...)
	if err != nil { return pagination.PageResult[fleet.VehicleLogService]{}, platformerrors.Internal("failed to list fleet service logs", err) }
	defer rows.Close()

	var list []fleet.VehicleLogService
	for rows.Next() {
		var v fleet.VehicleLogService
		if err := rows.Scan(
			&v.ID, &v.VehicleID, &v.Description, &v.Date, &v.Amount, &v.VendorID, &v.ServiceTypeID, &v.State,
			&v.InvRef, &v.Odometer, &v.Notes, &v.CompanyID,
			&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy,
		); err != nil {
			return pagination.PageResult[fleet.VehicleLogService]{}, platformerrors.Internal("failed to scan fleet service log", err)
		}
		list = append(list, v)
	}
	if err := rows.Err(); err != nil { return pagination.PageResult[fleet.VehicleLogService]{}, platformerrors.Internal("failed to iterate fleet service logs", err) }
	return pagination.NewPageResult(list, totalItems, page), nil
}

// ── Contracts ───────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateLogContract(ctx context.Context, v *fleet.VehicleLogContract) error {
	query := `
		INSERT INTO fleet_vehicle_log_contracts (
			vehicle_id, name, user_id, date, start_date, expiration_date, cost_generated,
			cost_frequency, ins_ref, insurer_id, state, notes, company_id, created_at,
			updated_at, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW(), $14, $14)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		v.VehicleID, nullableString(v.Name), nullableInt(v.UserID), nullableTime(v.Date),
		v.StartDate, nullableTime(v.ExpirationDate), v.CostGenerated, v.CostFrequency,
		nullableString(v.InsRef), nullableInt(v.InsurerID), v.State, nullableString(v.Notes),
		v.CompanyID, auditCreatedBy(ctx),
	).Scan(&v.ID, &v.Audit.CreatedAt, &v.Audit.UpdatedAt)
}
func (r *PostgresRepo) GetLogContractByID(ctx context.Context, companyID, id int64) (*fleet.VehicleLogContract, error) {
	query := `SELECT ` + fleetContractColumns + ` FROM fleet_vehicle_log_contracts WHERE id = $1 AND company_id = $2`
	var v fleet.VehicleLogContract
	if err := r.pool.QueryRow(ctx, query, id, companyID).Scan(
		&v.ID, &v.VehicleID, &v.Name, &v.UserID, &v.Date, &v.StartDate, &v.ExpirationDate, &v.CostGenerated,
		&v.CostFrequency, &v.InsRef, &v.InsurerID, &v.State, &v.Notes, &v.CompanyID,
		&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, notFound("fleet contract", id) }
		return nil, platformerrors.Internal("failed to get fleet contract", err)
	}
	return &v, nil
}
func (r *PostgresRepo) UpdateLogContract(ctx context.Context, v *fleet.VehicleLogContract) error {
	query := `
		UPDATE fleet_vehicle_log_contracts SET
			vehicle_id = $1, name = $2, user_id = $3, date = $4, start_date = $5,
			expiration_date = $6, cost_generated = $7, cost_frequency = $8, ins_ref = $9,
			insurer_id = $10, state = $11, notes = $12, updated_at = NOW(), updated_by = $13
		WHERE id = $14 RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		v.VehicleID, nullableString(v.Name), nullableInt(v.UserID), nullableTime(v.Date),
		v.StartDate, nullableTime(v.ExpirationDate), v.CostGenerated, v.CostFrequency,
		nullableString(v.InsRef), nullableInt(v.InsurerID), v.State, nullableString(v.Notes),
		auditCreatedBy(ctx), v.ID,
	).Scan(&v.Audit.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return notFound("fleet contract", v.ID) }
		return platformerrors.Internal("failed to update fleet contract", err)
	}
	return nil
}
func (r *PostgresRepo) DeleteLogContract(ctx context.Context, companyID, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM fleet_vehicle_log_contracts WHERE id = $1 AND company_id = $2`, id, companyID)
	if err != nil { return platformerrors.Internal("failed to delete fleet contract", err) }
	if tag.RowsAffected() == 0 { return notFound("fleet contract", id) }
	return nil
}
func (r *PostgresRepo) ListLogContracts(ctx context.Context, companyID int64, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[fleet.VehicleLogContract], error) {
	criteria := []filter.Criterion{{Field: "company_id", Operator: filter.OpEqual, Value: companyID}}
	if f != nil { criteria = append(criteria, f.Criteria...) }
	combined := &filter.Filter{Criteria: criteria}

	whereClause, args, nextIdx, err := combined.BuildWhereClause(allowedContractFilterFields, 1)
	if err != nil {
		return pagination.PageResult[fleet.VehicleLogContract]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}
	var totalItems int64
	if err := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM fleet_vehicle_log_contracts %s`, whereClause), args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[fleet.VehicleLogContract]{}, platformerrors.Internal("failed to count fleet contracts", err)
	}
	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedContractFilterFields[page.SortBy]; ok { sortBy = col }
	}
	limit := page.LimitClamped()
	offset := page.Offset()
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`SELECT %s FROM fleet_vehicle_log_contracts %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		fleetContractColumns, whereClause, sortBy, orderDir, nextIdx, nextIdx+1), append(args, limit, offset)...)
	if err != nil { return pagination.PageResult[fleet.VehicleLogContract]{}, platformerrors.Internal("failed to list fleet contracts", err) }
	defer rows.Close()

	var list []fleet.VehicleLogContract
	for rows.Next() {
		var v fleet.VehicleLogContract
		if err := rows.Scan(
			&v.ID, &v.VehicleID, &v.Name, &v.UserID, &v.Date, &v.StartDate, &v.ExpirationDate, &v.CostGenerated,
			&v.CostFrequency, &v.InsRef, &v.InsurerID, &v.State, &v.Notes, &v.CompanyID,
			&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy,
		); err != nil {
			return pagination.PageResult[fleet.VehicleLogContract]{}, platformerrors.Internal("failed to scan fleet contract", err)
		}
		list = append(list, v)
	}
	if err := rows.Err(); err != nil { return pagination.PageResult[fleet.VehicleLogContract]{}, platformerrors.Internal("failed to iterate fleet contracts", err) }
	return pagination.NewPageResult(list, totalItems, page), nil
}
func (r *PostgresRepo) ListContractsForRefresh(ctx context.Context) ([]fleet.VehicleLogContract, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+fleetContractColumns+` FROM fleet_vehicle_log_contracts WHERE state = 'open' ORDER BY id`)
	if err != nil { return nil, platformerrors.Internal("failed to list fleet contracts for refresh", err) }
	defer rows.Close()
	var list []fleet.VehicleLogContract
	for rows.Next() {
		var v fleet.VehicleLogContract
		if err := rows.Scan(
			&v.ID, &v.VehicleID, &v.Name, &v.UserID, &v.Date, &v.StartDate, &v.ExpirationDate, &v.CostGenerated,
			&v.CostFrequency, &v.InsRef, &v.InsurerID, &v.State, &v.Notes, &v.CompanyID,
			&v.Audit.CreatedAt, &v.Audit.UpdatedAt, &v.Audit.CreatedBy, &v.Audit.UpdatedBy,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan fleet contract", err)
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// ── Reports ─────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CostByVehicle(ctx context.Context, companyID int64) ([]fleet.VehicleCost, error) {
	query := `
		SELECT v.id, COALESCE(v.name, ''), v.license_plate,
		       COALESCE(SUM(s.amount), 0), COUNT(s.id),
		       MAX(s.date)::text
		FROM fleet_vehicles v
		LEFT JOIN fleet_vehicle_log_services s ON s.vehicle_id = v.id AND s.company_id = v.company_id
		WHERE v.company_id = $1
		GROUP BY v.id, v.name, v.license_plate
		ORDER BY v.id
	`
	rows, err := r.pool.Query(ctx, query, companyID)
	if err != nil { return nil, platformerrors.Internal("failed to compute fleet cost report", err) }
	defer rows.Close()
	var list []fleet.VehicleCost
	for rows.Next() {
		var v fleet.VehicleCost
		var last *string
		if err := rows.Scan(&v.VehicleID, &v.VehicleName, &v.LicensePlate, &v.TotalAmount, &v.ServiceCount, &last); err != nil {
			return nil, platformerrors.Internal("failed to scan fleet cost report", err)
		}
		if last != nil && *last != "" { v.LastServiceDate = last }
		list = append(list, v)
	}
	return list, rows.Err()
}

// ── helpers ─────────────────────────────────────────────────────────────────

func nullableInt(ptr *int64) any { if ptr == nil { return nil }; return *ptr }
func nullableTime(ptr *time.Time) any { if ptr == nil { return nil }; return *ptr }
func nullableFloat(ptr *float64) any { if ptr == nil { return nil }; return *ptr }
func nullableString(s string) any { if s == "" { return nil }; return s }

func auditCreatedBy(ctx context.Context) any {
	if uid := audit.UserIDFromContext(ctx); uid != nil { return *uid }
	return nil
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}