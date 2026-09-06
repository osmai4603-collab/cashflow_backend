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
	"picking_type": "picking_type",
	"state":        "state",
	"partner_id":   "partner_id",
	"origin":       "origin",
	"active":       "active",
}

var allowedMoveFilterFields = map[string]string{
	"product_id": "product_id",
	"picking_id": "picking_id",
	"state":      "state",
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
			name, complete_name, usage, parent_id, scrap_location, return_location, company_id, active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, true, NOW(), NOW()
		) RETURNING id, created_at, updated_at
	`
	if loc.Usage == "" {
		loc.Usage = stock.LocationUsageInternal
	}
	if loc.CompleteName == "" {
		loc.CompleteName = loc.Name
	}

	err := r.pool.QueryRow(ctx, query,
		loc.Name, loc.CompleteName, string(loc.Usage), loc.ParentID, loc.ScrapLocation, loc.ReturnLocation, loc.CompanyID,
	).Scan(&loc.ID, &loc.CreatedAt, &loc.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to insert stock location", err)
	}
	loc.Active = true
	return nil
}

func (r *PostgresRepo) GetLocationByID(ctx context.Context, id int64) (*stock.StockLocation, error) {
	query := `
		SELECT id, name, complete_name, usage, parent_id, scrap_location, return_location, company_id, active, created_at, updated_at
		FROM stock_locations
		WHERE id = $1 AND active = true
	`
	var loc stock.StockLocation
	var usageStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&loc.ID, &loc.Name, &loc.CompleteName, &usageStr, &loc.ParentID,
		&loc.ScrapLocation, &loc.ReturnLocation, &loc.CompanyID, &loc.Active, &loc.CreatedAt, &loc.UpdatedAt,
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
		SELECT id, name, complete_name, usage, parent_id, scrap_location, return_location, company_id, active, created_at, updated_at
		FROM stock_locations
		WHERE LOWER(name) = LOWER($1) AND active = true
		LIMIT 1
	`
	var loc stock.StockLocation
	var usageStr string
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&loc.ID, &loc.Name, &loc.CompleteName, &usageStr, &loc.ParentID,
		&loc.ScrapLocation, &loc.ReturnLocation, &loc.CompanyID, &loc.Active, &loc.CreatedAt, &loc.UpdatedAt,
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
		SELECT id, name, complete_name, usage, parent_id, scrap_location, return_location, company_id, active, created_at, updated_at
		FROM stock_locations
		WHERE usage = $1 AND active = true
		ORDER BY id ASC
		LIMIT 1
	`
	var loc stock.StockLocation
	var usageStr string
	err := r.pool.QueryRow(ctx, query, string(usage)).Scan(
		&loc.ID, &loc.Name, &loc.CompleteName, &usageStr, &loc.ParentID,
		&loc.ScrapLocation, &loc.ReturnLocation, &loc.CompanyID, &loc.Active, &loc.CreatedAt, &loc.UpdatedAt,
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
		    return_location = $6, company_id = $7, updated_at = NOW()
		WHERE id = $8 AND active = true
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		loc.Name, loc.CompleteName, string(loc.Usage), loc.ParentID, loc.ScrapLocation,
		loc.ReturnLocation, loc.CompanyID, loc.ID,
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
		SELECT id, name, complete_name, usage, parent_id, scrap_location, return_location, company_id, active, created_at, updated_at
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
			&loc.ScrapLocation, &loc.ReturnLocation, &loc.CompanyID, &loc.Active, &loc.CreatedAt, &loc.UpdatedAt,
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
				scheduled_date, date_done, origin, source_order_id, company_id, note, active, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, true, NOW(), NOW()
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
			picking.Origin, picking.SourceOrderID, picking.CompanyID, picking.Note,
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
				location_id, location_dest_id, state, sale_line_id, purchase_line_id, date, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW()
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
				m.LocationID, m.LocationDestID, string(m.State), m.SaleLineID, m.PurchaseLineID, m.Date,
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
		       scheduled_date, date_done, origin, source_order_id, company_id, note, active, created_at, updated_at
		FROM stock_pickings
		WHERE id = $1 AND active = true
	`
	var p stock.StockPicking
	var ptStr, stateStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &ptStr, &stateStr, &p.PartnerID, &p.LocationID, &p.LocationDestID,
		&p.ScheduledDate, &p.DateDone, &p.Origin, &p.SourceOrderID, &p.CompanyID, &p.Note, &p.Active,
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
		       scheduled_date, date_done, origin, source_order_id, company_id, note, active, created_at, updated_at
		FROM stock_pickings
		WHERE name = $1 AND active = true
		LIMIT 1
	`
	var p stock.StockPicking
	var ptStr, stateStr string
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&p.ID, &p.Name, &ptStr, &stateStr, &p.PartnerID, &p.LocationID, &p.LocationDestID,
		&p.ScheduledDate, &p.DateDone, &p.Origin, &p.SourceOrderID, &p.CompanyID, &p.Note, &p.Active,
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
			    scheduled_date = $6, date_done = $7, origin = $8, source_order_id = $9, company_id = $10,
			    note = $11, updated_at = NOW()
			WHERE id = $12 AND active = true
			RETURNING updated_at
		`
		err := tx.QueryRow(ctx, query,
			picking.Name, string(picking.State), picking.PartnerID, picking.LocationID, picking.LocationDestID,
			picking.ScheduledDate, picking.DateDone, picking.Origin, picking.SourceOrderID, picking.CompanyID,
			picking.Note, picking.ID,
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
				location_id, location_dest_id, state, sale_line_id, purchase_line_id, date, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW()
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
				m.LocationID, m.LocationDestID, string(m.State), m.SaleLineID, m.PurchaseLineID, m.Date,
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
		       scheduled_date, date_done, origin, source_order_id, company_id, note, active, created_at, updated_at
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
			&p.ScheduledDate, &p.DateDone, &p.Origin, &p.SourceOrderID, &p.CompanyID, &p.Note, &p.Active,
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
			location_id, location_dest_id, state, sale_line_id, purchase_line_id, date, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW()
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
		string(move.State), move.SaleLineID, move.PurchaseLineID, move.Date,
	).Scan(&move.ID, &move.CreatedAt, &move.UpdatedAt)

	if err != nil {
		return platformerrors.Internal("failed to create stock move", err)
	}
	return nil
}

