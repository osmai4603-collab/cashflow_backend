package purchasestorage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cashflow_backend/internal/domain/purchase"
	"cashflow_backend/internal/platform/audit"
	"cashflow_backend/internal/platform/database"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedPurchaseOrderFilterFields = map[string]string{
	"partner_id":           "partner_id",
	"state":                "state",
	"invoice_status":       "invoice_status",
	"receipt_status":       "receipt_status",
	"procurement_group_id": "procurement_group_id",
	"name":                 "name",
	"active":               "active",
	"company_id":           "company_id",
}

// PostgresRepo implements purchase.Repository using PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// CreateOrder persists a new purchase order and its lines within an atomic transaction.
func (r *PostgresRepo) CreateOrder(ctx context.Context, order *purchase.PurchaseOrder) error {
	if order.CompanyID == nil {
		order.CompanyID = audit.CompanyIDFromContext(ctx)
	}
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			INSERT INTO purchase_orders (
				name, partner_id, date_order, date_planned, state, invoice_status,
				payment_term_id, user_id, company_id, currency, note,
				amount_untaxed, amount_tax, amount_total, receipt_status, procurement_group_id, active, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11,
				$12, $13, $14, $15, $16, true, NOW(), NOW()
			) RETURNING id, created_at, updated_at
		`
		now := time.Now().UTC()
		if order.DateOrder.IsZero() {
			order.DateOrder = now
		}
		if order.Currency == "" {
			order.Currency = "USD"
		}
		if order.State == "" {
			order.State = purchase.OrderStateDraft
		}
		if order.InvoiceStatus == "" {
			order.InvoiceStatus = purchase.InvoiceStatusNo
		}
		if order.ReceiptStatus == "" {
			order.ReceiptStatus = "nothing"
		}

		err := tx.QueryRow(ctx, query,
			order.Name, order.PartnerID, order.DateOrder, order.DatePlanned, string(order.State), string(order.InvoiceStatus),
			order.PaymentTermID, order.UserID, order.CompanyID, order.Currency, order.Note,
			order.AmountUntaxed, order.AmountTax, order.AmountTotal, order.ReceiptStatus, order.ProcurementGroupID,
		).Scan(&order.ID, &order.Audit.CreatedAt, &order.Audit.UpdatedAt)

		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "purchase_orders_name_key") {
				return platformerrors.Conflict(fmt.Sprintf("purchase order name '%s' already exists", order.Name), err)
			}
			return platformerrors.Internal("failed to insert purchase order", err)
		}

		lineQuery := `
			INSERT INTO purchase_order_lines (
				order_id, sequence, product_id, name, product_qty, product_uom,
				price_unit, discount, tax_ids, price_subtotal, price_tax, price_total,
				qty_received, qty_invoiced, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11, $12,
				$13, $14, NOW(), NOW()
			) RETURNING id, created_at, updated_at
		`

		for i := range order.Lines {
			order.Lines[i].OrderID = order.ID
			if order.Lines[i].TaxIDs == nil {
				order.Lines[i].TaxIDs = []int64{}
			}
			err := tx.QueryRow(ctx, lineQuery,
				order.ID, order.Lines[i].Sequence, order.Lines[i].ProductID, order.Lines[i].Name,
				order.Lines[i].ProductQty, order.Lines[i].ProductUom, order.Lines[i].UnitPrice,
				order.Lines[i].Discount, order.Lines[i].TaxIDs, order.Lines[i].PriceSubtotal,
				order.Lines[i].PriceTax, order.Lines[i].PriceTotal, order.Lines[i].QtyReceived,
				order.Lines[i].QtyInvoiced,
			).Scan(&order.Lines[i].ID, &order.Lines[i].CreatedAt, &order.Lines[i].UpdatedAt)
			if err != nil {
				return platformerrors.Internal("failed to insert purchase order line", err)
			}
		}

		return nil
	})
}

// GetOrderByID retrieves an order by its ID with all lines and linked bill IDs.
func (r *PostgresRepo) GetOrderByID(ctx context.Context, id int64) (*purchase.PurchaseOrder, error) {
	query := `
		SELECT id, name, partner_id, date_order, date_planned, state, invoice_status,
		       payment_term_id, user_id, company_id, currency, COALESCE(note, ''),
		       amount_untaxed, amount_tax, amount_total, receipt_status, procurement_group_id, active, created_at, updated_at
		FROM purchase_orders
		WHERE id = $1 AND active = true
	`
	args := []any{id}
	if companyID := audit.CompanyIDFromContext(ctx); companyID != nil {
		query = strings.Replace(query, "WHERE id = $1 AND active = true", "WHERE id = $1 AND active = true AND company_id = $2", 1)
		args = append(args, *companyID)
	}
	var o purchase.PurchaseOrder
	var stateStr, invStatusStr string
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&o.ID, &o.Name, &o.PartnerID, &o.DateOrder, &o.DatePlanned, &stateStr, &invStatusStr,
		&o.PaymentTermID, &o.UserID, &o.CompanyID, &o.Currency, &o.Note,
		&o.AmountUntaxed, &o.AmountTax, &o.AmountTotal, &o.ReceiptStatus, &o.ProcurementGroupID, &o.Active, &o.Audit.CreatedAt, &o.Audit.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("purchase order with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch purchase order", err)
	}
	o.State = purchase.PurchaseOrderState(stateStr)
	o.InvoiceStatus = purchase.InvoiceStatus(invStatusStr)

	// Fetch lines
	lines, err := r.fetchLinesByOrderID(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Lines = lines

	// Fetch linked bills
	billIDs, err := r.GetLinkedBillIDs(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.BillIDs = billIDs

	return &o, nil
}

// GetOrderByName retrieves an order by its sequence name.
func (r *PostgresRepo) GetOrderByName(ctx context.Context, name string) (*purchase.PurchaseOrder, error) {
	query := `
		SELECT id, name, partner_id, date_order, date_planned, state, invoice_status,
		       payment_term_id, user_id, company_id, currency, COALESCE(note, ''),
		       amount_untaxed, amount_tax, amount_total, receipt_status, procurement_group_id, active, created_at, updated_at
		FROM purchase_orders
		WHERE name = $1 AND active = true
	`
	args := []any{name}
	if companyID := audit.CompanyIDFromContext(ctx); companyID != nil {
		query = strings.Replace(query, "WHERE name = $1 AND active = true", "WHERE name = $1 AND active = true AND company_id = $2", 1)
		args = append(args, *companyID)
	}
	var o purchase.PurchaseOrder
	var stateStr, invStatusStr string
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&o.ID, &o.Name, &o.PartnerID, &o.DateOrder, &o.DatePlanned, &stateStr, &invStatusStr,
		&o.PaymentTermID, &o.UserID, &o.CompanyID, &o.Currency, &o.Note,
		&o.AmountUntaxed, &o.AmountTax, &o.AmountTotal, &o.ReceiptStatus, &o.ProcurementGroupID, &o.Active, &o.Audit.CreatedAt, &o.Audit.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("purchase order '%s' not found", name))
		}
		return nil, platformerrors.Internal("failed to fetch purchase order", err)
	}
	o.State = purchase.PurchaseOrderState(stateStr)
	o.InvoiceStatus = purchase.InvoiceStatus(invStatusStr)

	lines, err := r.fetchLinesByOrderID(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Lines = lines

	billIDs, err := r.GetLinkedBillIDs(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.BillIDs = billIDs

	return &o, nil
}

// UpdateOrder updates the order master and synchronizes lines.
func (r *PostgresRepo) UpdateOrder(ctx context.Context, order *purchase.PurchaseOrder) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			UPDATE purchase_orders
			SET name = $1, partner_id = $2, date_order = $3, date_planned = $4,
			    state = $5, invoice_status = $6, payment_term_id = $7,
			    user_id = $8, company_id = $9, currency = $10, note = $11,
			    amount_untaxed = $12, amount_tax = $13, amount_total = $14,
			    receipt_status = $15, procurement_group_id = $16,
			    updated_at = NOW()
			WHERE id = $17 AND active = true
			RETURNING updated_at
		`
		args := []any{
			order.Name, order.PartnerID, order.DateOrder, order.DatePlanned,
			string(order.State), string(order.InvoiceStatus), order.PaymentTermID,
			order.UserID, order.CompanyID, order.Currency, order.Note,
			order.AmountUntaxed, order.AmountTax, order.AmountTotal,
			order.ReceiptStatus, order.ProcurementGroupID, order.ID,
		}
		if companyID := audit.CompanyIDFromContext(ctx); companyID != nil {
			query = strings.Replace(query, "WHERE id = $17 AND active = true", "WHERE id = $17 AND active = true AND company_id = $18", 1)
			args = append(args, *companyID)
		}
		err := tx.QueryRow(ctx, query, args...).Scan(&order.Audit.UpdatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return platformerrors.NotFound(fmt.Sprintf("purchase order with id %d not found", order.ID))
			}
			return platformerrors.Internal("failed to update purchase order", err)
		}

		// Delete old lines
		if _, err := tx.Exec(ctx, `DELETE FROM purchase_order_lines WHERE order_id = $1`, order.ID); err != nil {
			return platformerrors.Internal("failed to remove existing order lines", err)
		}

		// Insert updated lines
		lineQuery := `
			INSERT INTO purchase_order_lines (
				order_id, sequence, product_id, name, product_qty, product_uom,
				price_unit, discount, tax_ids, price_subtotal, price_tax, price_total,
				qty_received, qty_invoiced, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11, $12,
				$13, $14, NOW(), NOW()
			) RETURNING id, created_at, updated_at
		`
		for i := range order.Lines {
			order.Lines[i].OrderID = order.ID
			if order.Lines[i].TaxIDs == nil {
				order.Lines[i].TaxIDs = []int64{}
			}
			err := tx.QueryRow(ctx, lineQuery,
				order.ID, order.Lines[i].Sequence, order.Lines[i].ProductID, order.Lines[i].Name,
				order.Lines[i].ProductQty, order.Lines[i].ProductUom, order.Lines[i].UnitPrice,
				order.Lines[i].Discount, order.Lines[i].TaxIDs, order.Lines[i].PriceSubtotal,
				order.Lines[i].PriceTax, order.Lines[i].PriceTotal, order.Lines[i].QtyReceived,
				order.Lines[i].QtyInvoiced,
			).Scan(&order.Lines[i].ID, &order.Lines[i].CreatedAt, &order.Lines[i].UpdatedAt)
			if err != nil {
				return platformerrors.Internal("failed to insert updated purchase order line", err)
			}
		}

		return nil
	})
}

