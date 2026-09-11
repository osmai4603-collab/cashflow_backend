package mrpstorage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cashflow_backend/internal/domain/mrp"
	platformerrors "cashflow_backend/internal/platform/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// ─────────────────────────────────────────────────────────────────────────────
// Workcenters
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateWorkcenter(ctx context.Context, wc *mrp.Workcenter) error {
	query := `
		INSERT INTO mrp_workcenters (
			name, code, active, sequence, company_id,
			time_start, time_stop, time_efficiency, capacity,
			cost_per_hour, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		wc.Name, wc.Code, wc.Active, wc.Sequence, wc.CompanyID,
		wc.TimeStart, wc.TimeStop, wc.TimeEfficiency, wc.Capacity,
		wc.CostPerHour, wc.Audit.CreatedAt, wc.Audit.UpdatedAt, wc.Audit.CreatedBy, wc.Audit.UpdatedBy,
	).Scan(&wc.ID)

	if err != nil {
		return platformerrors.Internal("failed to create workcenter", err)
	}
	return nil
}

func (r *PostgresRepo) UpdateWorkcenter(ctx context.Context, wc *mrp.Workcenter) error {
	query := `
		UPDATE mrp_workcenters SET
			name = $1, code = $2, active = $3, sequence = $4,
			time_start = $5, time_stop = $6, time_efficiency = $7,
			capacity = $8, cost_per_hour = $9, updated_at = $10, updated_by = $11
		WHERE id = $12
	`
	tag, err := r.pool.Exec(ctx, query,
		wc.Name, wc.Code, wc.Active, wc.Sequence,
		wc.TimeStart, wc.TimeStop, wc.TimeEfficiency,
		wc.Capacity, wc.CostPerHour, wc.Audit.UpdatedAt, wc.Audit.UpdatedBy, wc.ID,
	)
	if err != nil {
		return platformerrors.Internal("failed to update workcenter", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound("workcenter not found", nil)
	}
	return nil
}

func (r *PostgresRepo) GetWorkcenterByID(ctx context.Context, id int64) (*mrp.Workcenter, error) {
	query := `
		SELECT
			id, name, code, active, sequence, company_id,
			time_start, time_stop, time_efficiency, capacity, cost_per_hour,
			created_at, updated_at, created_by, updated_by
		FROM mrp_workcenters WHERE id = $1
	`
	wc := &mrp.Workcenter{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&wc.ID, &wc.Name, &wc.Code, &wc.Active, &wc.Sequence, &wc.CompanyID,
		&wc.TimeStart, &wc.TimeStop, &wc.TimeEfficiency, &wc.Capacity, &wc.CostPerHour,
		&wc.Audit.CreatedAt, &wc.Audit.UpdatedAt, &wc.Audit.CreatedBy, &wc.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("workcenter not found", nil)
		}
		return nil, platformerrors.Internal("failed to get workcenter", err)
	}
	return wc, nil
}

func (r *PostgresRepo) ListWorkcenters(ctx context.Context, f mrp.WorkcenterFilter) ([]*mrp.Workcenter, int64, error) {
	where := []string{"1=1"}
	args := []any{}
	argPos := 1

	if f.CompanyID != nil {
		where = append(where, fmt.Sprintf("company_id = $%d", argPos))
		args = append(args, *f.CompanyID)
		argPos++
	}
	if f.Active != nil {
		where = append(where, fmt.Sprintf("active = $%d", argPos))
		args = append(args, *f.Active)
		argPos++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR code ILIKE $%d)", argPos, argPos))
		args = append(args, "%"+f.Search+"%")
		argPos++
	}

	query := fmt.Sprintf(`
		SELECT
			id, name, code, active, sequence, company_id,
			time_start, time_stop, time_efficiency, capacity, cost_per_hour,
			created_at, updated_at, created_by, updated_by
		FROM mrp_workcenters
		WHERE %s
		ORDER BY sequence, name
		LIMIT $%d OFFSET $%d
	`, strings.Join(where, " AND "), argPos, argPos+1)

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM mrp_workcenters WHERE %s", strings.Join(where, " AND "))

	var total int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, platformerrors.Internal("failed to count workcenters", err)
	}

	args = append(args, f.Limit, f.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, platformerrors.Internal("failed to list workcenters", err)
	}
	defer rows.Close()

	var res []*mrp.Workcenter
	for rows.Next() {
		wc := &mrp.Workcenter{}
		err := rows.Scan(
			&wc.ID, &wc.Name, &wc.Code, &wc.Active, &wc.Sequence, &wc.CompanyID,
			&wc.TimeStart, &wc.TimeStop, &wc.TimeEfficiency, &wc.Capacity, &wc.CostPerHour,
			&wc.Audit.CreatedAt, &wc.Audit.UpdatedAt, &wc.Audit.CreatedBy, &wc.Audit.UpdatedBy,
		)
		if err != nil {
			return nil, 0, platformerrors.Internal("failed to scan workcenter", err)
		}
		res = append(res, wc)
	}
	return res, total, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Workorders
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateWorkorder(ctx context.Context, wo *mrp.Workorder) error {
	query := `
		INSERT INTO mrp_workorders (
			production_id, workcenter_id, operation_id, name, sequence, state,
			duration_expected, duration, date_start, date_finished,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		wo.ProductionID, wo.WorkcenterID, wo.OperationID, wo.Name, wo.Sequence, string(wo.State),
		wo.DurationExpected, wo.Duration, wo.DateStart, wo.DateFinished,
		wo.Audit.CreatedAt, wo.Audit.UpdatedAt, wo.Audit.CreatedBy, wo.Audit.UpdatedBy,
	).Scan(&wo.ID)
	if err != nil {
		return platformerrors.Internal("failed to create workorder", err)
	}
	return nil
}

