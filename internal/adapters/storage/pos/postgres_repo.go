package posstorage

import (
	"context"
	"fmt"

	"cashflow_backend/internal/domain/pos"
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

func (repo *PostgresRepo) CreateConfig(ctx context.Context, config *pos.PosConfig) error {
	return repo.pool.QueryRow(ctx, `
		INSERT INTO pos_configs (name, warehouse_id, stock_location_id, journal_id, invoice_journal_id, module_pos_restaurant, update_stock_at_closing, allow_discount, manual_discount_limit, active, company_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		config.Name, config.WarehouseID, config.StockLocationID, config.JournalID, config.InvoiceJournalID,
		config.ModulePosRestaurant, config.UpdateStockAtClosing, config.AllowDiscount, config.ManualDiscountLimit, config.Active, config.CompanyID,
	).Scan(&config.ID)
}

func (repo *PostgresRepo) GetConfig(ctx context.Context, id int64) (*pos.PosConfig, error) {
	config := &pos.PosConfig{ID: id}
	err := repo.pool.QueryRow(ctx, `SELECT name, warehouse_id, stock_location_id, journal_id, invoice_journal_id, module_pos_restaurant, update_stock_at_closing, allow_discount, manual_discount_limit, active, company_id, created_at, updated_at FROM pos_configs WHERE id=$1`, id).Scan(
		&config.Name, &config.WarehouseID, &config.StockLocationID, &config.JournalID, &config.InvoiceJournalID, &config.ModulePosRestaurant,
		&config.UpdateStockAtClosing, &config.AllowDiscount, &config.ManualDiscountLimit, &config.Active, &config.CompanyID, &config.CreatedAt, &config.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, platformerrors.NotFound(fmt.Sprintf("pos config with ID %d not found", id))
	}
	return config, err
}

func (repo *PostgresRepo) CreateSession(ctx context.Context, session *pos.PosSession) error {
	return repo.pool.QueryRow(ctx, `
		INSERT INTO pos_sessions (config_id, user_id, name, state, start_at, cash_register_balance_start, company_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, session.ConfigID, session.UserID, session.Name, session.State, session.StartAt, session.CashRegisterBalanceStart, session.CompanyID).Scan(&session.ID)
}

func (repo *PostgresRepo) GetSession(ctx context.Context, id int64) (*pos.PosSession, error) {
	session := &pos.PosSession{ID: id}
	err := repo.pool.QueryRow(ctx, `SELECT config_id, user_id, name, state, start_at, stop_at, cash_register_balance_start, cash_register_balance_end, cash_register_balance_real, cash_register_difference, total_orders_count, total_payments_amount, stock_picking_id, account_move_id, company_id FROM pos_sessions WHERE id=$1`, id).Scan(
		&session.ConfigID, &session.UserID, &session.Name, &session.State, &session.StartAt, &session.StopAt, &session.CashRegisterBalanceStart,
		&session.CashRegisterBalanceEnd, &session.CashRegisterBalanceReal, &session.CashRegisterDifference, &session.TotalOrdersCount, &session.TotalPaymentsAmount,
		&session.StockPickingID, &session.AccountMoveID, &session.CompanyID,
	)
	if err == pgx.ErrNoRows {
		return nil, platformerrors.NotFound(fmt.Sprintf("pos session with ID %d not found", id))
	}
	return session, err
}

func (repo *PostgresRepo) UpdateSession(ctx context.Context, session *pos.PosSession) error {
	result, err := repo.pool.Exec(ctx, `UPDATE pos_sessions SET state=$1, stop_at=$2, cash_register_balance_start=$3, cash_register_balance_end=$4, cash_register_balance_real=$5, cash_register_difference=$6, total_orders_count=$7, total_payments_amount=$8, stock_picking_id=$9, account_move_id=$10, updated_at=NOW() WHERE id=$11`, session.State, session.StopAt, session.CashRegisterBalanceStart, session.CashRegisterBalanceEnd, session.CashRegisterBalanceReal, session.CashRegisterDifference, session.TotalOrdersCount, session.TotalPaymentsAmount, session.StockPickingID, session.AccountMoveID, session.ID)
	if err == nil && result.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("pos session with ID %d not found", session.ID))
	}
	return err
}

