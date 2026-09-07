package salestorage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cashflow_backend/internal/domain/sale"
	"cashflow_backend/internal/platform/audit"
	"cashflow_backend/internal/platform/database"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedSaleOrderFilterFields = map[string]string{
	"partner_id":           "partner_id",
	"state":                "state",
	"invoice_status":       "invoice_status",
	"delivery_status":      "delivery_status",
	"procurement_group_id": "procurement_group_id",
	"name":                 "name",
	"active":               "active",
	"company_id":           "company_id",
}

// PostgresRepo implements sale.Repository using PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// CreateOrder persists a new sale order and its lines within an atomic transaction.
func (r *PostgresRepo) CreateOrder(ctx context.Context, order *sale.SaleOrder) error {
	if order.CompanyID == nil {
		order.CompanyID = audit.CompanyIDFromContext(ctx)
	}
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			INSERT INTO sale_orders (
				name, partner_id, date_order, validity_date, state, invoice_status,
				pricelist_id, payment_term_id, user_id, company_id, currency, note,
				amount_untaxed, amount_tax, amount_total, delivery_status, procurement_group_id, active, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11, $12,
				$13, $14, $15, $16, $17, true, NOW(), NOW()
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
			order.State = sale.OrderStateDraft
		}
		if order.InvoiceStatus == "" {
			order.InvoiceStatus = sale.InvoiceStatusNo
		}
		if order.DeliveryStatus == "" {
			order.DeliveryStatus = "nothing"
		}

		err := tx.QueryRow(ctx, query,
			order.Name, order.PartnerID, order.DateOrder, order.ValidityDate, string(order.State), string(order.InvoiceStatus),
			order.PricelistID, order.PaymentTermID, order.UserID, order.CompanyID, order.Currency, order.Note,
			order.AmountUntaxed, order.AmountTax, order.AmountTotal, order.DeliveryStatus, order.ProcurementGroupID,
		).Scan(&order.ID, &order.Audit.CreatedAt, &order.Audit.UpdatedAt)

		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "sale_orders_name_key") {
				return platformerrors.Conflict(fmt.Sprintf("sale order name '%s' already exists", order.Name), err)
			}
			return platformerrors.Internal("failed to insert sale order", err)
		}

		lineQuery := `
			INSERT INTO sale_order_lines (
				order_id, sequence, product_id, name, product_uom_qty, product_uom,
				price_unit, discount, tax_ids, price_subtotal, price_tax, price_total,
				qty_delivered, qty_invoiced, route_id, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11, $12,
				$13, $14, $15, NOW(), NOW()
			) RETURNING id, created_at, updated_at
		`

		for i := range order.Lines {
			order.Lines[i].OrderID = order.ID
			if order.Lines[i].TaxIDs == nil {
				order.Lines[i].TaxIDs = []int64{}
			}
			err := tx.QueryRow(ctx, lineQuery,
				order.ID, order.Lines[i].Sequence, order.Lines[i].ProductID, order.Lines[i].Name,
				order.Lines[i].ProductUomQty, order.Lines[i].ProductUom, order.Lines[i].UnitPrice,
				order.Lines[i].Discount, order.Lines[i].TaxIDs, order.Lines[i].PriceSubtotal,
				order.Lines[i].PriceTax, order.Lines[i].PriceTotal, order.Lines[i].QtyDelivered,
				order.Lines[i].QtyInvoiced, order.Lines[i].RouteID,
			).Scan(&order.Lines[i].ID, &order.Lines[i].CreatedAt, &order.Lines[i].UpdatedAt)
			if err != nil {
				return platformerrors.Internal("failed to insert sale order line", err)
			}
		}

		return nil
	})
}

