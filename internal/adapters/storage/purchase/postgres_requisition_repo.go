package purchasestorage

import (
	"context"
	"errors"
	"fmt"

	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/platform/database"
	platformerrors "cashflow_backend/internal/platform/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RequisitionPostgresRepo persists purchase requisitions and their lines in PostgreSQL.
type RequisitionPostgresRepo struct {
	pool *pgxpool.Pool
}

// NewRequisitionPostgresRepo creates a PostgreSQL-backed requisition repository.
func NewRequisitionPostgresRepo(pool *pgxpool.Pool) *RequisitionPostgresRepo {
	return &RequisitionPostgresRepo{pool: pool}
}

func (r *RequisitionPostgresRepo) CreateRequisition(ctx context.Context, req *purchase.PurchaseRequisition) error {
	if req == nil {
		return platformerrors.Validation("requisition is required", map[string]string{"requisition": "must not be nil"})
	}
	if req.State == "" {
		req.State = purchase.RequisitionDraft
	}
	if req.Type == "" {
		req.Type = purchase.RequisitionBlanketOrder
	}

	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			INSERT INTO purchase_requisitions (
				name, requisition_type, vendor_id, user_id, date_start, date_end,
				state, currency_id, company_id, description, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW()
			) RETURNING id, created_at, updated_at
		`
		if err := tx.QueryRow(ctx, query,
			req.Name, string(req.Type), req.VendorID, req.UserID, req.DateStart, req.DateEnd,
			string(req.State), req.CurrencyID, req.CompanyID, req.Description,
		).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt); err != nil {
			return platformerrors.Internal("failed to create purchase requisition", err)
		}

		lineQuery := `
			INSERT INTO purchase_requisition_lines (
				requisition_id, product_id, product_qty, product_uom, price_unit,
				schedule_date, supplier_id, description, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
			RETURNING id, created_at, updated_at
		`
		for i := range req.Lines {
			req.Lines[i].RequisitionID = req.ID
			if err := tx.QueryRow(ctx, lineQuery,
				req.ID, req.Lines[i].ProductID, req.Lines[i].ProductQty, req.Lines[i].ProductUOMID,
				req.Lines[i].PriceUnit, req.Lines[i].ScheduleDate, req.Lines[i].SupplierID,
				req.Lines[i].Description,
			).Scan(&req.Lines[i].ID, &req.Lines[i].CreatedAt, &req.Lines[i].UpdatedAt); err != nil {
				return platformerrors.Internal("failed to insert purchase requisition line", err)
			}
		}
		return nil
	})
}

func (r *RequisitionPostgresRepo) GetRequisitionByID(ctx context.Context, id int64) (*purchase.PurchaseRequisition, error) {
	query := `
		SELECT id, name, requisition_type, vendor_id, user_id, date_start, date_end,
		       state, currency_id, company_id, description, created_at, updated_at
		FROM purchase_requisitions
		WHERE id = $1
	`
	var req purchase.PurchaseRequisition
	var stateStr, typeStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&req.ID,
		&req.Name,
		&typeStr,
		&req.VendorID,
		&req.UserID,
		&req.DateStart,
		&req.DateEnd,
		&stateStr,
		&req.CurrencyID,
		&req.CompanyID,
		&req.Description,
		&req.CreatedAt,
		&req.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("purchase requisition %d not found", id))
		}
		return nil, platformerrors.Internal("failed to load purchase requisition", err)
	}
	req.Type = purchase.RequisitionType(typeStr)
	req.State = purchase.RequisitionState(stateStr)
	lines, err := r.fetchLinesByRequisitionID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	req.Lines = lines
	orderIDs, err := r.fetchPurchaseOrderIDs(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	req.PurchaseOrderIDs = orderIDs
	return &req, nil
}

func (r *RequisitionPostgresRepo) ListRequisitions(ctx context.Context, page int, limit int) ([]purchase.PurchaseRequisition, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT id, name, requisition_type, vendor_id, user_id, date_start, date_end,
		       state, currency_id, company_id, description, created_at, updated_at
		FROM purchase_requisitions
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.pool.Query(ctx, query, limit, (page-1)*limit)
	if err != nil {
		return nil, platformerrors.Internal("failed to list purchase requisitions", err)
	}
	defer rows.Close()

	var items []purchase.PurchaseRequisition
	for rows.Next() {
		var req purchase.PurchaseRequisition
		var stateStr, typeStr string
		if err := rows.Scan(
			&req.ID,
			&req.Name,
			&typeStr,
			&req.VendorID,
			&req.UserID,
			&req.DateStart,
			&req.DateEnd,
			&stateStr,
			&req.CurrencyID,
			&req.CompanyID,
			&req.Description,
			&req.CreatedAt,
			&req.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan purchase requisition", err)
		}
		req.Type = purchase.RequisitionType(typeStr)
		req.State = purchase.RequisitionState(stateStr)
		lines, err := r.fetchLinesByRequisitionID(ctx, req.ID)
		if err != nil {
			return nil, err
		}
		req.Lines = lines
		orderIDs, err := r.fetchPurchaseOrderIDs(ctx, req.ID)
		if err != nil {
			return nil, err
		}
		req.PurchaseOrderIDs = orderIDs
		items = append(items, req)
	}
	if err := rows.Err(); err != nil {
		return nil, platformerrors.Internal("failed to iterate purchase requisitions", err)
	}
	return items, nil
}

func (r *RequisitionPostgresRepo) UpdateRequisition(ctx context.Context, req *purchase.PurchaseRequisition) error {
	if req == nil {
		return platformerrors.Validation("requisition is required", map[string]string{"requisition": "must not be nil"})
	}
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			UPDATE purchase_requisitions
			SET name = $1, requisition_type = $2, vendor_id = $3, user_id = $4,
			    date_start = $5, date_end = $6, state = $7, currency_id = $8,
			    company_id = $9, description = $10, updated_at = NOW()
			WHERE id = $11
		`
		res, err := tx.Exec(ctx, query,
			req.Name, string(req.Type), req.VendorID, req.UserID, req.DateStart, req.DateEnd,
			string(req.State), req.CurrencyID, req.CompanyID, req.Description, req.ID,
		)
		if err != nil {
			return platformerrors.Internal("failed to update purchase requisition", err)
		}
		if res.RowsAffected() == 0 {
			return platformerrors.NotFound(fmt.Sprintf("purchase requisition %d not found", req.ID))
		}

		if _, err := tx.Exec(ctx, `DELETE FROM purchase_requisition_lines WHERE requisition_id = $1`, req.ID); err != nil {
			return platformerrors.Internal("failed to replace requisition lines", err)
		}
		lineQuery := `
			INSERT INTO purchase_requisition_lines (
				requisition_id, product_id, product_qty, product_uom, price_unit,
				schedule_date, supplier_id, description, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
			RETURNING id, created_at, updated_at
		`
		for i := range req.Lines {
			req.Lines[i].RequisitionID = req.ID
			if err := tx.QueryRow(ctx, lineQuery,
				req.ID, req.Lines[i].ProductID, req.Lines[i].ProductQty, req.Lines[i].ProductUOMID,
				req.Lines[i].PriceUnit, req.Lines[i].ScheduleDate, req.Lines[i].SupplierID,
				req.Lines[i].Description,
			).Scan(&req.Lines[i].ID, &req.Lines[i].CreatedAt, &req.Lines[i].UpdatedAt); err != nil {
				return platformerrors.Internal("failed to insert requisition line", err)
			}
		}
		return nil
	})
}

