-- 000006_create_purchase_schema.up.sql
-- Purchase Order schema: RFQ, Purchase Orders, Lines, and Bill Junction

-- 1. Sequence for Purchase Orders
CREATE SEQUENCE IF NOT EXISTS purchase_order_seq START 1;

-- 2. Purchase Orders (purchase.order in Odoo)
CREATE TABLE IF NOT EXISTS purchase_orders (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    partner_id BIGINT NOT NULL REFERENCES res_partners(id) ON DELETE RESTRICT,
    date_order TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    date_planned TIMESTAMPTZ,
    state VARCHAR(20) NOT NULL DEFAULT 'draft', -- 'draft', 'sent', 'purchase', 'done', 'cancel'
    invoice_status VARCHAR(20) NOT NULL DEFAULT 'no', -- 'no', 'to_invoice', 'invoiced'
    payment_term_id BIGINT REFERENCES account_payment_terms(id) ON DELETE SET NULL,
    user_id BIGINT,
    company_id BIGINT,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    note TEXT,
    amount_untaxed NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    amount_tax NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    amount_total NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_name ON purchase_orders(name);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_partner_id ON purchase_orders(partner_id);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_state ON purchase_orders(state);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_invoice_status ON purchase_orders(invoice_status);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_date_order ON purchase_orders(date_order);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_active ON purchase_orders(active);

CREATE TRIGGER trg_purchase_orders_updated_at
    BEFORE UPDATE ON purchase_orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 3. Purchase Order Lines (purchase.order.line in Odoo)
CREATE TABLE IF NOT EXISTS purchase_order_lines (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    sequence INT NOT NULL DEFAULT 10,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    product_qty NUMERIC(15, 4) NOT NULL DEFAULT 1.0000,
    product_uom BIGINT REFERENCES uom_uoms(id) ON DELETE SET NULL,
    price_unit NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    discount NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    tax_ids BIGINT[] DEFAULT '{}',
    price_subtotal NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    price_tax NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    price_total NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    qty_received NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    qty_invoiced NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_order_lines_order_id ON purchase_order_lines(order_id);
CREATE INDEX IF NOT EXISTS idx_purchase_order_lines_product_id ON purchase_order_lines(product_id);

CREATE TRIGGER trg_purchase_order_lines_updated_at
    BEFORE UPDATE ON purchase_order_lines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 4. Purchase Order Invoices (Vendor Bills) Junction
CREATE TABLE IF NOT EXISTS purchase_order_invoices (
    order_id BIGINT NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    move_id BIGINT NOT NULL REFERENCES account_moves(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (order_id, move_id)
);

CREATE INDEX IF NOT EXISTS idx_purchase_order_invoices_move_id ON purchase_order_invoices(move_id);
