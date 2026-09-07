package stockstorage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cashflow_backend/internal/domain/stock"
	"cashflow_backend/internal/platform/database"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedLocationFilterFields = map[string]string{
	"usage":  "usage",
	"name":   "name",
	"active": "active",
}

var allowedWarehouseFilterFields = map[string]string{
	"code":   "code",
	"name":   "name",
	"active": "active",
}

var allowedPickingFilterFields = map[string]string{
	"picking_type":         "picking_type",
	"state":                "state",
	"partner_id":           "partner_id",
	"origin":               "origin",
	"procurement_group_id": "procurement_group_id",
	"active":               "active",
}

var allowedMoveFilterFields = map[string]string{
	"product_id":           "product_id",
	"picking_id":           "picking_id",
	"procurement_group_id": "procurement_group_id",
	"state":                "state",
}

// PostgresRepo implements stock.Repository using PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo initializes a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// ─────────────────────────────────────────────────────────────────────────────
// Locations
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateLocation(ctx context.Context, loc *stock.StockLocation) error {
	query := `
		INSERT INTO stock_locations (
			name, complete_name, usage, parent_id, scrap_location, return_location, valuation_account_id, company_id, active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, true, NOW(), NOW()
		) RETURNING id, created_at, updated_at
	`
	if loc.Usage == "" {
		loc.Usage = stock.LocationUsageInternal
	}
	if loc.CompleteName == "" {
		loc.CompleteName = loc.Name
	}

	err := r.pool.QueryRow(ctx, query,
		loc.Name, loc.CompleteName, string(loc.Usage), loc.ParentID, loc.ScrapLocation, loc.ReturnLocation, loc.ValuationAccountID, loc.CompanyID,
	).Scan(&loc.ID, &loc.CreatedAt, &loc.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to insert stock location", err)
	}
	loc.Active = true
	return nil
}

func (r *PostgresRepo) GetLocationByID(ctx context.Context, id int64) (*stock.StockLocation, error) {
	query := `
		SELECT id, name, complete_name, usage, parent_id, scrap_location, return_location, valuation_account_id, company_id, active, created_at, updated_at
		FROM stock_locations
		WHERE id = $1 AND active = true
	`
	var loc stock.StockLocation
	var usageStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&loc.ID, &loc.Name, &loc.CompleteName, &usageStr, &loc.ParentID,
		&loc.ScrapLocation, &loc.ReturnLocation, &loc.ValuationAccountID, &loc.CompanyID, &loc.Active, &loc.CreatedAt, &loc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("stock location #%d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch stock location", err)
	}
	loc.Usage = stock.LocationUsage(usageStr)
	return &loc, nil
}

func (r *PostgresRepo) GetLocationByName(ctx context.Context, name string) (*stock.StockLocation, error) {
	query := `
		SELECT id, name, complete_name, usage, parent_id, scrap_location, return_location, valuation_account_id, company_id, active, created_at, updated_at
		FROM stock_locations
		WHERE LOWER(name) = LOWER($1) AND active = true
		LIMIT 1
	`
	var loc stock.StockLocation
	var usageStr string
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&loc.ID, &loc.Name, &loc.CompleteName, &usageStr, &loc.ParentID,
		&loc.ScrapLocation, &loc.ReturnLocation, &loc.ValuationAccountID, &loc.CompanyID, &loc.Active, &loc.CreatedAt, &loc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("stock location '%s' not found", name))
		}
		return nil, platformerrors.Internal("failed to fetch stock location by name", err)
	}
	loc.Usage = stock.LocationUsage(usageStr)
	return &loc, nil
}

func (r *PostgresRepo) GetLocationByUsage(ctx context.Context, usage stock.LocationUsage) (*stock.StockLocation, error) {
	query := `
		SELECT id, name, complete_name, usage, parent_id, scrap_location, return_location, valuation_account_id, company_id, active, created_at, updated_at
		FROM stock_locations
		WHERE usage = $1 AND active = true
		ORDER BY id ASC
		LIMIT 1
	`
	var loc stock.StockLocation
	var usageStr string
	err := r.pool.QueryRow(ctx, query, string(usage)).Scan(
		&loc.ID, &loc.Name, &loc.CompleteName, &usageStr, &loc.ParentID,
		&loc.ScrapLocation, &loc.ReturnLocation, &loc.ValuationAccountID, &loc.CompanyID, &loc.Active, &loc.CreatedAt, &loc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("stock location with usage '%s' not found", usage))
		}
		return nil, platformerrors.Internal("failed to fetch stock location by usage", err)
	}
	loc.Usage = stock.LocationUsage(usageStr)
	return &loc, nil
}