func (r *RequisitionPostgresRepo) DeleteRequisition(ctx context.Context, id int64) error {
	res, err := r.pool.Exec(ctx, `DELETE FROM purchase_requisitions WHERE id = $1`, id)
	if err != nil {
		return platformerrors.Internal("failed to delete purchase requisition", err)
	}
	if res.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("purchase requisition %d not found", id))
	}
	return nil
}

func (r *RequisitionPostgresRepo) fetchLinesByRequisitionID(ctx context.Context, requisitionID int64) ([]purchase.PurchaseRequisitionLine, error) {
	query := `
		SELECT id, requisition_id, product_id, product_qty, product_uom, price_unit,
		       schedule_date, supplier_id, description, created_at, updated_at
		FROM purchase_requisition_lines
		WHERE requisition_id = $1
		ORDER BY id ASC
	`
	rows, err := r.pool.Query(ctx, query, requisitionID)
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch requisition lines", err)
	}
	defer rows.Close()

	var lines []purchase.PurchaseRequisitionLine
	for rows.Next() {
		var l purchase.PurchaseRequisitionLine
		if err := rows.Scan(
			&l.ID,
			&l.RequisitionID,
			&l.ProductID,
			&l.ProductQty,
			&l.ProductUOMID,
			&l.PriceUnit,
			&l.ScheduleDate,
			&l.SupplierID,
			&l.Description,
			&l.CreatedAt,
			&l.UpdatedAt,
		); err != nil {
			return nil, platformerrors.Internal("failed to scan requisition line", err)
		}
		lines = append(lines, l)
	}
	if err := rows.Err(); err != nil {
		return nil, platformerrors.Internal("failed to iterate requisition lines", err)
	}
	return lines, nil
}

func (r *RequisitionPostgresRepo) fetchPurchaseOrderIDs(ctx context.Context, requisitionID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id
		FROM purchase_orders
		WHERE requisition_id = $1 AND active = true
		ORDER BY id ASC
	`, requisitionID)
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch linked purchase orders", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, platformerrors.Internal("failed to scan linked purchase order", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, platformerrors.Internal("failed to iterate linked purchase orders", err)
	}
	return ids, nil
}
