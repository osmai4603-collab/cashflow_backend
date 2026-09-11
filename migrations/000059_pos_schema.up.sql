CREATE TABLE IF NOT EXISTS pos_configs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    warehouse_id BIGINT NOT NULL REFERENCES warehouses(id),
    stock_location_id BIGINT NOT NULL REFERENCES stock_locations(id),
    journal_id BIGINT NOT NULL REFERENCES account_journals(id),
    invoice_journal_id BIGINT REFERENCES account_journals(id),
    module_pos_restaurant BOOLEAN NOT NULL DEFAULT FALSE,
    update_stock_at_closing BOOLEAN NOT NULL DEFAULT TRUE,
    allow_discount BOOLEAN NOT NULL DEFAULT TRUE,
    manual_discount_limit NUMERIC(5,2) NOT NULL DEFAULT 100.00,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    company_id BIGINT NOT NULL REFERENCES companies(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pos_payment_methods (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    journal_id BIGINT NOT NULL REFERENCES account_journals(id),
    is_cash_count BOOLEAN NOT NULL DEFAULT FALSE,
    company_id BIGINT NOT NULL REFERENCES companies(id),
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS pos_config_payment_method_rel (
    config_id BIGINT NOT NULL REFERENCES pos_configs(id) ON DELETE CASCADE,
    payment_method_id BIGINT NOT NULL REFERENCES pos_payment_methods(id) ON DELETE CASCADE,
    PRIMARY KEY (config_id, payment_method_id)
);

CREATE TABLE IF NOT EXISTS pos_sessions (
    id BIGSERIAL PRIMARY KEY,
    config_id BIGINT NOT NULL REFERENCES pos_configs(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    name VARCHAR(64) NOT NULL UNIQUE,
    state VARCHAR(32) NOT NULL DEFAULT 'opening_control',
    start_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stop_at TIMESTAMPTZ,
    cash_register_balance_start NUMERIC(15,4) NOT NULL DEFAULT 0,
    cash_register_balance_end NUMERIC(15,4) NOT NULL DEFAULT 0,
    cash_register_balance_real NUMERIC(15,4) NOT NULL DEFAULT 0,
    cash_register_difference NUMERIC(15,4) NOT NULL DEFAULT 0,
    total_orders_count INT NOT NULL DEFAULT 0,
    total_payments_amount NUMERIC(15,4) NOT NULL DEFAULT 0,
    stock_picking_id BIGINT REFERENCES stock_pickings(id),
    account_move_id BIGINT REFERENCES account_moves(id),
    company_id BIGINT NOT NULL REFERENCES companies(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pos_orders (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    client_uuid VARCHAR(64) NOT NULL UNIQUE,
    session_id BIGINT NOT NULL REFERENCES pos_sessions(id),
    partner_id BIGINT REFERENCES partners(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    table_id BIGINT,
    customer_count INT NOT NULL DEFAULT 0,
    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    amount_untaxed NUMERIC(15,4) NOT NULL DEFAULT 0,
    amount_tax NUMERIC(15,4) NOT NULL DEFAULT 0,
    amount_total NUMERIC(15,4) NOT NULL DEFAULT 0,
    amount_paid NUMERIC(15,4) NOT NULL DEFAULT 0,
    amount_return NUMERIC(15,4) NOT NULL DEFAULT 0,
    tip_amount NUMERIC(15,4) NOT NULL DEFAULT 0,
    company_id BIGINT NOT NULL REFERENCES companies(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pos_orders_session ON pos_orders(session_id);
CREATE INDEX IF NOT EXISTS idx_pos_orders_partner ON pos_orders(partner_id);

CREATE TABLE IF NOT EXISTS pos_order_lines (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES pos_orders(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products(id),
    qty NUMERIC(15,4) NOT NULL,
    price_unit NUMERIC(15,4) NOT NULL DEFAULT 0,
    discount NUMERIC(5,2) NOT NULL DEFAULT 0,
    tax_rate NUMERIC(5,2) NOT NULL DEFAULT 0,
    price_subtotal NUMERIC(15,4) NOT NULL DEFAULT 0,
    price_subtotal_incl NUMERIC(15,4) NOT NULL DEFAULT 0,
    customer_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pos_payments (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES pos_orders(id) ON DELETE CASCADE,
    session_id BIGINT NOT NULL REFERENCES pos_sessions(id),
    payment_method_id BIGINT NOT NULL REFERENCES pos_payment_methods(id),
    amount NUMERIC(15,4) NOT NULL,
    payment_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    transaction_id VARCHAR(128)
);

CREATE TABLE IF NOT EXISTS pos_cash_movements (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES pos_sessions(id) ON DELETE CASCADE,
    type VARCHAR(8) NOT NULL CHECK (type IN ('in', 'out')),
    amount NUMERIC(15,4) NOT NULL CHECK (amount > 0),
    reason TEXT NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pos_sync_batches (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES pos_sessions(id),
    idempotency_key VARCHAR(128) NOT NULL UNIQUE,
    accepted_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);