func (r *PostgresRepo) UpdateLocation(ctx context.Context, loc *stock.StockLocation) error {
	query := `
		UPDATE stock_locations
		SET name = $1, complete_name = $2, usage = $3, parent_id = $4, scrap_location = $5,
		    return_location = $6, valuation_account_id = $7, company_id = $8, updated_at = NOW()
		WHERE id = $9 AND active = true
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		loc.Name, loc.CompleteName, string(loc.Usage), loc.ParentID, loc.ScrapLocation,
		loc.ReturnLocation, loc.ValuationAccountID, loc.CompanyID, loc.ID,
	).Scan(&loc.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("stock location #%d not found", loc.ID))
		}
		return platformerrors.Internal("failed to update stock location", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteLocation(ctx context.Context, id int64) error {
	query := `UPDATE stock_locations SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	res, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete stock location", err)
	}
	if res.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("stock location #%d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListLocations(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.StockLocation], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedLocationFilterFields, 1)
	if err != nil {
		return pagination.PageResult[stock.StockLocation]{}, platformerrors.Validation("invalid filter", err)
	}

	if whereClause == "" {
		whereClause = "WHERE active = true"
	} else {
		whereClause += " AND active = true"
	}

	countQuery := "SELECT COUNT(*) FROM stock_locations " + whereClause
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.PageResult[stock.StockLocation]{}, platformerrors.Internal("failed to count stock locations", err)
	}

	query := fmt.Sprintf(`
		SELECT id, name, complete_name, usage, parent_id, scrap_location, return_location, valuation_account_id, company_id, active, created_at, updated_at
		FROM stock_locations
		%s
		ORDER BY id ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIdx, nextIdx+1)

	args = append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return pagination.PageResult[stock.StockLocation]{}, platformerrors.Internal("failed to list stock locations", err)
	}
	defer rows.Close()

	var locations []stock.StockLocation
	for rows.Next() {
		var loc stock.StockLocation
		var usageStr string
		if err := rows.Scan(
			&loc.ID, &loc.Name, &loc.CompleteName, &usageStr, &loc.ParentID,
			&loc.ScrapLocation, &loc.ReturnLocation, &loc.ValuationAccountID, &loc.CompanyID, &loc.Active, &loc.CreatedAt, &loc.UpdatedAt,
		); err != nil {
			return pagination.PageResult[stock.StockLocation]{}, platformerrors.Internal("failed to scan stock location", err)
		}
		loc.Usage = stock.LocationUsage(usageStr)
		locations = append(locations, loc)
	}

	return pagination.NewPageResult(locations, total, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Warehouses
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateWarehouse(ctx context.Context, wh *stock.Warehouse) error {
	query := `
		INSERT INTO stock_warehouses (
			name, code, company_id, partner_id, view_location_id, lot_stock_id, active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, true, NOW(), NOW()
		) RETURNING id, created_at, updated_at
	`
	wh.Code = strings.ToUpper(strings.TrimSpace(wh.Code))
	err := r.pool.QueryRow(ctx, query,
		wh.Name, wh.Code, wh.CompanyID, wh.PartnerID, wh.ViewLocationID, wh.LotStockID,
	).Scan(&wh.ID, &wh.CreatedAt, &wh.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "stock_warehouses_code_key") || strings.Contains(err.Error(), "unique constraint") {
			return platformerrors.Conflict(fmt.Sprintf("warehouse code '%s' already exists", wh.Code), err)
		}
		return platformerrors.Internal("failed to create warehouse", err)
	}
	wh.Active = true
	return nil
}

func (r *PostgresRepo) GetWarehouseByID(ctx context.Context, id int64) (*stock.Warehouse, error) {
	query := `
		SELECT id, name, code, company_id, partner_id, view_location_id, lot_stock_id, active, created_at, updated_at
		FROM stock_warehouses
		WHERE id = $1 AND active = true
	`
	var wh stock.Warehouse
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&wh.ID, &wh.Name, &wh.Code, &wh.CompanyID, &wh.PartnerID,
		&wh.ViewLocationID, &wh.LotStockID, &wh.Active, &wh.CreatedAt, &wh.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("warehouse #%d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch warehouse", err)
	}
	return &wh, nil
}

func (r *PostgresRepo) GetWarehouseByCode(ctx context.Context, code string) (*stock.Warehouse, error) {
	query := `
		SELECT id, name, code, company_id, partner_id, view_location_id, lot_stock_id, active, created_at, updated_at
		FROM stock_warehouses
		WHERE UPPER(code) = UPPER($1) AND active = true
		LIMIT 1
	`
	var wh stock.Warehouse
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&wh.ID, &wh.Name, &wh.Code, &wh.CompanyID, &wh.PartnerID,
		&wh.ViewLocationID, &wh.LotStockID, &wh.Active, &wh.CreatedAt, &wh.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("warehouse with code '%s' not found", code))
		}
		return nil, platformerrors.Internal("failed to fetch warehouse by code", err)
	}
	return &wh, nil
}

func (r *PostgresRepo) UpdateWarehouse(ctx context.Context, wh *stock.Warehouse) error {
	query := `
		UPDATE stock_warehouses
		SET name = $1, code = $2, company_id = $3, partner_id = $4, view_location_id = $5, lot_stock_id = $6, updated_at = NOW()
		WHERE id = $7 AND active = true
		RETURNING updated_at
	`
	wh.Code = strings.ToUpper(strings.TrimSpace(wh.Code))
	err := r.pool.QueryRow(ctx, query,
		wh.Name, wh.Code, wh.CompanyID, wh.PartnerID, wh.ViewLocationID, wh.LotStockID, wh.ID,
	).Scan(&wh.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("warehouse #%d not found", wh.ID))
		}
		if strings.Contains(err.Error(), "stock_warehouses_code_key") || strings.Contains(err.Error(), "unique constraint") {
			return platformerrors.Conflict(fmt.Sprintf("warehouse code '%s' already exists", wh.Code), err)
		}
		return platformerrors.Internal("failed to update warehouse", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteWarehouse(ctx context.Context, id int64) error {
	query := `UPDATE stock_warehouses SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	res, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete warehouse", err)
	}
	if res.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("warehouse #%d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListWarehouses(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.Warehouse], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedWarehouseFilterFields, 1)
	if err != nil {
		return pagination.PageResult[stock.Warehouse]{}, platformerrors.Validation("invalid filter", err)
	}

	if whereClause == "" {
		whereClause = "WHERE active = true"
	} else {
		whereClause += " AND active = true"
	}

	countQuery := "SELECT COUNT(*) FROM stock_warehouses " + whereClause
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.PageResult[stock.Warehouse]{}, platformerrors.Internal("failed to count warehouses", err)
	}

	query := fmt.Sprintf(`
		SELECT id, name, code, company_id, partner_id, view_location_id, lot_stock_id, active, created_at, updated_at
		FROM stock_warehouses
		%s
		ORDER BY id ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIdx, nextIdx+1)

	args = append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return pagination.PageResult[stock.Warehouse]{}, platformerrors.Internal("failed to list warehouses", err)
	}
	defer rows.Close()

	var warehouses []stock.Warehouse
	for rows.Next() {
		var wh stock.Warehouse
		if err := rows.Scan(
			&wh.ID, &wh.Name, &wh.Code, &wh.CompanyID, &wh.PartnerID,
			&wh.ViewLocationID, &wh.LotStockID, &wh.Active, &wh.CreatedAt, &wh.UpdatedAt,
		); err != nil {
			return pagination.PageResult[stock.Warehouse]{}, platformerrors.Internal("failed to scan warehouse", err)
		}
		warehouses = append(warehouses, wh)
	}

	return pagination.NewPageResult(warehouses, total, page), nil
}

func (r *PostgresRepo) GetDefaultWarehouse(ctx context.Context) (*stock.Warehouse, error) {
	query := `
		SELECT id, name, code, company_id, partner_id, view_location_id, lot_stock_id, active, created_at, updated_at
		FROM stock_warehouses
		WHERE active = true
		ORDER BY id ASC
		LIMIT 1
	`
	var wh stock.Warehouse
	err := r.pool.QueryRow(ctx, query).Scan(
		&wh.ID, &wh.Name, &wh.Code, &wh.CompanyID, &wh.PartnerID,
		&wh.ViewLocationID, &wh.LotStockID, &wh.Active, &wh.CreatedAt, &wh.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound("no default warehouse found")
		}
		return nil, platformerrors.Internal("failed to fetch default warehouse", err)
	}
	return &wh, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Sequences & Pickings
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) NextSequence(ctx context.Context, pickingType stock.PickingType, year int) (string, error) {
	var seqName, prefix string
	switch pickingType {
	case stock.PickingTypeIncoming:
		seqName = "stock_picking_in_seq"
		prefix = "WH/IN"
	case stock.PickingTypeOutgoing:
		seqName = "stock_picking_out_seq"
		prefix = "WH/OUT"
	case stock.PickingTypeInternal:
		seqName = "stock_picking_int_seq"
		prefix = "WH/INT"
	default:
		return "", platformerrors.Validation("unknown picking type", nil)
	}

	var nextVal int64
	query := fmt.Sprintf("SELECT nextval('%s')", seqName)
	if err := r.pool.QueryRow(ctx, query).Scan(&nextVal); err != nil {
		return "", platformerrors.Internal("failed to generate next sequence", err)
	}

	return fmt.Sprintf("%s/%d/%05d", prefix, year, nextVal), nil
}

func (r *PostgresRepo) CreatePicking(ctx context.Context, picking *stock.StockPicking) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			INSERT INTO stock_pickings (
				name, picking_type, state, partner_id, location_id, location_dest_id,
				scheduled_date, date_done, origin, source_order_id, procurement_group_id, company_id, note, active, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, true, NOW(), NOW()
			) RETURNING id, created_at, updated_at
		`
		if picking.ScheduledDate.IsZero() {
			picking.ScheduledDate = time.Now().UTC()
		}
		if picking.State == "" {
			picking.State = stock.PickingStateDraft
		}

		err := tx.QueryRow(ctx, query,
			picking.Name, string(picking.PickingType), string(picking.State), picking.PartnerID,
			picking.LocationID, picking.LocationDestID, picking.ScheduledDate, picking.DateDone,
			picking.Origin, picking.SourceOrderID, picking.ProcurementGroupID, picking.CompanyID, picking.Note,
		).Scan(&picking.ID, &picking.CreatedAt, &picking.UpdatedAt)

		if err != nil {
			if strings.Contains(err.Error(), "stock_pickings_name_key") || strings.Contains(err.Error(), "unique constraint") {
				return platformerrors.Conflict(fmt.Sprintf("picking name '%s' already exists", picking.Name), err)
			}
			return platformerrors.Internal("failed to insert stock picking", err)
		}

		// Insert moves
		moveQuery := `
			INSERT INTO stock_moves (
				picking_id, sequence, name, product_id, product_uom, product_qty, quantity_done,
				location_id, location_dest_id, state, sale_line_id, purchase_line_id, procurement_group_id, date, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW()
			) RETURNING id, created_at, updated_at
		`
		for i := range picking.Moves {
			m := &picking.Moves[i]
			m.PickingID = &picking.ID
			if m.Sequence == 0 {
				m.Sequence = (i + 1) * 10
			}
			if m.State == "" {
				m.State = stock.MoveState(picking.State)
			}
			if m.Date.IsZero() {
				m.Date = picking.ScheduledDate
			}

			err := tx.QueryRow(ctx, moveQuery,
				m.PickingID, m.Sequence, m.Name, m.ProductID, m.ProductUom, m.ProductQty, m.QuantityDone,
				m.LocationID, m.LocationDestID, string(m.State), m.SaleLineID, m.PurchaseLineID, m.ProcurementGroupID, m.Date,
			).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)

			if err != nil {
				return platformerrors.Internal("failed to insert stock move", err)
			}
		}

		picking.Active = true
		return nil
	})
}

func (r *PostgresRepo) GetPickingByID(ctx context.Context, id int64) (*stock.StockPicking, error) {
	query := `
		SELECT id, name, picking_type, state, partner_id, location_id, location_dest_id,
		       scheduled_date, date_done, origin, source_order_id, procurement_group_id, company_id, note, active, created_at, updated_at
		FROM stock_pickings
		WHERE id = $1 AND active = true
	`
	var p stock.StockPicking
	var ptStr, stateStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &ptStr, &stateStr, &p.PartnerID, &p.LocationID, &p.LocationDestID,
		&p.ScheduledDate, &p.DateDone, &p.Origin, &p.SourceOrderID, &p.ProcurementGroupID, &p.CompanyID, &p.Note, &p.Active,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("stock picking #%d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch stock picking", err)
	}
	p.PickingType = stock.PickingType(ptStr)
	p.State = stock.PickingState(stateStr)

	moves, err := r.GetMovesByPickingID(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Moves = moves
	return &p, nil
}