func (repo *PostgresRepo) CreateOrder(ctx context.Context, order *pos.PosOrder) error {
	transaction, err := repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer transaction.Rollback(ctx)
	if err := transaction.QueryRow(ctx, `INSERT INTO pos_orders (name, client_uuid, session_id, partner_id, user_id, table_id, customer_count, state, amount_untaxed, amount_tax, amount_total, amount_paid, amount_return, tip_amount, company_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id`, order.Name, order.ClientUUID, order.SessionID, order.PartnerID, order.UserID, order.TableID, order.CustomerCount, order.State, order.AmountUntaxed, order.AmountTax, order.AmountTotal, order.AmountPaid, order.AmountReturn, order.TipAmount, order.CompanyID).Scan(&order.ID); err != nil {
		return err
	}
	for index := range order.Lines {
		line := &order.Lines[index]
		if err := transaction.QueryRow(ctx, `INSERT INTO pos_order_lines (order_id, product_id, qty, price_unit, discount, tax_rate, price_subtotal, price_subtotal_incl, customer_note) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, order.ID, line.ProductID, line.Qty, line.PriceUnit, line.Discount, line.TaxRate, line.PriceSubtotal, line.PriceSubtotalIncl, line.CustomerNote).Scan(&line.ID); err != nil {
			return err
		}
	}
	return transaction.Commit(ctx)
}

func (repo *PostgresRepo) GetOrderByClientUUID(ctx context.Context, clientUUID string) (*pos.PosOrder, error) {
	order := &pos.PosOrder{ClientUUID: clientUUID}
	err := repo.pool.QueryRow(ctx, `SELECT id, name, session_id, partner_id, user_id, table_id, customer_count, state, amount_untaxed, amount_tax, amount_total, amount_paid, amount_return, tip_amount, company_id, created_at, updated_at FROM pos_orders WHERE client_uuid=$1`, clientUUID).Scan(
		&order.ID, &order.Name, &order.SessionID, &order.PartnerID, &order.UserID, &order.TableID, &order.CustomerCount, &order.State,
		&order.AmountUntaxed, &order.AmountTax, &order.AmountTotal, &order.AmountPaid, &order.AmountReturn, &order.TipAmount, &order.CompanyID, &order.CreatedAt, &order.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, platformerrors.NotFound("pos order not found")
	}
	return order, err
}

func (repo *PostgresRepo) UpdateOrder(ctx context.Context, order *pos.PosOrder) error {
	result, err := repo.pool.Exec(ctx, `UPDATE pos_orders SET state=$1, amount_untaxed=$2, amount_tax=$3, amount_total=$4, amount_paid=$5, amount_return=$6, tip_amount=$7, updated_at=NOW() WHERE client_uuid=$8`, order.State, order.AmountUntaxed, order.AmountTax, order.AmountTotal, order.AmountPaid, order.AmountReturn, order.TipAmount, order.ClientUUID)
	if err == nil && result.RowsAffected() == 0 {
		return platformerrors.NotFound("pos order not found")
	}
	return err
}

func (repo *PostgresRepo) CreateCashMovement(ctx context.Context, movement *pos.CashInOutMovement) error {
	return repo.pool.QueryRow(ctx, `INSERT INTO pos_cash_movements (session_id, type, amount, reason, user_id) VALUES ($1,$2,$3,$4,$5) RETURNING id`, movement.SessionID, movement.Type, movement.Amount, movement.Reason, movement.UserID).Scan(&movement.ID)
}

func (repo *PostgresRepo) SaveSyncResult(ctx context.Context, result *pos.SyncResult) error {
	_, err := repo.pool.Exec(ctx, `INSERT INTO pos_sync_batches (session_id, idempotency_key, accepted_count) VALUES ($1,$2,$3)`, result.SessionID, result.IdempotencyKey, result.Accepted)
	return err
}

func (repo *PostgresRepo) GetSyncResult(ctx context.Context, key string) (*pos.SyncResult, error) {
	result := &pos.SyncResult{IdempotencyKey: key}
	err := repo.pool.QueryRow(ctx, `SELECT session_id, accepted_count FROM pos_sync_batches WHERE idempotency_key=$1`, key).Scan(&result.SessionID, &result.Accepted)
	if err == pgx.ErrNoRows {
		return nil, platformerrors.NotFound("pos sync result not found")
	}
	return result, err
}