// DeleteOrder soft-deletes a purchase order.
func (r *PostgresRepo) DeleteOrder(ctx context.Context, id int64) error {
	query := `
		UPDATE purchase_orders
		SET active = false, updated_at = NOW()
		WHERE id = $1 AND active = true
	`
	args := []any{id}
	if companyID := audit.CompanyIDFromContext(ctx); companyID != nil {
		query = strings.Replace(query, "WHERE id = $1 AND active = true", "WHERE id = $1 AND active = true AND company_id = $2", 1)
		args = append(args, *companyID)
	}
	res, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return platformerrors.Internal("failed to delete purchase order", err)
	}
	if res.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("purchase order with id %d not found", id))
	}
	return nil
}

// ListOrders returns paginated purchase orders matching filters.
func (r *PostgresRepo) ListOrders(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[purchase.PurchaseOrder], error) {
	if companyID := audit.CompanyIDFromContext(ctx); companyID != nil {
		if f == nil {
			f = filter.NewFilter()
		}
		f.Add("company_id", filter.OpEqual, *companyID)
	}
	var activeExplicit bool
	if f != nil {
		for _, c := range f.Criteria {
			if strings.EqualFold(c.Field, "active") {
				activeExplicit = true
				break
			}
		}
	}
	if !activeExplicit {
		if f == nil {
			f = filter.NewFilter()
		}
		f.Add("active", filter.OpEqual, true)
	}

	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedPurchaseOrderFilterFields, 1)
	if err != nil {
		return pagination.PageResult[purchase.PurchaseOrder]{}, err
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM purchase_orders %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[purchase.PurchaseOrder]{}, platformerrors.Internal("failed to count purchase orders", err)
	}

	if totalItems == 0 {
		return pagination.NewPageResult([]purchase.PurchaseOrder{}, 0, page), nil
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, name, partner_id, date_order, date_planned, state, invoice_status,
		       payment_term_id, user_id, company_id, currency, COALESCE(note, ''),
		       amount_untaxed, amount_tax, amount_total, receipt_status, procurement_group_id, active, created_at, updated_at
		FROM purchase_orders
		%s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIdx, nextIdx+1)

	queryArgs := append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		return pagination.PageResult[purchase.PurchaseOrder]{}, platformerrors.Internal("failed to list purchase orders", err)
	}
	defer rows.Close()

	var orders []purchase.PurchaseOrder
	for rows.Next() {
		var o purchase.PurchaseOrder
		var stateStr, invStatusStr string
		err := rows.Scan(
			&o.ID, &o.Name, &o.PartnerID, &o.DateOrder, &o.DatePlanned, &stateStr, &invStatusStr,
			&o.PaymentTermID, &o.UserID, &o.CompanyID, &o.Currency, &o.Note,
			&o.AmountUntaxed, &o.AmountTax, &o.AmountTotal, &o.ReceiptStatus, &o.ProcurementGroupID, &o.Active, &o.Audit.CreatedAt, &o.Audit.UpdatedAt,
		)
		if err != nil {
			return pagination.PageResult[purchase.PurchaseOrder]{}, platformerrors.Internal("failed to scan purchase order", err)
		}
		o.State = purchase.PurchaseOrderState(stateStr)
		o.InvoiceStatus = purchase.InvoiceStatus(invStatusStr)
		orders = append(orders, o)
	}

	// For each order, fetch lines and bills
	for i := range orders {
		lines, err := r.fetchLinesByOrderID(ctx, orders[i].ID)
		if err == nil {
			orders[i].Lines = lines
		}
		billIDs, err := r.GetLinkedBillIDs(ctx, orders[i].ID)
		if err == nil {
			orders[i].BillIDs = billIDs
		}
	}

	return pagination.NewPageResult(orders, totalItems, page), nil
}