func (r *PostgresRepo) GetPickingByName(ctx context.Context, name string) (*stock.StockPicking, error) {
	query := `
		SELECT id, name, picking_type, state, partner_id, location_id, location_dest_id,
		       scheduled_date, date_done, origin, source_order_id, procurement_group_id, company_id, note, active, created_at, updated_at
		FROM stock_pickings
		WHERE name = $1 AND active = true
		LIMIT 1
	`
	var p stock.StockPicking
	var ptStr, stateStr string
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&p.ID, &p.Name, &ptStr, &stateStr, &p.PartnerID, &p.LocationID, &p.LocationDestID,
		&p.ScheduledDate, &p.DateDone, &p.Origin, &p.SourceOrderID, &p.ProcurementGroupID, &p.CompanyID, &p.Note, &p.Active,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("stock picking '%s' not found", name))
		}
		return nil, platformerrors.Internal("failed to fetch stock picking by name", err)
	}
	p.PickingType = stock.PickingType(ptStr)
	p.State = stock.PickingState(stateStr)

	moves, err := r.GetMovesByPickingID(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.Moves = moves
	return &p, nil
}

func (r *PostgresRepo) UpdatePicking(ctx context.Context, picking *stock.StockPicking) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			UPDATE stock_pickings
			SET name = $1, state = $2, partner_id = $3, location_id = $4, location_dest_id = $5,
			    scheduled_date = $6, date_done = $7, origin = $8, source_order_id = $9,
			    procurement_group_id = $10, company_id = $11, note = $12, updated_at = NOW()
			WHERE id = $13 AND active = true
			RETURNING updated_at
		`
		err := tx.QueryRow(ctx, query,
			picking.Name, string(picking.State), picking.PartnerID, picking.LocationID, picking.LocationDestID,
			picking.ScheduledDate, picking.DateDone, picking.Origin, picking.SourceOrderID,
			picking.ProcurementGroupID, picking.CompanyID, picking.Note, picking.ID,
		).Scan(&picking.UpdatedAt)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return platformerrors.NotFound(fmt.Sprintf("stock picking #%d not found", picking.ID))
			}
			return platformerrors.Internal("failed to update stock picking", err)
		}

		// Delete and re-insert moves
		if _, err := tx.Exec(ctx, "DELETE FROM stock_moves WHERE picking_id = $1", picking.ID); err != nil {
			return platformerrors.Internal("failed to clear previous stock moves", err)
		}

		moveQuery := `
			INSERT INTO stock_moves (
				picking_id, sequence, name, product_id, product_uom, product_qty, quantity_done,
				location_id, location_dest_id, state, sale_line_id, purchase_line_id,
				procurement_group_id, date, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW()
			) RETURNING id, created_at, updated_at
		`
		for i := range picking.Moves {
			m := &picking.Moves[i]
			m.PickingID = &picking.ID
			if m.Sequence == 0 {
				m.Sequence = (i + 1) * 10
			}
			if m.State == "" {
				m.State = stock.MoveState(picking.State)
			}
			if m.Date.IsZero() {
				m.Date = picking.ScheduledDate
			}

			err := tx.QueryRow(ctx, moveQuery,
				m.PickingID, m.Sequence, m.Name, m.ProductID, m.ProductUom, m.ProductQty, m.QuantityDone,
				m.LocationID, m.LocationDestID, string(m.State), m.SaleLineID, m.PurchaseLineID,
				m.ProcurementGroupID, m.Date,
			).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)

			if err != nil {
				return platformerrors.Internal("failed to update stock move line", err)
			}
		}

		return nil
	})
}

func (r *PostgresRepo) DeletePicking(ctx context.Context, id int64) error {
	query := `UPDATE stock_pickings SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	res, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete stock picking", err)
	}
	if res.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("stock picking #%d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListPickings(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.StockPicking], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedPickingFilterFields, 1)
	if err != nil {
		return pagination.PageResult[stock.StockPicking]{}, platformerrors.Validation("invalid filter", err)
	}

	if whereClause == "" {
		whereClause = "WHERE active = true"
	} else {
		whereClause += " AND active = true"
	}

	countQuery := "SELECT COUNT(*) FROM stock_pickings " + whereClause
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.PageResult[stock.StockPicking]{}, platformerrors.Internal("failed to count stock pickings", err)
	}

	query := fmt.Sprintf(`
		SELECT id, name, picking_type, state, partner_id, location_id, location_dest_id,
		       scheduled_date, date_done, origin, source_order_id, procurement_group_id, company_id, note, active, created_at, updated_at
		FROM stock_pickings
		%s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIdx, nextIdx+1)

	args = append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return pagination.PageResult[stock.StockPicking]{}, platformerrors.Internal("failed to list stock pickings", err)
	}
	defer rows.Close()

	var pickings []stock.StockPicking
	for rows.Next() {
		var p stock.StockPicking
		var ptStr, stateStr string
		if err := rows.Scan(
			&p.ID, &p.Name, &ptStr, &stateStr, &p.PartnerID, &p.LocationID, &p.LocationDestID,
			&p.ScheduledDate, &p.DateDone, &p.Origin, &p.SourceOrderID, &p.ProcurementGroupID, &p.CompanyID, &p.Note, &p.Active,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return pagination.PageResult[stock.StockPicking]{}, platformerrors.Internal("failed to scan stock picking", err)
		}
		p.PickingType = stock.PickingType(ptStr)
		p.State = stock.PickingState(stateStr)
		pickings = append(pickings, p)
	}

	return pagination.NewPageResult(pickings, total, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Moves
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateMove(ctx context.Context, move *stock.StockMove) error {
	query := `
		INSERT INTO stock_moves (
			picking_id, sequence, name, product_id, product_uom, product_qty, quantity_done,
			location_id, location_dest_id, state, sale_line_id, purchase_line_id,
			production_id, production_finished_id, procurement_group_id, date, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW(), NOW()
		) RETURNING id, created_at, updated_at
	`
	if move.Date.IsZero() {
		move.Date = time.Now().UTC()
	}
	if move.State == "" {
		move.State = stock.MoveStateDraft
	}

	err := r.pool.QueryRow(ctx, query,
		move.PickingID, move.Sequence, move.Name, move.ProductID, move.ProductUom,
		move.ProductQty, move.QuantityDone, move.LocationID, move.LocationDestID,
		string(move.State), move.SaleLineID, move.PurchaseLineID,
		move.ProductionID, move.ProductionFinishedID, move.ProcurementGroupID, move.Date,
	).Scan(&move.ID, &move.CreatedAt, &move.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create stock move", err)
	}
	return nil
}

func (r *PostgresRepo) GetMoveByID(ctx context.Context, id int64) (*stock.StockMove, error) {
	query := `
		SELECT id, picking_id, sequence, name, product_id, product_uom, product_qty, quantity_done,
		       location_id, location_dest_id, state, sale_line_id, purchase_line_id,
		       production_id, production_finished_id, procurement_group_id,
		       date, value, value_manual, standard_price, is_in, is_out, is_dropship, remaining_qty,
		       remaining_value, account_move_id, created_at, updated_at
		FROM stock_moves
		WHERE id = $1
	`
	var m stock.StockMove
	var stateStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.PickingID, &m.Sequence, &m.Name, &m.ProductID, &m.ProductUom,
		&m.ProductQty, &m.QuantityDone, &m.LocationID, &m.LocationDestID, &stateStr,
		&m.SaleLineID, &m.PurchaseLineID, &m.ProductionID, &m.ProductionFinishedID, &m.ProcurementGroupID, &m.Date, &m.Value, &m.ValueManual,
		&m.StandardPrice, &m.IsIn, &m.IsOut, &m.IsDropship, &m.RemainingQty, &m.RemainingValue,
		&m.AccountMoveID, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("stock move #%d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch stock move", err)
	}
	m.State = stock.MoveState(stateStr)
	return &m, nil
}

func (r *PostgresRepo) UpdateMove(ctx context.Context, move *stock.StockMove) error {
	query := `
		UPDATE stock_moves
		SET sequence = $1, name = $2, product_id = $3, product_uom = $4, product_qty = $5, quantity_done = $6,
		    location_id = $7, location_dest_id = $8, state = $9, sale_line_id = $10, purchase_line_id = $11,
		    production_id = $12, production_finished_id = $13,
		    procurement_group_id = $14, date = $15, value = $16, value_manual = $17, standard_price = $18,
		    is_in = $19, is_out = $20, is_dropship = $21, remaining_qty = $22, remaining_value = $23,
		    account_move_id = $24, updated_at = NOW()
		WHERE id = $25
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		move.Sequence, move.Name, move.ProductID, move.ProductUom, move.ProductQty, move.QuantityDone,
		move.LocationID, move.LocationDestID, string(move.State), move.SaleLineID, move.PurchaseLineID,
		move.ProductionID, move.ProductionFinishedID,
		move.ProcurementGroupID, move.Date, move.Value, move.ValueManual, move.StandardPrice,
		move.IsIn, move.IsOut, move.IsDropship, move.RemainingQty, move.RemainingValue,
		move.AccountMoveID, move.ID,
	).Scan(&move.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("stock move #%d not found", move.ID))
		}
		return platformerrors.Internal("failed to update stock move", err)
	}
	return nil
}