func (r *PostgresRepo) GetMoveByID(ctx context.Context, id int64) (*stock.StockMove, error) {
	query := `
		SELECT id, picking_id, sequence, name, product_id, product_uom, product_qty, quantity_done,
		       location_id, location_dest_id, state, sale_line_id, purchase_line_id, date, created_at, updated_at
		FROM stock_moves
		WHERE id = $1
	`
	var m stock.StockMove
	var stateStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.PickingID, &m.Sequence, &m.Name, &m.ProductID, &m.ProductUom,
		&m.ProductQty, &m.QuantityDone, &m.LocationID, &m.LocationDestID, &stateStr,
		&m.SaleLineID, &m.PurchaseLineID, &m.Date, &m.CreatedAt, &m.UpdatedAt,
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
		    date = $12, updated_at = NOW()
		WHERE id = $13
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		move.Sequence, move.Name, move.ProductID, move.ProductUom, move.ProductQty, move.QuantityDone,
		move.LocationID, move.LocationDestID, string(move.State), move.SaleLineID, move.PurchaseLineID,
		move.Date, move.ID,
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
		       location_id, location_dest_id, state, sale_line_id, purchase_line_id, date, created_at, updated_at
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
			&m.SaleLineID, &m.PurchaseLineID, &m.Date, &m.CreatedAt, &m.UpdatedAt,
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
		       location_id, location_dest_id, state, sale_line_id, purchase_line_id, date, created_at, updated_at
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
			&m.SaleLineID, &m.PurchaseLineID, &m.Date, &m.CreatedAt, &m.UpdatedAt,
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
