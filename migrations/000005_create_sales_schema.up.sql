-- 000005_create_sales_schema.up.sql
-- Sales Order schema: Quotations, Sale Orders, Lines, and Invoice Junction

-- 1. Sequence for Sales Orders
CREATE SEQUENCE IF NOT EXISTS sale_order_seq START 1;

-- 2. Sale Orders (sale.order in Odoo)
CREATE TABLE IF NOT EXISTS sale_orders (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    partner_id BIGINT NOT NULL REFERENCES res_partners(id) ON DELETE RESTRICT,
    date_order TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    validity_date TIMESTAMPTZ,
    state VARCHAR(20) NOT NULL DEFAULT 'draft', -- 'draft', 'sent', 'sale', 'done', 'cancel'
    invoice_status VARCHAR(20) NOT NULL DEFAULT 'no', -- 'no', 'to_invoice', 'invoiced'
    pricelist_id BIGINT REFERENCES product_pricelists(id) ON DELETE SET NULL,
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

CREATE INDEX IF NOT EXISTS idx_sale_orders_name ON sale_orders(name);
CREATE INDEX IF NOT EXISTS idx_sale_orders_partner_id ON sale_orders(partner_id);
CREATE INDEX IF NOT EXISTS idx_sale_orders_state ON sale_orders(state);
CREATE INDEX IF NOT EXISTS idx_sale_orders_invoice_status ON sale_orders(invoice_status);
CREATE INDEX IF NOT EXISTS idx_sale_orders_date_order ON sale_orders(date_order);
CREATE INDEX IF NOT EXISTS idx_sale_orders_active ON sale_orders(active);

CREATE TRIGGER trg_sale_orders_updated_at
    BEFORE UPDATE ON sale_orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 3. Sale Order Lines (sale.order.line in Odoo)
CREATE TABLE IF NOT EXISTS sale_order_lines (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES sale_orders(id) ON DELETE CASCADE,
    sequence INT NOT NULL DEFAULT 10,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    product_uom_qty NUMERIC(15, 4) NOT NULL DEFAULT 1.0000,
    product_uom BIGINT REFERENCES uom_uoms(id) ON DELETE SET NULL,
    price_unit NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    discount NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    tax_ids BIGINT[] DEFAULT '{}',
    price_subtotal NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    price_tax NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    price_total NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    qty_delivered NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    qty_invoiced NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sale_order_lines_order_id ON sale_order_lines(order_id);
CREATE INDEX IF NOT EXISTS idx_sale_order_lines_product_id ON sale_order_lines(product_id);

CREATE TRIGGER trg_sale_order_lines_updated_at
    BEFORE UPDATE ON sale_order_lines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 4. Sale Order Invoices Junction
CREATE TABLE IF NOT EXISTS sale_order_invoices (
    order_id BIGINT NOT NULL REFERENCES sale_orders(id) ON DELETE CASCADE,
    move_id BIGINT NOT NULL REFERENCES account_moves(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (order_id, move_id)
);

CREATE INDEX IF NOT EXISTS idx_sale_order_invoices_move_id ON sale_order_invoices(move_id);