func (r *PostgresRepo) ListMoves(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.StockMove], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedMoveFilterFields, 1)
	if err != nil {
		return pagination.PageResult[stock.StockMove]{}, platformerrors.Validation("invalid filter", err)
	}

	countQuery := "SELECT COUNT(*) FROM stock_moves " + whereClause
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.PageResult[stock.StockMove]{}, platformerrors.Internal("failed to count stock moves", err)
	}

	query := fmt.Sprintf(`
		SELECT id, picking_id, sequence, name, product_id, product_uom, product_qty, quantity_done,
		       location_id, location_dest_id, state, sale_line_id, purchase_line_id,
		       production_id, production_finished_id, procurement_group_id,
		       date, value, value_manual, standard_price, is_in, is_out, is_dropship, remaining_qty,
		       remaining_value, account_move_id, created_at, updated_at
		FROM stock_moves
		%s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIdx, nextIdx+1)

	args = append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return pagination.PageResult[stock.StockMove]{}, platformerrors.Internal("failed to list stock moves", err)
	}
	defer rows.Close()

	var moves []stock.StockMove
	for rows.Next() {
		var m stock.StockMove
		var stateStr string
		if err := rows.Scan(
			&m.ID, &m.PickingID, &m.Sequence, &m.Name, &m.ProductID, &m.ProductUom,
			&m.ProductQty, &m.QuantityDone, &m.LocationID, &m.LocationDestID, &stateStr,
			&m.SaleLineID, &m.PurchaseLineID, &m.ProductionID, &m.ProductionFinishedID, &m.ProcurementGroupID, &m.Date, &m.Value, &m.ValueManual,
			&m.StandardPrice, &m.IsIn, &m.IsOut, &m.IsDropship, &m.RemainingQty, &m.RemainingValue,
			&m.AccountMoveID, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return pagination.PageResult[stock.StockMove]{}, platformerrors.Internal("failed to scan stock move", err)
		}
		m.State = stock.MoveState(stateStr)
		moves = append(moves, m)
	}

	return pagination.NewPageResult(moves, total, page), nil
}

func (r *PostgresRepo) GetMovesByPickingID(ctx context.Context, pickingID int64) ([]stock.StockMove, error) {
	query := `
		SELECT id, picking_id, sequence, name, product_id, product_uom, product_qty, quantity_done,
		       location_id, location_dest_id, state, sale_line_id, purchase_line_id, procurement_group_id,
		       date, value, value_manual, standard_price, is_in, is_out, is_dropship, remaining_qty,
		       remaining_value, account_move_id, created_at, updated_at
		FROM stock_moves
		WHERE picking_id = $1
		ORDER BY sequence ASC, id ASC
	`
	rows, err := r.pool.Query(ctx, query, pickingID)
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch stock moves by picking ID", err)
	}
	defer rows.Close()

	var moves []stock.StockMove
	for rows.Next() {
		var m stock.StockMove
		var stateStr string
		if err := rows.Scan(
			&m.ID, &m.PickingID, &m.Sequence, &m.Name, &m.ProductID, &m.ProductUom,
			&m.ProductQty, &m.QuantityDone, &m.LocationID, &m.LocationDestID, &stateStr,
			&m.SaleLineID, &m.PurchaseLineID, &m.ProductionID, &m.ProductionFinishedID, &m.ProcurementGroupID, &m.Date, &m.Value, &m.ValueManual,
			&m.StandardPrice, &m.IsIn, &m.IsOut, &m.IsDropship, &m.RemainingQty, &m.RemainingValue,
			&m.AccountMoveID, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan stock move", err)
		}
		m.State = stock.MoveState(stateStr)
		moves = append(moves, m)
	}
	return moves, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Quants & Balances
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) GetQuant(ctx context.Context, productID, locationID int64) (*stock.StockQuant, error) {
	query := `
		SELECT id, product_id, location_id, quantity, reserved_quantity, company_id, created_at, updated_at
		FROM stock_quants
		WHERE product_id = $1 AND location_id = $2
	`
	var q stock.StockQuant
	err := r.pool.QueryRow(ctx, query, productID, locationID).Scan(
		&q.ID, &q.ProductID, &q.LocationID, &q.Quantity, &q.ReservedQuantity,
		&q.CompanyID, &q.CreatedAt, &q.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &stock.StockQuant{
				ProductID:  productID,
				LocationID: locationID,
				Quantity:   0,
			}, nil
		}
		return nil, platformerrors.Internal("failed to fetch stock quant", err)
	}
	return &q, nil
}

func (r *PostgresRepo) UpdateQuantQuantity(ctx context.Context, productID, locationID int64, deltaQty float64) error {
	query := `
		INSERT INTO stock_quants (product_id, location_id, quantity, reserved_quantity, created_at, updated_at)
		VALUES ($1, $2, $3, 0, NOW(), NOW())
		ON CONFLICT (product_id, location_id)
		DO UPDATE SET quantity = stock_quants.quantity + EXCLUDED.quantity, updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, productID, locationID, deltaQty)
	if err != nil {
		return platformerrors.Internal("failed to update quant quantity", err)
	}
	return nil
}

func (r *PostgresRepo) SetQuantQuantity(ctx context.Context, productID, locationID int64, newQty float64) error {
	query := `
		INSERT INTO stock_quants (product_id, location_id, quantity, reserved_quantity, created_at, updated_at)
		VALUES ($1, $2, $3, 0, NOW(), NOW())
		ON CONFLICT (product_id, location_id)
		DO UPDATE SET quantity = EXCLUDED.quantity, updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, productID, locationID, newQty)
	if err != nil {
		return platformerrors.Internal("failed to set quant quantity", err)
	}
	return nil
}

func (r *PostgresRepo) ListQuants(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.StockQuant], error) {
	allowedFields := map[string]string{
		"product_id":  "product_id",
		"location_id": "location_id",
	}
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedFields, 1)
	if err != nil {
		return pagination.PageResult[stock.StockQuant]{}, platformerrors.Validation("invalid filter", err)
	}

	countQuery := "SELECT COUNT(*) FROM stock_quants " + whereClause
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.PageResult[stock.StockQuant]{}, platformerrors.Internal("failed to count quants", err)
	}

	query := fmt.Sprintf(`
		SELECT id, product_id, location_id, quantity, reserved_quantity, company_id, created_at, updated_at
		FROM stock_quants
		%s
		ORDER BY product_id ASC, location_id ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIdx, nextIdx+1)

	args = append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return pagination.PageResult[stock.StockQuant]{}, platformerrors.Internal("failed to list quants", err)
	}
	defer rows.Close()

	var quants []stock.StockQuant
	for rows.Next() {
		var q stock.StockQuant
		if err := rows.Scan(
			&q.ID, &q.ProductID, &q.LocationID, &q.Quantity, &q.ReservedQuantity,
			&q.CompanyID, &q.CreatedAt, &q.UpdatedAt,
		); err != nil {
			return pagination.PageResult[stock.StockQuant]{}, platformerrors.Internal("failed to scan quant", err)
		}
		quants = append(quants, q)
	}

	return pagination.NewPageResult(quants, total, page), nil
}