func (r *PostgresRepo) UpdateWorkorder(ctx context.Context, wo *mrp.Workorder) error {
	query := `
		UPDATE mrp_workorders SET
			state = $1, duration = $2, date_start = $3, date_finished = $4,
			updated_at = $5, updated_by = $6
		WHERE id = $7
	`
	tag, err := r.pool.Exec(ctx, query,
		string(wo.State), wo.Duration, wo.DateStart, wo.DateFinished,
		wo.Audit.UpdatedAt, wo.Audit.UpdatedBy, wo.ID,
	)
	if err != nil {
		return platformerrors.Internal("failed to update workorder", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound("workorder not found", nil)
	}
	return nil
}

func (r *PostgresRepo) GetWorkorderByID(ctx context.Context, id int64) (*mrp.Workorder, error) {
	query := `
		SELECT
			id, production_id, workcenter_id, operation_id, name, sequence, state,
			duration_expected, duration, date_start, date_finished,
			created_at, updated_at, created_by, updated_by
		FROM mrp_workorders WHERE id = $1
	`
	wo := &mrp.Workorder{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&wo.ID, &wo.ProductionID, &wo.WorkcenterID, &wo.OperationID, &wo.Name, &wo.Sequence, (*string)(&wo.State),
		&wo.DurationExpected, &wo.Duration, &wo.DateStart, &wo.DateFinished,
		&wo.Audit.CreatedAt, &wo.Audit.UpdatedAt, &wo.Audit.CreatedBy, &wo.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("workorder not found", nil)
		}
		return nil, platformerrors.Internal("failed to get workorder", err)
	}
	return wo, nil
}