// NextSequence generates an incremental sequence from purchase_order_seq.
func (r *PostgresRepo) NextSequence(ctx context.Context, year int) (string, error) {
	if year <= 0 {
		year = time.Now().UTC().Year()
	}

	var nextVal int64
	err := r.pool.QueryRow(ctx, "SELECT nextval('purchase_order_seq')").Scan(&nextVal)
	if err != nil {
		return "", platformerrors.Internal("failed to generate next purchase order sequence", err)
	}

	return fmt.Sprintf("PO/%d/%05d", year, nextVal), nil
}

// LinkBill associates a vendor bill with an order.
func (r *PostgresRepo) LinkBill(ctx context.Context, orderID int64, moveID int64) error {
	query := `
		INSERT INTO purchase_order_invoices (order_id, move_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (order_id, move_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, query, orderID, moveID)
	if err != nil {
		return platformerrors.Internal("failed to link vendor bill to purchase order", err)
	}
	return nil
}

// GetLinkedBillIDs returns all bill move IDs linked to an order.
func (r *PostgresRepo) GetLinkedBillIDs(ctx context.Context, orderID int64) ([]int64, error) {
	query := `
		SELECT move_id FROM purchase_order_invoices
		WHERE order_id = $1
		ORDER BY move_id ASC
	`
	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, platformerrors.Internal("failed to query linked vendor bill ids", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, platformerrors.Internal("failed to scan linked vendor bill id", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *PostgresRepo) fetchLinesByOrderID(ctx context.Context, orderID int64) ([]purchase.PurchaseOrderLine, error) {
	query := `
		SELECT id, order_id, sequence, product_id, name, product_qty, product_uom,
		       price_unit, discount, tax_ids, price_subtotal, price_tax, price_total,
		       qty_received, qty_invoiced, created_at, updated_at
		FROM purchase_order_lines
		WHERE order_id = $1
		ORDER BY sequence ASC, id ASC
	`
	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch purchase order lines", err)
	}
	defer rows.Close()

	var lines []purchase.PurchaseOrderLine
	for rows.Next() {
		var l purchase.PurchaseOrderLine
		err := rows.Scan(
			&l.ID, &l.OrderID, &l.Sequence, &l.ProductID, &l.Name,
			&l.ProductQty, &l.ProductUom, &l.UnitPrice, &l.Discount,
			&l.TaxIDs, &l.PriceSubtotal, &l.PriceTax, &l.PriceTotal,
			&l.QtyReceived, &l.QtyInvoiced, &l.CreatedAt, &l.UpdatedAt,
		)
		if err != nil {
			return nil, platformerrors.Internal("failed to scan purchase order line", err)
		}
		lines = append(lines, l)
	}
	return lines, nil
}