func (r *PostgresRepo) GetOnHandStock(ctx context.Context, productID *int64, locationID *int64, warehouseID *int64) ([]stock.StockOnHandItem, error) {
	query := `
		SELECT q.product_id, pt.name as product_name, COALESCE(pt.internal_ref, '') as product_sku, pt.type as product_type,
		       q.location_id, sl.complete_name as location_name,
		       sw.id as warehouse_id, COALESCE(sw.name, '') as warehouse_name,
		       q.quantity, q.reserved_quantity, (q.quantity - q.reserved_quantity) as available_quantity,
		       COALESCE(uom.name, '') as uom_name
		FROM stock_quants q
		JOIN product_templates pt ON pt.id = q.product_id
		JOIN stock_locations sl ON sl.id = q.location_id
		LEFT JOIN stock_warehouses sw ON sw.lot_stock_id = sl.id AND sw.active = true
		LEFT JOIN uom_uoms uom ON uom.id = pt.uom_id
		WHERE sl.active = true
	`
	var args []any
	idx := 1

	if productID != nil {
		query += fmt.Sprintf(" AND q.product_id = $%d", idx)
		args = append(args, *productID)
		idx++
	}

	if locationID != nil {
		query += fmt.Sprintf(" AND q.location_id = $%d", idx)
		args = append(args, *locationID)
		idx++
	} else {
		// By default only internal storage locations
		query += " AND sl.usage = 'internal'"
	}

	if warehouseID != nil {
		query += fmt.Sprintf(" AND sw.id = $%d", idx)
		args = append(args, *warehouseID)
		idx++
	}

	query += " ORDER BY q.product_id ASC, q.location_id ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, platformerrors.Internal("failed to query on-hand stock", err)
	}
	defer rows.Close()

	var items []stock.StockOnHandItem
	for rows.Next() {
		var item stock.StockOnHandItem
		if err := rows.Scan(
			&item.ProductID, &item.ProductName, &item.ProductSKU, &item.ProductType,
			&item.LocationID, &item.LocationName,
			&item.WarehouseID, &item.WarehouseName,
			&item.Quantity, &item.ReservedQuantity, &item.AvailableQuantity,
			&item.UoMName,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan on-hand stock row", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// ValidatePickingTx updates picking and moves to done, and executes double-entry quant transfers atomically.
func (r *PostgresRepo) ValidatePickingTx(ctx context.Context, picking *stock.StockPicking) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		picking.State = stock.PickingStateDone
		picking.DateDone = &now

		// Update picking state
		updatePickingQ := `
			UPDATE stock_pickings
			SET state = 'done', date_done = $1, updated_at = NOW()
			WHERE id = $2
		`
		if _, err := tx.Exec(ctx, updatePickingQ, picking.DateDone, picking.ID); err != nil {
			return platformerrors.Internal("failed to update picking status to done", err)
		}

		// Update moves and apply quant deltas
		upsertQuantQ := `
			INSERT INTO stock_quants (product_id, location_id, quantity, reserved_quantity, created_at, updated_at)
			VALUES ($1, $2, $3, 0, NOW(), NOW())
			ON CONFLICT (product_id, location_id)
			DO UPDATE SET quantity = stock_quants.quantity + EXCLUDED.quantity, updated_at = NOW()
		`

		for i := range picking.Moves {
			m := &picking.Moves[i]
			qty := m.QuantityDone
			if qty <= 0 {
				qty = m.ProductQty
			}
			m.QuantityDone = qty
			m.State = stock.MoveStateDone

			// Update move record
			updateMoveQ := `
				UPDATE stock_moves
				SET state = 'done', quantity_done = $1, updated_at = NOW()
				WHERE id = $2
			`
			if _, err := tx.Exec(ctx, updateMoveQ, qty, m.ID); err != nil {
				return platformerrors.Internal("failed to update move status to done", err)
			}

			// Decrease source location
			if _, err := tx.Exec(ctx, upsertQuantQ, m.ProductID, m.LocationID, -qty); err != nil {
				return platformerrors.Internal("failed to debit source location quant", err)
			}

			// Increase destination location
			if _, err := tx.Exec(ctx, upsertQuantQ, m.ProductID, m.LocationDestID, qty); err != nil {
				return platformerrors.Internal("failed to credit destination location quant", err)
			}
		}

		return nil
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Valuations (Phase 12 — stock-account integration)
// ─────────────────────────────────────────────────────────────────────────────

// UpdateMoveValue persists the valuation fields of a stock move.
func (r *PostgresRepo) UpdateMoveValue(ctx context.Context, move *stock.StockMove) error {
	query := `
		UPDATE stock_moves
		SET value = $1, value_manual = $2, standard_price = $3, is_in = $4, is_out = $5, is_dropship = $6,
		    remaining_qty = $7, remaining_value = $8, account_move_id = $9, updated_at = NOW()
		WHERE id = $10
	`
	res, err := r.pool.Exec(ctx, query,
		move.Value, move.ValueManual, move.StandardPrice, move.IsIn, move.IsOut, move.IsDropship,
		move.RemainingQty, move.RemainingValue, move.AccountMoveID, move.ID,
	)
	if err != nil {
		return platformerrors.Internal("failed to update stock move valuation", err)
	}
	if res.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("stock move #%d not found", move.ID))
	}
	return nil
}

// GetFIFOStack returns the remaining incoming layers for a product, oldest first.
func (r *PostgresRepo) GetFIFOStack(ctx context.Context, productID, companyID int64) (stock.FIFOStack, error) {
	query := `
		SELECT m.id, m.remaining_qty, m.remaining_value
		FROM stock_moves m
		LEFT JOIN stock_pickings sp ON sp.id = m.picking_id
		WHERE m.product_id = $1 AND m.is_in = true AND m.state = 'done' AND m.remaining_qty > 0
		  AND (sp.company_id = $2 OR sp.id IS NULL)
		ORDER BY m.date ASC, m.id ASC
	`
	rows, err := r.pool.Query(ctx, query, productID, companyID)
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch FIFO stack", err)
	}
	defer rows.Close()

	var stack stock.FIFOStack
	for rows.Next() {
		var e stock.FIFOEntry
		if err := rows.Scan(&e.MoveID, &e.Qty, &e.Value); err != nil {
			return nil, platformerrors.Internal("failed to scan FIFO stack row", err)
		}
		stack = append(stack, e)
	}
	return stack, nil
}

// CreateProductValue inserts a history record of a product value update.
func (r *PostgresRepo) CreateProductValue(ctx context.Context, pv *stock.ProductValue) error {
	query := `
		INSERT INTO product_values (product_id, move_id, lot_id, value, company_id, date, user_id, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, created_at
	`
	err := r.pool.QueryRow(ctx, query,
		pv.ProductID, pv.MoveID, pv.LotID, pv.Value, pv.CompanyID, pv.Date, pv.UserID, pv.Description,
	).Scan(&pv.ID, &pv.CreatedAt)
	if err != nil {
		return platformerrors.Internal("failed to create product value", err)
	}
	return nil
}

var allowedProductValueFilterFields = map[string]string{
	"product_id": "product_id",
	"move_id":    "move_id",
}

// ListProductValues returns the history of product value records.
func (r *PostgresRepo) ListProductValues(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.ProductValue], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedProductValueFilterFields, 1)
	if err != nil {
		return pagination.PageResult[stock.ProductValue]{}, platformerrors.Validation("invalid filter", err)
	}

	countQuery := "SELECT COUNT(*) FROM product_values " + whereClause
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.PageResult[stock.ProductValue]{}, platformerrors.Internal("failed to count product values", err)
	}

	query := fmt.Sprintf(`
		SELECT id, product_id, move_id, lot_id, value, company_id, date, user_id, COALESCE(description, ''), created_at
		FROM product_values
		%s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIdx, nextIdx+1)

	args = append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return pagination.PageResult[stock.ProductValue]{}, platformerrors.Internal("failed to list product values", err)
	}
	defer rows.Close()

	var values []stock.ProductValue
	for rows.Next() {
		var pv stock.ProductValue
		if err := rows.Scan(
			&pv.ID, &pv.ProductID, &pv.MoveID, &pv.LotID, &pv.Value, &pv.CompanyID,
			&pv.Date, &pv.UserID, &pv.Description, &pv.CreatedAt,
		); err != nil {
			return pagination.PageResult[stock.ProductValue]{}, platformerrors.Internal("failed to scan product value", err)
		}
		values = append(values, pv)
	}

	return pagination.NewPageResult(values, total, page), nil
}

// ComputeTotalValuation aggregates the current valuation by product (and optional location).
func (r *PostgresRepo) ComputeTotalValuation(ctx context.Context, productID *int64, locationID *int64) ([]stock.ValuationSummary, error) {
	query := `
		SELECT q.product_id, pt.name as product_name, q.location_id, sl.complete_name as location_name,
		       SUM(q.quantity) as quantity,
		       COALESCE(pt.avg_cost, pt.standard_price) as unit_cost,
		       SUM(q.quantity * COALESCE(pt.avg_cost, pt.standard_price)) as value
		FROM stock_quants q
		JOIN product_templates pt ON pt.id = q.product_id
		JOIN stock_locations sl ON sl.id = q.location_id
		WHERE sl.usage = 'internal' AND q.quantity <> 0
	`
	var args []any
	idx := 1
	if productID != nil {
		query += fmt.Sprintf(" AND q.product_id = $%d", idx)
		args = append(args, *productID)
		idx++
	}
	if locationID != nil {
		query += fmt.Sprintf(" AND q.location_id = $%d", idx)
		args = append(args, *locationID)
		idx++
	}
	query += " GROUP BY q.product_id, pt.name, q.location_id, sl.complete_name, COALESCE(pt.avg_cost, pt.standard_price)"
	query += " ORDER BY q.product_id ASC, q.location_id ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, platformerrors.Internal("failed to compute total valuation", err)
	}
	defer rows.Close()

	var summaries []stock.ValuationSummary
	for rows.Next() {
		var s stock.ValuationSummary
		if err := rows.Scan(&s.ProductID, &s.ProductName, &s.LocationID, &s.Location,
			&s.Quantity, &s.UnitCost, &s.Value); err != nil {
			return nil, platformerrors.Internal("failed to scan valuation summary", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// Accounting Periods (periodic closing valuation)

// CreateAccountingPeriod inserts a new period.
func (r *PostgresRepo) CreateAccountingPeriod(ctx context.Context, p *stock.AccountingPeriod) error {
	query := `
		INSERT INTO accounting_periods (name, date_from, date_to, state, journal_id, account_move_id, company_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		p.Name, p.DateFrom, p.DateTo, p.State, p.JournalID, p.AccountMoveID, p.CompanyID,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return platformerrors.Internal("failed to create accounting period", err)
	}
	return nil
}