func (r *PostgresRepo) ListWorkorders(ctx context.Context, productionID int64) ([]*mrp.Workorder, error) {
	query := `
		SELECT
			id, production_id, workcenter_id, operation_id, name, sequence, state,
			duration_expected, duration, date_start, date_finished,
			created_at, updated_at, created_by, updated_by
		FROM mrp_workorders WHERE production_id = $1 ORDER BY sequence
	`
	rows, err := r.pool.Query(ctx, query, productionID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list workorders", err)
	}
	defer rows.Close()

	var res []*mrp.Workorder
	for rows.Next() {
		wo := &mrp.Workorder{}
		err := rows.Scan(
			&wo.ID, &wo.ProductionID, &wo.WorkcenterID, &wo.OperationID, &wo.Name, &wo.Sequence, (*string)(&wo.State),
			&wo.DurationExpected, &wo.Duration, &wo.DateStart, &wo.DateFinished,
			&wo.Audit.CreatedAt, &wo.Audit.UpdatedAt, &wo.Audit.CreatedBy, &wo.Audit.UpdatedBy,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, wo)
	}
	return res, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Unbuild Orders
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateUnbuild(ctx context.Context, uo *mrp.UnbuildOrder) error {
	query := `
		INSERT INTO mrp_unbuilds (
			name, product_id, bom_id, mo_id, quantity, uom_id,
			location_id, dest_location_id, state, company_id,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		uo.Name, uo.ProductID, uo.BomID, uo.MOID, uo.Quantity, uo.UoMID,
		uo.LocationID, uo.DestLocationID, string(uo.State), uo.CompanyID,
		uo.Audit.CreatedAt, uo.Audit.UpdatedAt, uo.Audit.CreatedBy, uo.Audit.UpdatedBy,
	).Scan(&uo.ID)
	if err != nil {
		return platformerrors.Internal("failed to create unbuild order", err)
	}
	return nil
}

func (r *PostgresRepo) GetUnbuildByID(ctx context.Context, id int64) (*mrp.UnbuildOrder, error) {
	query := `
		SELECT
			id, name, product_id, bom_id, mo_id, quantity, uom_id,
			location_id, dest_location_id, state, company_id,
			created_at, updated_at, created_by, updated_by
		FROM mrp_unbuilds WHERE id = $1
	`
	uo := &mrp.UnbuildOrder{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&uo.ID, &uo.Name, &uo.ProductID, &uo.BomID, &uo.MOID, &uo.Quantity, &uo.UoMID,
		&uo.LocationID, &uo.DestLocationID, (*string)(&uo.State), &uo.CompanyID,
		&uo.Audit.CreatedAt, &uo.Audit.UpdatedAt, &uo.Audit.CreatedBy, &uo.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("unbuild order not found", nil)
		}
		return nil, platformerrors.Internal("failed to get unbuild order", err)
	}
	return uo, nil
}

func (r *PostgresRepo) CreateWorkorderTimeLog(ctx context.Context, log *mrp.WorkorderTimeLog) error {
	return r.pool.QueryRow(ctx, `INSERT INTO mrp_workorder_time_logs (workorder_id, user_id, date_start, date_end, duration, loss_id) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`, log.WorkorderID, log.UserID, log.DateStart, log.DateEnd, log.Duration, log.LossID).Scan(&log.ID)
}

func (r *PostgresRepo) ListWorkorderTimeLogs(ctx context.Context, workorderID int64) ([]mrp.WorkorderTimeLog, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, workorder_id, user_id, date_start, date_end, duration, loss_id FROM mrp_workorder_time_logs WHERE workorder_id=$1 ORDER BY date_start`, workorderID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list workorder time logs", err)
	}
	defer rows.Close()
	result := make([]mrp.WorkorderTimeLog, 0)
	for rows.Next() {
		var log mrp.WorkorderTimeLog
		if err := rows.Scan(&log.ID, &log.WorkorderID, &log.UserID, &log.DateStart, &log.DateEnd, &log.Duration, &log.LossID); err != nil {
			return nil, platformerrors.Internal("failed to scan workorder time log", err)
		}
		result = append(result, log)
	}
	return result, rows.Err()
}

func (r *PostgresRepo) CreateWorkcenterCalendar(ctx context.Context, calendar *mrp.WorkcenterCalendar) error {
	return r.pool.QueryRow(ctx, `INSERT INTO mrp_workcenter_calendars (workcenter_id, day_of_week, hour_from, hour_to, attendance_type, company_id) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`, calendar.WorkcenterID, calendar.DayOfWeek, calendar.HourFrom, calendar.HourTo, calendar.AttendanceType, calendar.CompanyID).Scan(&calendar.ID)
}