// GetOrderByID retrieves an order by its ID with all lines and linked invoice IDs.
func (r *PostgresRepo) GetOrderByID(ctx context.Context, id int64) (*sale.SaleOrder, error) {
	query := `
		SELECT id, name, partner_id, date_order, validity_date, state, invoice_status,
		       pricelist_id, payment_term_id, user_id, company_id, currency, COALESCE(note, ''),
		       amount_untaxed, amount_tax, amount_total, delivery_status, procurement_group_id, active, created_at, updated_at
		FROM sale_orders
		WHERE id = $1 AND active = true
	`
	args := []any{id}
	if companyID := audit.CompanyIDFromContext(ctx); companyID != nil {
		query = strings.Replace(query, "WHERE id = $1 AND active = true", "WHERE id = $1 AND active = true AND company_id = $2", 1)
		args = append(args, *companyID)
	}
	var o sale.SaleOrder
	var stateStr, invStatusStr string
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&o.ID, &o.Name, &o.PartnerID, &o.DateOrder, &o.ValidityDate, &stateStr, &invStatusStr,
		&o.PricelistID, &o.PaymentTermID, &o.UserID, &o.CompanyID, &o.Currency, &o.Note,
		&o.AmountUntaxed, &o.AmountTax, &o.AmountTotal, &o.DeliveryStatus, &o.ProcurementGroupID, &o.Active, &o.Audit.CreatedAt, &o.Audit.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("sale order with id %d not found", id))
		}
		return nil, platformerrors.Internal("failed to fetch sale order", err)
	}
	o.State = sale.SaleOrderState(stateStr)
	o.InvoiceStatus = sale.InvoiceStatus(invStatusStr)

	// Fetch lines
	lines, err := r.fetchLinesByOrderID(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Lines = lines

	// Fetch linked invoices
	invIDs, err := r.GetLinkedInvoiceIDs(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.InvoiceIDs = invIDs

	return &o, nil
}

// GetOrderByName retrieves an order by its sequence name.
func (r *PostgresRepo) GetOrderByName(ctx context.Context, name string) (*sale.SaleOrder, error) {
	query := `
		SELECT id, name, partner_id, date_order, validity_date, state, invoice_status,
		       pricelist_id, payment_term_id, user_id, company_id, currency, COALESCE(note, ''),
		       amount_untaxed, amount_tax, amount_total, delivery_status, procurement_group_id, active, created_at, updated_at
		FROM sale_orders
		WHERE name = $1 AND active = true
	`
	args := []any{name}
	if companyID := audit.CompanyIDFromContext(ctx); companyID != nil {
		query = strings.Replace(query, "WHERE name = $1 AND active = true", "WHERE name = $1 AND active = true AND company_id = $2", 1)
		args = append(args, *companyID)
	}
	var o sale.SaleOrder
	var stateStr, invStatusStr string
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&o.ID, &o.Name, &o.PartnerID, &o.DateOrder, &o.ValidityDate, &stateStr, &invStatusStr,
		&o.PricelistID, &o.PaymentTermID, &o.UserID, &o.CompanyID, &o.Currency, &o.Note,
		&o.AmountUntaxed, &o.AmountTax, &o.AmountTotal, &o.DeliveryStatus, &o.ProcurementGroupID, &o.Active, &o.Audit.CreatedAt, &o.Audit.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("sale order '%s' not found", name))
		}
		return nil, platformerrors.Internal("failed to fetch sale order", err)
	}
	o.State = sale.SaleOrderState(stateStr)
	o.InvoiceStatus = sale.InvoiceStatus(invStatusStr)

	lines, err := r.fetchLinesByOrderID(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Lines = lines

	invIDs, err := r.GetLinkedInvoiceIDs(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.InvoiceIDs = invIDs

	return &o, nil
}

// UpdateOrder updates the order master and synchronizes lines.
func (r *PostgresRepo) UpdateOrder(ctx context.Context, order *sale.SaleOrder) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		query := `
			UPDATE sale_orders
			SET name = $1, partner_id = $2, date_order = $3, validity_date = $4,
			    state = $5, invoice_status = $6, pricelist_id = $7, payment_term_id = $8,
			    user_id = $9, company_id = $10, currency = $11, note = $12,
			    amount_untaxed = $13, amount_tax = $14, amount_total = $15,
			    delivery_status = $16, procurement_group_id = $17,
			    updated_at = NOW()
			WHERE id = $18 AND active = true
			RETURNING updated_at
		`
		args := []any{
			order.Name, order.PartnerID, order.DateOrder, order.ValidityDate,
			string(order.State), string(order.InvoiceStatus), order.PricelistID, order.PaymentTermID,
			order.UserID, order.CompanyID, order.Currency, order.Note,
			order.AmountUntaxed, order.AmountTax, order.AmountTotal,
			order.DeliveryStatus, order.ProcurementGroupID, order.ID,
		}
		if companyID := audit.CompanyIDFromContext(ctx); companyID != nil {
			query = strings.Replace(query, "WHERE id = $18 AND active = true", "WHERE id = $18 AND active = true AND company_id = $19", 1)
			args = append(args, *companyID)
		}
		err := tx.QueryRow(ctx, query, args...).Scan(&order.Audit.UpdatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return platformerrors.NotFound(fmt.Sprintf("sale order with id %d not found", order.ID))
			}
			return platformerrors.Internal("failed to update sale order", err)
		}

		// Delete old lines
		if _, err := tx.Exec(ctx, `DELETE FROM sale_order_lines WHERE order_id = $1`, order.ID); err != nil {
			return platformerrors.Internal("failed to remove existing order lines", err)
		}

		// Insert updated lines
		lineQuery := `
			INSERT INTO sale_order_lines (
				order_id, sequence, product_id, name, product_uom_qty, product_uom,
				price_unit, discount, tax_ids, price_subtotal, price_tax, price_total,
				qty_delivered, qty_invoiced, route_id, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11, $12,
				$13, $14, $15, NOW(), NOW()
			) RETURNING id, created_at, updated_at
		`
		for i := range order.Lines {
			order.Lines[i].OrderID = order.ID
			if order.Lines[i].TaxIDs == nil {
				order.Lines[i].TaxIDs = []int64{}
			}
			err := tx.QueryRow(ctx, lineQuery,
				order.ID, order.Lines[i].Sequence, order.Lines[i].ProductID, order.Lines[i].Name,
				order.Lines[i].ProductUomQty, order.Lines[i].ProductUom, order.Lines[i].UnitPrice,
				order.Lines[i].Discount, order.Lines[i].TaxIDs, order.Lines[i].PriceSubtotal,
				order.Lines[i].PriceTax, order.Lines[i].PriceTotal, order.Lines[i].QtyDelivered,
				order.Lines[i].QtyInvoiced, order.Lines[i].RouteID,
			).Scan(&order.Lines[i].ID, &order.Lines[i].CreatedAt, &order.Lines[i].UpdatedAt)
			if err != nil {
				return platformerrors.Internal("failed to insert updated order line", err)
			}
		}

		return nil
	})
}