// GetAccountingPeriodByID fetches a single period.
func (r *PostgresRepo) GetAccountingPeriodByID(ctx context.Context, id int64) (*stock.AccountingPeriod, error) {
	query := `
		SELECT id, name, date_from, date_to, state, journal_id, account_move_id, company_id, created_at, updated_at
		FROM accounting_periods
		WHERE id = $1
	`
	var p stock.AccountingPeriod
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.DateFrom, &p.DateTo, &p.State, &p.JournalID,
		&p.AccountMoveID, &p.CompanyID, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("accounting period #%d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch accounting period", err)
	}
	return &p, nil
}

var allowedPeriodFilterFields = map[string]string{
	"state":      "state",
	"journal_id": "journal_id",
}

// ListAccountingPeriods returns a paginated list of periods.
func (r *PostgresRepo) ListAccountingPeriods(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.AccountingPeriod], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedPeriodFilterFields, 1)
	if err != nil {
		return pagination.PageResult[stock.AccountingPeriod]{}, platformerrors.Validation("invalid filter", err)
	}

	countQuery := "SELECT COUNT(*) FROM accounting_periods " + whereClause
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.PageResult[stock.AccountingPeriod]{}, platformerrors.Internal("failed to count accounting periods", err)
	}

	query := fmt.Sprintf(`
		SELECT id, name, date_from, date_to, state, journal_id, account_move_id, company_id, created_at, updated_at
		FROM accounting_periods
		%s
		ORDER BY date_from DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIdx, nextIdx+1)

	args = append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return pagination.PageResult[stock.AccountingPeriod]{}, platformerrors.Internal("failed to list accounting periods", err)
	}
	defer rows.Close()

	var periods []stock.AccountingPeriod
	for rows.Next() {
		var p stock.AccountingPeriod
		if err := rows.Scan(
			&p.ID, &p.Name, &p.DateFrom, &p.DateTo, &p.State, &p.JournalID,
			&p.AccountMoveID, &p.CompanyID, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return pagination.PageResult[stock.AccountingPeriod]{}, platformerrors.Internal("failed to scan accounting period", err)
		}
		periods = append(periods, p)
	}

	return pagination.NewPageResult(periods, total, page), nil
}

// CloseAccountingPeriod marks a period as closed and links the closing entry.
func (r *PostgresRepo) CloseAccountingPeriod(ctx context.Context, p *stock.AccountingPeriod) error {
	query := `
		UPDATE accounting_periods
		SET state = 'closed', account_move_id = $1, updated_at = NOW()
		WHERE id = $2 AND state = 'open'
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query, p.AccountMoveID, p.ID).Scan(&p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.Conflict("accounting period is not open or does not exist")
		}
		return platformerrors.Internal("failed to close accounting period", err)
	}
	p.State = "closed"
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Reorder Rules (Phase 13 — stock.orderpoint)
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) orderpointNameTx(ctx context.Context, tx pgx.Tx, year int) (string, error) {
	var n int64
	if err := tx.QueryRow(ctx, "SELECT nextval('stock_orderpoint_sequence')").Scan(&n); err != nil {
		return "", platformerrors.Internal("failed to generate orderpoint sequence", err)
	}
	return fmt.Sprintf("ROP/%d/%05d", year, n), nil
}

func (r *PostgresRepo) CreateOrderpoint(ctx context.Context, op *stock.Orderpoint) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		if op.Name == "" {
			name, err := r.orderpointNameTx(ctx, tx, time.Now().UTC().Year())
			if err != nil {
				return err
			}
			op.Name = name
		}
		query := `
			INSERT INTO stock_orderpoints (
				name, product_id, warehouse_id, location_id, vendor_id, min_qty, max_qty, qty_multiple,
				lead_days, source, trigger, snoozed_until, qty_on_hand, qty_forecast, qty_to_order,
				qty_to_order_manual, deadline_date, active, company_id, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, true, $18, NOW(), NOW()
			) RETURNING id, created_at, updated_at
		`
		err := tx.QueryRow(ctx, query,
			op.Name, op.ProductID, op.WarehouseID, op.LocationID, op.VendorID, op.MinQty, op.MaxQty,
			op.QtyMultiple, op.LeadDays, string(op.Source), string(op.Trigger), op.SnoozedUntil,
			op.QtyOnHand, op.QtyForecast, op.QtyToOrder, op.QtyToOrderManual, op.DeadlineDate, op.CompanyID,
		).Scan(&op.ID, &op.CreatedAt, &op.UpdatedAt)

		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") {
				return platformerrors.Conflict("an orderpoint already exists for this product in this location", err)
			}
			return platformerrors.Internal("failed to insert stock orderpoint", err)
		}
		op.Active = true
		return nil
	})
}

func (r *PostgresRepo) GetOrderpointByID(ctx context.Context, id int64) (*stock.Orderpoint, error) {
	query := `
		SELECT id, name, product_id, warehouse_id, location_id, vendor_id, min_qty, max_qty, qty_multiple,
		       lead_days, source, trigger, snoozed_until, qty_on_hand, qty_forecast, qty_to_order,
		       qty_to_order_manual, deadline_date, active, company_id, created_at, updated_at
		FROM stock_orderpoints
		WHERE id = $1
	`
	var op stock.Orderpoint
	var sourceStr, triggerStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&op.ID, &op.Name, &op.ProductID, &op.WarehouseID, &op.LocationID, &op.VendorID,
		&op.MinQty, &op.MaxQty, &op.QtyMultiple, &op.LeadDays, &sourceStr, &triggerStr,
		&op.SnoozedUntil, &op.QtyOnHand, &op.QtyForecast, &op.QtyToOrder, &op.QtyToOrderManual,
		&op.DeadlineDate, &op.Active, &op.CompanyID, &op.CreatedAt, &op.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("stock orderpoint #%d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch stock orderpoint", err)
	}
	op.Source = stock.OrderpointSource(sourceStr)
	op.Trigger = stock.OrderpointTrigger(triggerStr)
	return &op, nil
}