func (r *PostgresRepo) ListWorkcenterCalendars(ctx context.Context, workcenterID int64) ([]mrp.WorkcenterCalendar, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, workcenter_id, day_of_week, hour_from, hour_to, attendance_type, company_id FROM mrp_workcenter_calendars WHERE workcenter_id=$1 ORDER BY day_of_week, hour_from`, workcenterID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list workcenter calendars", err)
	}
	defer rows.Close()
	result := make([]mrp.WorkcenterCalendar, 0)
	for rows.Next() {
		var calendar mrp.WorkcenterCalendar
		if err := rows.Scan(&calendar.ID, &calendar.WorkcenterID, &calendar.DayOfWeek, &calendar.HourFrom, &calendar.HourTo, &calendar.AttendanceType, &calendar.CompanyID); err != nil {
			return nil, platformerrors.Internal("failed to scan workcenter calendar", err)
		}
		result = append(result, calendar)
	}
	return result, rows.Err()
}

func (r *PostgresRepo) DeleteWorkcenter(ctx context.Context, id int64) error {
	query := `DELETE FROM mrp_workcenters WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete workcenter", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound("workcenter not found", nil)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Bill of Materials (BoM)
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateBoM(ctx context.Context, bom *mrp.BillOfMaterials) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return platformerrors.Internal("failed to start transaction", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO mrp_boms (
			code, product_id, product_qty, uom_id, type,
			ready_to_produce, consumption, active, company_id,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) RETURNING id
	`
	err = tx.QueryRow(ctx, query,
		bom.Code, bom.ProductID, bom.ProductQty, bom.UoMID, string(bom.Type),
		bom.ReadyToProduce, bom.Consumption, bom.Active, bom.CompanyID,
		bom.Audit.CreatedAt, bom.Audit.UpdatedAt, bom.Audit.CreatedBy, bom.Audit.UpdatedBy,
	).Scan(&bom.ID)
	if err != nil {
		return platformerrors.Internal("failed to create bom", err)
	}

	for i := range bom.Lines {
		bom.Lines[i].BomID = bom.ID
		if err := r.addBomLineTx(ctx, tx, &bom.Lines[i]); err != nil {
			return err
		}
	}

	for i := range bom.Operations {
		bom.Operations[i].BomID = bom.ID
		if err := r.addRoutingOperationTx(ctx, tx, &bom.Operations[i]); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepo) GetBoMByID(ctx context.Context, id int64) (*mrp.BillOfMaterials, error) {
	query := `
		SELECT
			id, code, product_id, product_qty, uom_id, type,
			ready_to_produce, consumption, active, company_id,
			created_at, updated_at, created_by, updated_by
		FROM mrp_boms WHERE id = $1
	`
	bom := &mrp.BillOfMaterials{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&bom.ID, &bom.Code, &bom.ProductID, &bom.ProductQty, &bom.UoMID, (*string)(&bom.Type),
		&bom.ReadyToProduce, &bom.Consumption, &bom.Active, &bom.CompanyID,
		&bom.Audit.CreatedAt, &bom.Audit.UpdatedAt, &bom.Audit.CreatedBy, &bom.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("bom not found", nil)
		}
		return nil, platformerrors.Internal("failed to get bom", err)
	}

	// Load Lines
	linesQuery := `SELECT id, bom_id, product_id, quantity, uom_id, operation_id, sequence FROM mrp_bom_lines WHERE bom_id = $1 ORDER BY sequence`
	rows, err := r.pool.Query(ctx, linesQuery, id)
	if err != nil {
		return nil, platformerrors.Internal("failed to load bom lines", err)
	}
	defer rows.Close()
	for rows.Next() {
		var l mrp.BomLine
		if err := rows.Scan(&l.ID, &l.BomID, &l.ProductID, &l.Quantity, &l.UoMID, &l.OperationID, &l.Sequence); err != nil {
			return nil, err
		}
		bom.Lines = append(bom.Lines, l)
	}

	// Load Operations
	opsQuery := `SELECT id, bom_id, workcenter_id, name, sequence, time_mode, time_cycle_manual FROM mrp_routing_operations WHERE bom_id = $1 ORDER BY sequence`
	opRows, err := r.pool.Query(ctx, opsQuery, id)
	if err != nil {
		return nil, platformerrors.Internal("failed to load routing operations", err)
	}
	defer opRows.Close()
	for opRows.Next() {
		var op mrp.RoutingOperation
		if err := opRows.Scan(&op.ID, &op.BomID, &op.WorkcenterID, &op.Name, &op.Sequence, &op.TimeMode, &op.TimeCycleManual); err != nil {
			return nil, err
		}
		bom.Operations = append(bom.Operations, op)
	}

	return bom, nil
}

func (r *PostgresRepo) UpdateBoM(ctx context.Context, bom *mrp.BillOfMaterials) error {
	query := `
		UPDATE mrp_boms SET
			code = $1, product_id = $2, product_qty = $3, uom_id = $4, type = $5,
			ready_to_produce = $6, consumption = $7, active = $8, updated_at = $9, updated_by = $10
		WHERE id = $11
	`
	tag, err := r.pool.Exec(ctx, query,
		bom.Code, bom.ProductID, bom.ProductQty, bom.UoMID, string(bom.Type),
		bom.ReadyToProduce, bom.Consumption, bom.Active, bom.Audit.UpdatedAt, bom.Audit.UpdatedBy, bom.ID,
	)
	if err != nil {
		return platformerrors.Internal("failed to update bom", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound("bom not found", nil)
	}
	return nil
}

func (r *PostgresRepo) ListBoMs(ctx context.Context, f mrp.BoMFilter) ([]*mrp.BillOfMaterials, int64, error) {
	where := []string{"1=1"}
	args := []any{}
	argPos := 1

	if f.ProductID != nil {
		where = append(where, fmt.Sprintf("product_id = $%d", argPos))
		args = append(args, *f.ProductID)
		argPos++
	}
	if f.CompanyID != nil {
		where = append(where, fmt.Sprintf("company_id = $%d", argPos))
		args = append(args, *f.CompanyID)
		argPos++
	}
	if f.Active != nil {
		where = append(where, fmt.Sprintf("active = $%d", argPos))
		args = append(args, *f.Active)
		argPos++
	}
	if f.Type != "" {
		where = append(where, fmt.Sprintf("type = $%d", argPos))
		args = append(args, string(f.Type))
		argPos++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM mrp_boms WHERE %s", strings.Join(where, " AND "))
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT
			id, code, product_id, product_qty, uom_id, type,
			ready_to_produce, consumption, active, company_id,
			created_at, updated_at, created_by, updated_by
		FROM mrp_boms
		WHERE %s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, strings.Join(where, " AND "), argPos, argPos+1)

	args = append(args, f.Limit, f.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var res []*mrp.BillOfMaterials
	for rows.Next() {
		bom := &mrp.BillOfMaterials{}
		err := rows.Scan(
			&bom.ID, &bom.Code, &bom.ProductID, &bom.ProductQty, &bom.UoMID, (*string)(&bom.Type),
			&bom.ReadyToProduce, &bom.Consumption, &bom.Active, &bom.CompanyID,
			&bom.Audit.CreatedAt, &bom.Audit.UpdatedAt, &bom.Audit.CreatedBy, &bom.Audit.UpdatedBy,
		)
		if err != nil {
			return nil, 0, err
		}
		res = append(res, bom)
	}
	return res, total, nil
}

func (r *PostgresRepo) DeleteBoM(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM mrp_boms WHERE id = $1", id)
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers for Lines & Operations
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) AddBomLine(ctx context.Context, line *mrp.BomLine) error {
	return r.addBomLineTx(ctx, r.pool, line)
}

func (r *PostgresRepo) addBomLineTx(ctx context.Context, getter interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, line *mrp.BomLine) error {
	query := `
		INSERT INTO mrp_bom_lines (bom_id, product_id, quantity, uom_id, operation_id, sequence)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`
	return getter.QueryRow(ctx, query, line.BomID, line.ProductID, line.Quantity, line.UoMID, line.OperationID, line.Sequence).Scan(&line.ID)
}

func (r *PostgresRepo) RemoveBomLine(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM mrp_bom_lines WHERE id = $1", id)
	return err
}

func (r *PostgresRepo) AddRoutingOperation(ctx context.Context, op *mrp.RoutingOperation) error {
	return r.addRoutingOperationTx(ctx, r.pool, op)
}

func (r *PostgresRepo) addRoutingOperationTx(ctx context.Context, getter interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, op *mrp.RoutingOperation) error {
	query := `
		INSERT INTO mrp_routing_operations (bom_id, workcenter_id, name, sequence, time_mode, time_cycle_manual)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`
	return getter.QueryRow(ctx, query, op.BomID, op.WorkcenterID, op.Name, op.Sequence, op.TimeMode, op.TimeCycleManual).Scan(&op.ID)
}

func (r *PostgresRepo) RemoveRoutingOperation(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM mrp_routing_operations WHERE id = $1", id)
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// Production Orders (MO)
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateProduction(ctx context.Context, mo *mrp.ProductionOrder) error {
	query := `
		INSERT INTO mrp_productions (
			name, priority, backorder_sequence, origin,
			product_id, product_qty, uom_id, qty_producing, qty_produced,
			bom_id, picking_type_id, location_src_id, location_dest_id,
			date_deadline, date_start, state, reservation_state, company_id,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		) RETURNING id
	`
	err := r.pool.QueryRow(ctx, query,
		mo.Name, mo.Priority, mo.BackorderSeq, mo.Origin,
		mo.ProductID, mo.ProductQty, mo.UoMID, mo.QtyProducing, mo.QtyProduced,
		mo.BomID, mo.PickingTypeID, mo.LocationSrcID, mo.LocationDestID,
		mo.DateDeadline, mo.DateStart, string(mo.State), string(mo.ReservationState), mo.CompanyID,
		mo.Audit.CreatedAt, mo.Audit.UpdatedAt, mo.Audit.CreatedBy, mo.Audit.UpdatedBy,
	).Scan(&mo.ID)
	if err != nil {
		return platformerrors.Internal("failed to create production order", err)
	}
	return nil
}

func (r *PostgresRepo) UpdateProduction(ctx context.Context, mo *mrp.ProductionOrder) error {
	query := `
		UPDATE mrp_productions SET
			priority = $1, qty_producing = $2, qty_produced = $3,
			date_deadline = $4, date_finished = $5, state = $6,
			reservation_state = $7, updated_at = $8, updated_by = $9
		WHERE id = $10
	`
	tag, err := r.pool.Exec(ctx, query,
		mo.Priority, mo.QtyProducing, mo.QtyProduced,
		mo.DateDeadline, mo.DateFinished, string(mo.State),
		string(mo.ReservationState), mo.Audit.UpdatedAt, mo.Audit.UpdatedBy, mo.ID,
	)
	if err != nil {
		return platformerrors.Internal("failed to update production order", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound("production order not found", nil)
	}
	return nil
}

func (r *PostgresRepo) GetProductionByID(ctx context.Context, id int64) (*mrp.ProductionOrder, error) {
	query := `
		SELECT
			id, name, priority, backorder_sequence, origin,
			product_id, product_qty, uom_id, qty_producing, qty_produced,
			bom_id, picking_type_id, location_src_id, location_dest_id,
			date_deadline, date_start, date_finished, state, reservation_state, company_id,
			created_at, updated_at, created_by, updated_by
		FROM mrp_productions WHERE id = $1
	`
	mo := &mrp.ProductionOrder{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&mo.ID, &mo.Name, &mo.Priority, &mo.BackorderSeq, &mo.Origin,
		&mo.ProductID, &mo.ProductQty, &mo.UoMID, &mo.QtyProducing, &mo.QtyProduced,
		&mo.BomID, &mo.PickingTypeID, &mo.LocationSrcID, &mo.LocationDestID,
		&mo.DateDeadline, &mo.DateStart, &mo.DateFinished, (*string)(&mo.State), (*string)(&mo.ReservationState), &mo.CompanyID,
		&mo.Audit.CreatedAt, &mo.Audit.UpdatedAt, &mo.Audit.CreatedBy, &mo.Audit.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("production order not found", nil)
		}
		return nil, platformerrors.Internal("failed to get production order", err)
	}
	return mo, nil
}

func (r *PostgresRepo) ListProductions(ctx context.Context, f mrp.ProductionFilter) ([]*mrp.ProductionOrder, int64, error) {
	where := []string{"1=1"}
	args := []any{}
	argPos := 1

	if f.ProductID != nil {
		where = append(where, fmt.Sprintf("product_id = $%d", argPos))
		args = append(args, *f.ProductID)
		argPos++
	}
	if f.State != "" {
		where = append(where, fmt.Sprintf("state = $%d", argPos))
		args = append(args, string(f.State))
		argPos++
	}
	if f.CompanyID != nil {
		where = append(where, fmt.Sprintf("company_id = $%d", argPos))
		args = append(args, *f.CompanyID)
		argPos++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM mrp_productions WHERE %s", strings.Join(where, " AND "))
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT
			id, name, priority, backorder_sequence, origin,
			product_id, product_qty, uom_id, qty_producing, qty_produced,
			bom_id, picking_type_id, location_src_id, location_dest_id,
			date_deadline, date_start, date_finished, state, reservation_state, company_id,
			created_at, updated_at, created_by, updated_by
		FROM mrp_productions
		WHERE %s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, strings.Join(where, " AND "), argPos, argPos+1)

	args = append(args, f.Limit, f.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var res []*mrp.ProductionOrder
	for rows.Next() {
		mo := &mrp.ProductionOrder{}
		err := rows.Scan(
			&mo.ID, &mo.Name, &mo.Priority, &mo.BackorderSeq, &mo.Origin,
			&mo.ProductID, &mo.ProductQty, &mo.UoMID, &mo.QtyProducing, &mo.QtyProduced,
			&mo.BomID, &mo.PickingTypeID, &mo.LocationSrcID, &mo.LocationDestID,
			&mo.DateDeadline, &mo.DateStart, &mo.DateFinished, (*string)(&mo.State), (*string)(&mo.ReservationState), &mo.CompanyID,
			&mo.Audit.CreatedAt, &mo.Audit.UpdatedAt, &mo.Audit.CreatedBy, &mo.Audit.UpdatedBy,
		)
		if err != nil {
			return nil, 0, err
		}
		res = append(res, mo)
	}
	return res, total, nil
}