// DeleteOrder soft-deletes a sale order.
func (r *PostgresRepo) DeleteOrder(ctx context.Context, id int64) error {
	query := `
		UPDATE sale_orders
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
		return platformerrors.Internal("failed to delete sale order", err)
	}
	if res.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("sale order with id %d not found", id))
	}
	return nil
}

// ListOrders returns paginated sale orders matching filters.
func (r *PostgresRepo) ListOrders(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[sale.SaleOrder], error) {
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

	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedSaleOrderFilterFields, 1)
	if err != nil {
		return pagination.PageResult[sale.SaleOrder]{}, err
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM sale_orders %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[sale.SaleOrder]{}, platformerrors.Internal("failed to count sale orders", err)
	}

	if totalItems == 0 {
		return pagination.NewPageResult([]sale.SaleOrder{}, 0, page), nil
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, name, partner_id, date_order, validity_date, state, invoice_status,
		       pricelist_id, payment_term_id, user_id, company_id, currency, COALESCE(note, ''),
		       amount_untaxed, amount_tax, amount_total, delivery_status, procurement_group_id, active, created_at, updated_at
		FROM sale_orders
		%s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, nextIdx, nextIdx+1)

	queryArgs := append(args, page.LimitClamped(), page.Offset())
	rows, err := r.pool.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		return pagination.PageResult[sale.SaleOrder]{}, platformerrors.Internal("failed to list sale orders", err)
	}
	defer rows.Close()

	var orders []sale.SaleOrder
	for rows.Next() {
		var o sale.SaleOrder
		var stateStr, invStatusStr string
		err := rows.Scan(
			&o.ID, &o.Name, &o.PartnerID, &o.DateOrder, &o.ValidityDate, &stateStr, &invStatusStr,
			&o.PricelistID, &o.PaymentTermID, &o.UserID, &o.CompanyID, &o.Currency, &o.Note,
			&o.AmountUntaxed, &o.AmountTax, &o.AmountTotal, &o.DeliveryStatus, &o.ProcurementGroupID, &o.Active, &o.Audit.CreatedAt, &o.Audit.UpdatedAt,
		)
		if err != nil {
			return pagination.PageResult[sale.SaleOrder]{}, platformerrors.Internal("failed to scan sale order", err)
		}
		o.State = sale.SaleOrderState(stateStr)
		o.InvoiceStatus = sale.InvoiceStatus(invStatusStr)
		orders = append(orders, o)
	}

	// For each order, fetch lines and invoices
	for i := range orders {
		lines, err := r.fetchLinesByOrderID(ctx, orders[i].ID)
		if err == nil {
			orders[i].Lines = lines
		}
		invIDs, err := r.GetLinkedInvoiceIDs(ctx, orders[i].ID)
		if err == nil {
			orders[i].InvoiceIDs = invIDs
		}
	}

	return pagination.NewPageResult(orders, totalItems, page), nil
}

// NextSequence generates an incremental sequence from sale_order_seq.
func (r *PostgresRepo) NextSequence(ctx context.Context, year int) (string, error) {
	if year <= 0 {
		year = time.Now().UTC().Year()
	}

	var nextVal int64
	err := r.pool.QueryRow(ctx, "SELECT nextval('sale_order_seq')").Scan(&nextVal)
	if err != nil {
		return "", platformerrors.Internal("failed to generate next order sequence", err)
	}

	return fmt.Sprintf("SO/%d/%05d", year, nextVal), nil
}

// LinkInvoice associates an invoice with an order.
func (r *PostgresRepo) LinkInvoice(ctx context.Context, orderID int64, moveID int64) error {
	query := `
		INSERT INTO sale_order_invoices (order_id, move_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (order_id, move_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, query, orderID, moveID)
	if err != nil {
		return platformerrors.Internal("failed to link invoice to order", err)
	}
	return nil
}

// GetLinkedInvoiceIDs returns all invoice IDs linked to an order.
func (r *PostgresRepo) GetLinkedInvoiceIDs(ctx context.Context, orderID int64) ([]int64, error) {
	query := `
		SELECT move_id FROM sale_order_invoices
		WHERE order_id = $1
		ORDER BY move_id ASC
	`
	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, platformerrors.Internal("failed to query linked invoice ids", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, platformerrors.Internal("failed to scan linked invoice id", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *PostgresRepo) fetchLinesByOrderID(ctx context.Context, orderID int64) ([]sale.SaleOrderLine, error) {
	query := `
		SELECT id, order_id, sequence, product_id, name, product_uom_qty, product_uom,
		       price_unit, discount, tax_ids, price_subtotal, price_tax, price_total,
		       qty_delivered, qty_invoiced, route_id, created_at, updated_at
		FROM sale_order_lines
		WHERE order_id = $1
		ORDER BY sequence ASC, id ASC
	`
	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch sale order lines", err)
	}
	defer rows.Close()

	var lines []sale.SaleOrderLine
	for rows.Next() {
		var l sale.SaleOrderLine
		err := rows.Scan(
			&l.ID, &l.OrderID, &l.Sequence, &l.ProductID, &l.Name,
			&l.ProductUomQty, &l.ProductUom, &l.UnitPrice, &l.Discount,
			&l.TaxIDs, &l.PriceSubtotal, &l.PriceTax, &l.PriceTotal,
			&l.QtyDelivered, &l.QtyInvoiced, &l.RouteID, &l.CreatedAt, &l.UpdatedAt,
		)
		if err != nil {
			return nil, platformerrors.Internal("failed to scan sale order line", err)
		}
		lines = append(lines, l)
	}
	return lines, nil
}