func (r *PostgresRepo) UpdateOrderpoint(ctx context.Context, op *stock.Orderpoint) error {
	query := `
		UPDATE stock_orderpoints
		SET name = $1, product_id = $2, warehouse_id = $3, location_id = $4, vendor_id = $5,
		    min_qty = $6, max_qty = $7, qty_multiple = $8, lead_days = $9, source = $10, trigger = $11,
		    snoozed_until = $12, qty_on_hand = $13, qty_forecast = $14, qty_to_order = $15,
		    qty_to_order_manual = $16, deadline_date = $17, active = $18, company_id = $19, updated_at = NOW()
		WHERE id = $20
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		op.Name, op.ProductID, op.WarehouseID, op.LocationID, op.VendorID,
		op.MinQty, op.MaxQty, op.QtyMultiple, op.LeadDays, string(op.Source), string(op.Trigger),
		op.SnoozedUntil, op.QtyOnHand, op.QtyForecast, op.QtyToOrder, op.QtyToOrderManual,
		op.DeadlineDate, op.Active, op.CompanyID, op.ID,
	).Scan(&op.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("stock orderpoint #%d not found", op.ID))
		}
		return platformerrors.Internal("failed to update stock orderpoint", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteOrderpoint(ctx context.Context, id int64) error {
	query := `UPDATE stock_orderpoints SET active = false, updated_at = NOW() WHERE id = $1 AND active = true`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to archive stock orderpoint", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("stock orderpoint #%d not found", id))
	}
	return nil
}

var allowedOrderpointFilterFields = map[string]string{
	"product_id":   "product_id",
	"warehouse_id": "warehouse_id",
	"location_id":  "location_id",
	"trigger":      "trigger",
	"source":       "source",
	"active":       "active",
}

func (r *PostgresRepo) ListOrderpoints(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.Orderpoint], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedOrderpointFilterFields, 1)
	if err != nil {
		return pagination.PageResult[stock.Orderpoint]{}, platformerrors.Validation("invalid filter", err)
	}
	if whereClause == "" {
		whereClause = "WHERE active = true"
	} else {
		whereClause += " AND active = true"
	}

	countQuery := "SELECT COUNT(*) FROM stock_orderpoints " + whereClause
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.PageResult[stock.Orderpoint]{}, platformerrors.Internal("failed to count stock orderpoints", err)
	}

	query := fmt.Sprintf(`
		SELECT id, name, product_id, warehouse_id, location_id, vendor_id, min_qty, max_qty, qty_multiple,
		       lead_days, source, trigger, snoozed_until, qty_on_hand, qty_forecast, qty_to_order,
		       qty_to_order_manual, deadline_date, active, company_id, created_at, updated_at
		FROM stock_orderpoints
		%s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIdx, nextIdx+1)

	args = append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return pagination.PageResult[stock.Orderpoint]{}, platformerrors.Internal("failed to list stock orderpoints", err)
	}
	defer rows.Close()

	var ops []stock.Orderpoint
	for rows.Next() {
		var op stock.Orderpoint
		var sourceStr, triggerStr string
		if err := rows.Scan(
			&op.ID, &op.Name, &op.ProductID, &op.WarehouseID, &op.LocationID, &op.VendorID,
			&op.MinQty, &op.MaxQty, &op.QtyMultiple, &op.LeadDays, &sourceStr, &triggerStr,
			&op.SnoozedUntil, &op.QtyOnHand, &op.QtyForecast, &op.QtyToOrder, &op.QtyToOrderManual,
			&op.DeadlineDate, &op.Active, &op.CompanyID, &op.CreatedAt, &op.UpdatedAt,
		); err != nil {
			return pagination.PageResult[stock.Orderpoint]{}, platformerrors.Internal("failed to scan stock orderpoint", err)
		}
		op.Source = stock.OrderpointSource(sourceStr)
		op.Trigger = stock.OrderpointTrigger(triggerStr)
		ops = append(ops, op)
	}

	return pagination.NewPageResult(ops, total, page), nil
}

func (r *PostgresRepo) ListOrderpointsForReplenishment(ctx context.Context, now time.Time) ([]*stock.Orderpoint, error) {
	query := `
		SELECT id, name, product_id, warehouse_id, location_id, vendor_id, min_qty, max_qty, qty_multiple,
		       lead_days, source, trigger, qty_to_order_manual, company_id
		FROM stock_orderpoints
		WHERE active = true
		  AND (snoozed_until IS NULL OR snoozed_until <= $1)
		  AND (trigger = 'auto' OR qty_to_order_manual > 0)
		ORDER BY product_id, warehouse_id, location_id
	`
	rows, err := r.pool.Query(ctx, query, now)
	if err != nil {
		return nil, platformerrors.Internal("failed to list orderpoints for replenishment", err)
	}
	defer rows.Close()

	var ops []*stock.Orderpoint
	for rows.Next() {
		op := &stock.Orderpoint{}
		var sourceStr, triggerStr string
		if err := rows.Scan(
			&op.ID, &op.Name, &op.ProductID, &op.WarehouseID, &op.LocationID, &op.VendorID,
			&op.MinQty, &op.MaxQty, &op.QtyMultiple, &op.LeadDays, &sourceStr, &triggerStr,
			&op.QtyToOrderManual, &op.CompanyID,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan orderpoint candidate", err)
		}
		op.Source = stock.OrderpointSource(sourceStr)
		op.Trigger = stock.OrderpointTrigger(triggerStr)
		ops = append(ops, op)
	}
	if err := rows.Err(); err != nil {
		return nil, platformerrors.Internal("failed to iterate orderpoint candidates", err)
	}
	return ops, nil
}

// StockForecast computes (on-hand, incoming, outgoing) for a product in a location
// with moves scheduled up to the given horizon (Odoo qty_forecast composition).
func (r *PostgresRepo) StockForecast(ctx context.Context, productID, locationID int64, at time.Time) (onHand, incoming, outgoing float64, err error) {
	query := `
		SELECT
			COALESCE((SELECT SUM(q.quantity) FROM stock_quants q WHERE q.product_id = $1 AND q.location_id = $2), 0),
			COALESCE((SELECT SUM(m.product_qty) FROM stock_moves m
			          WHERE m.product_id = $1 AND m.location_dest_id = $2
			            AND m.state IN ('confirmed', 'assigned') AND m.date <= $3), 0),
			COALESCE((SELECT SUM(m.product_qty) FROM stock_moves m
			          WHERE m.product_id = $1 AND m.location_id = $2
			            AND m.state IN ('confirmed', 'assigned') AND m.date <= $3), 0)
	`
	err = r.pool.QueryRow(ctx, query, productID, locationID, at).Scan(&onHand, &incoming, &outgoing)
	if err != nil {
		return 0, 0, 0, platformerrors.Internal("failed to compute stock forecast", err)
	}
	return onHand, incoming, outgoing, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Landed Costs (Phase 13 — stock.landed.cost)
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) nextLandedCostNameTx(ctx context.Context, tx pgx.Tx, year int) (string, error) {
	var n int64
	if err := tx.QueryRow(ctx, "SELECT nextval('stock_landed_cost_sequence')").Scan(&n); err != nil {
		return "", platformerrors.Internal("failed to generate landed cost sequence", err)
	}
	return fmt.Sprintf("LC/%d/%05d", year, n), nil
}

func (r *PostgresRepo) CreateLandedCost(ctx context.Context, lc *stock.LandedCost) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		if lc.Name == "" {
			name, err := r.nextLandedCostNameTx(ctx, tx, lc.Date.UTC().Year())
			if err != nil {
				return err
			}
			lc.Name = name
		}
		if lc.PickingIDs == nil {
			lc.PickingIDs = []int64{}
		}
		headerQuery := `
			INSERT INTO stock_landed_costs (
				name, date, state, picking_ids, amount_total, description, account_move_id,
				journal_id, vendor_bill_id, company_id, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW()
			) RETURNING id, created_at, updated_at
		`
		err := tx.QueryRow(ctx, headerQuery,
			lc.Name, lc.Date, string(lc.State), lc.PickingIDs, lc.AmountTotal, lc.Description,
			lc.AccountMoveID, lc.JournalID, lc.VendorBillID, lc.CompanyID,
		).Scan(&lc.ID, &lc.CreatedAt, &lc.UpdatedAt)
		if err != nil {
			return platformerrors.Internal("failed to insert stock landed cost", err)
		}

		for i := range lc.CostLines {
			if err := r.insertCostLineTx(ctx, tx, lc.ID, &lc.CostLines[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PostgresRepo) insertCostLineTx(ctx context.Context, tx pgx.Tx, landedCostID int64, line *stock.LandedCostLine) error {
	query := `
		INSERT INTO stock_landed_cost_lines (
			landed_cost_id, name, product_id, account_id, price_unit, split_method, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	line.LandedCostID = landedCostID
	return tx.QueryRow(ctx, query,
		line.LandedCostID, line.Name, line.ProductID, line.AccountID, line.PriceUnit, string(line.SplitMethod),
	).Scan(&line.ID, &line.CreatedAt, &line.UpdatedAt)
}

func (r *PostgresRepo) insertAdjustmentTx(ctx context.Context, tx pgx.Tx, adj *stock.ValuationAdjustment) error {
	query := `
		INSERT INTO stock_valuation_adjustment_lines (
			landed_cost_id, cost_line_id, move_id, product_id, quantity, weight, volume,
			former_cost, additional_cost, final_cost, move_remaining_qty, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	return tx.QueryRow(ctx, query,
		adj.LandedCostID, adj.CostLineID, adj.MoveID, adj.ProductID, adj.Quantity, adj.Weight, adj.Volume,
		adj.FormerCost, adj.AdditionalCost, adj.FinalCost, adj.MoveRemainingQty,
	).Scan(&adj.ID, &adj.CreatedAt, &adj.UpdatedAt)
}

func (r *PostgresRepo) GetLandedCostByID(ctx context.Context, id int64) (*stock.LandedCost, error) {
	query := `
		SELECT id, name, date, state, picking_ids, amount_total, description, account_move_id,
		       journal_id, vendor_bill_id, company_id, created_at, updated_at
		FROM stock_landed_costs
		WHERE id = $1
	`
	var lc stock.LandedCost
	var stateStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&lc.ID, &lc.Name, &lc.Date, &stateStr, &lc.PickingIDs, &lc.AmountTotal, &lc.Description,
		&lc.AccountMoveID, &lc.JournalID, &lc.VendorBillID, &lc.CompanyID, &lc.CreatedAt, &lc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("stock landed cost #%d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch stock landed cost", err)
	}
	lc.State = stock.LandedCostState(stateStr)

	lineQuery := `
		SELECT id, landed_cost_id, name, product_id, account_id, price_unit, split_method
		FROM stock_landed_cost_lines
		WHERE landed_cost_id = $1
		ORDER BY id
	`
	lRows, err := r.pool.Query(ctx, lineQuery, id)
	if err != nil {
		return nil, platformerrors.Internal("failed to list landed cost lines", err)
	}
	defer lRows.Close()
	for lRows.Next() {
		var line stock.LandedCostLine
		var splitStr string
		if err := lRows.Scan(&line.ID, &line.LandedCostID, &line.Name, &line.ProductID, &line.AccountID, &line.PriceUnit, &splitStr); err != nil {
			return nil, platformerrors.Internal("failed to scan landed cost line", err)
		}
		line.SplitMethod = stock.SplitMethod(splitStr)
		lc.CostLines = append(lc.CostLines, line)
	}
	if err := lRows.Err(); err != nil {
		return nil, platformerrors.Internal("failed to iterate landed cost lines", err)
	}

	adjQuery := `
		SELECT id, landed_cost_id, cost_line_id, move_id, product_id, quantity, weight, volume,
		       former_cost, additional_cost, final_cost, move_remaining_qty
		FROM stock_valuation_adjustment_lines
		WHERE landed_cost_id = $1
		ORDER BY id
	`
	aRows, err := r.pool.Query(ctx, adjQuery, id)
	if err != nil {
		return nil, platformerrors.Internal("failed to list valuation adjustments", err)
	}
	defer aRows.Close()
	for aRows.Next() {
		var adj stock.ValuationAdjustment
		if err := aRows.Scan(
			&adj.ID, &adj.LandedCostID, &adj.CostLineID, &adj.MoveID, &adj.ProductID,
			&adj.Quantity, &adj.Weight, &adj.Volume, &adj.FormerCost, &adj.AdditionalCost,
			&adj.FinalCost, &adj.MoveRemainingQty,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan valuation adjustment", err)
		}
		lc.ValuationAdjustments = append(lc.ValuationAdjustments, adj)
	}
	if err := aRows.Err(); err != nil {
		return nil, platformerrors.Internal("failed to iterate valuation adjustments", err)
	}

	return &lc, nil
}

func (r *PostgresRepo) UpdateLandedCost(ctx context.Context, lc *stock.LandedCost) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		if lc.PickingIDs == nil {
			lc.PickingIDs = []int64{}
		}
		headerQuery := `
			UPDATE stock_landed_costs
			SET name = $1, date = $2, state = $3, picking_ids = $4, amount_total = $5, description = $6,
			    account_move_id = $7, journal_id = $8, vendor_bill_id = $9, company_id = $10, updated_at = NOW()
			WHERE id = $11
			RETURNING updated_at
		`
		err := tx.QueryRow(ctx, headerQuery,
			lc.Name, lc.Date, string(lc.State), lc.PickingIDs, lc.AmountTotal, lc.Description,
			lc.AccountMoveID, lc.JournalID, lc.VendorBillID, lc.CompanyID, lc.ID,
		).Scan(&lc.UpdatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return platformerrors.NotFound(fmt.Sprintf("stock landed cost #%d not found", lc.ID))
			}
			return platformerrors.Internal("failed to update stock landed cost", err)
		}

		if _, err := tx.Exec(ctx, `DELETE FROM stock_landed_cost_lines WHERE landed_cost_id = $1`, lc.ID); err != nil {
			return platformerrors.Internal("failed to clear landed cost lines", err)
		}
		for i := range lc.CostLines {
			if err := r.insertCostLineTx(ctx, tx, lc.ID, &lc.CostLines[i]); err != nil {
				return err
			}
		}

		if _, err := tx.Exec(ctx, `DELETE FROM stock_valuation_adjustment_lines WHERE landed_cost_id = $1`, lc.ID); err != nil {
			return platformerrors.Internal("failed to clear valuation adjustments", err)
		}
		for i := range lc.ValuationAdjustments {
			adj := lc.ValuationAdjustments[i]
			adj.LandedCostID = lc.ID
			if err := r.insertAdjustmentTx(ctx, tx, &adj); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PostgresRepo) DeleteLandedCost(ctx context.Context, id int64) error {
	query := `DELETE FROM stock_landed_costs WHERE id = $1 AND state IN ('draft', 'cancel')`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return platformerrors.Internal("failed to delete stock landed cost", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.Conflict("stock landed cost cannot be deleted once validated")
	}
	return nil
}

var allowedLandedCostFilterFields = map[string]string{
	"state":          "state",
	"journal_id":     "journal_id",
	"vendor_bill_id": "vendor_bill_id",
	"name":           "name",
}

func (r *PostgresRepo) ListLandedCosts(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[stock.LandedCost], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedLandedCostFilterFields, 1)
	if err != nil {
		return pagination.PageResult[stock.LandedCost]{}, platformerrors.Validation("invalid filter", err)
	}

	countQuery := "SELECT COUNT(*) FROM stock_landed_costs " + whereClause
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.PageResult[stock.LandedCost]{}, platformerrors.Internal("failed to count stock landed costs", err)
	}

	query := fmt.Sprintf(`
		SELECT id, name, date, state, picking_ids, amount_total, description, account_move_id,
		       journal_id, vendor_bill_id, company_id, created_at, updated_at
		FROM stock_landed_costs
		%s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIdx, nextIdx+1)

	args = append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return pagination.PageResult[stock.LandedCost]{}, platformerrors.Internal("failed to list stock landed costs", err)
	}
	defer rows.Close()

	var lcs []stock.LandedCost
	for rows.Next() {
		var lc stock.LandedCost
		var stateStr string
		if err := rows.Scan(
			&lc.ID, &lc.Name, &lc.Date, &stateStr, &lc.PickingIDs, &lc.AmountTotal, &lc.Description,
			&lc.AccountMoveID, &lc.JournalID, &lc.VendorBillID, &lc.CompanyID, &lc.CreatedAt, &lc.UpdatedAt,
		); err != nil {
			return pagination.PageResult[stock.LandedCost]{}, platformerrors.Internal("failed to scan stock landed cost", err)
		}
		lc.State = stock.LandedCostState(stateStr)
		lcs = append(lcs, lc)
	}

	return pagination.NewPageResult(lcs, total, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Procurement Groups
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateProcurementGroup(ctx context.Context, pg *stock.ProcurementGroup) error {
	query := `
		INSERT INTO stock_procurement_groups (name, company_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	if pg.CompanyID == 0 {
		pg.CompanyID = 1
	}
	err := r.pool.QueryRow(ctx, query, pg.Name, pg.CompanyID).Scan(&pg.ID, &pg.CreatedAt, &pg.UpdatedAt)
	if err != nil {
		return platformerrors.Internal("failed to insert procurement group", err)
	}
	return nil
}

func (r *PostgresRepo) GetProcurementGroupByID(ctx context.Context, id int64) (*stock.ProcurementGroup, error) {
	query := `
		SELECT id, name, company_id, created_at, updated_at
		FROM stock_procurement_groups
		WHERE id = $1
	`
	var pg stock.ProcurementGroup
	err := r.pool.QueryRow(ctx, query, id).Scan(&pg.ID, &pg.Name, &pg.CompanyID, &pg.CreatedAt, &pg.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("procurement group #%d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch procurement group", err)
	}
	moveIDs, err := r.procurementMoveIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	pg.MoveIDs = moveIDs
	return &pg, nil
}

func (r *PostgresRepo) GetProcurementGroupByName(ctx context.Context, name string) (*stock.ProcurementGroup, error) {
	query := `
		SELECT id, name, company_id, created_at, updated_at
		FROM stock_procurement_groups
		WHERE name = $1
		LIMIT 1
	`
	var pg stock.ProcurementGroup
	err := r.pool.QueryRow(ctx, query, name).Scan(&pg.ID, &pg.Name, &pg.CompanyID, &pg.CreatedAt, &pg.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("procurement group '%s' not found", name))
		}
		return nil, platformerrors.Internal("failed to fetch procurement group by name", err)
	}
	moveIDs, err := r.procurementMoveIDs(ctx, pg.ID)
	if err != nil {
		return nil, err
	}
	pg.MoveIDs = moveIDs
	return &pg, nil
}

func (r *PostgresRepo) procurementMoveIDs(ctx context.Context, pgID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM stock_moves WHERE procurement_group_id = $1 ORDER BY id`, pgID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list procurement group moves", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, platformerrors.Internal("failed to scan procurement group move", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
