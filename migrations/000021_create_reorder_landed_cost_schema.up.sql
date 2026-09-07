-- 000021_create_reorder_landed_cost_schema.up.sql
-- Phase 13: Auto Reorder (stock.orderpoint) + Landed Costs (stock.landed.cost)
-- Reference: Odoo 19.0 addons/stock/models/stock_orderpoint.py,
--            addons/stock_landed_costs/models/stock_landed_cost.py, purchase.py, account_move.py

-- 1. Reorder rules (stock.orderpoint)
CREATE TABLE IF NOT EXISTS stock_orderpoints (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,               -- "ROP/2026/00001"
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE CASCADE,
    warehouse_id BIGINT REFERENCES stock_warehouses(id) ON DELETE CASCADE,
    location_id BIGINT NOT NULL REFERENCES stock_locations(id) ON DELETE CASCADE,
    vendor_id BIGINT REFERENCES partners(id) ON DELETE SET NULL,   -- preferred supplier (Odoo res.partner)
    min_qty NUMERIC(15, 4) NOT NULL DEFAULT 0.0,                  -- qty_forecast lower bound
    max_qty NUMERIC(15, 4) NOT NULL DEFAULT 0.0,                  -- qty_forecast upper bound
    qty_multiple NUMERIC(15, 4) NOT NULL DEFAULT 1.0,             -- qty ordering multiple
    lead_days INTEGER NOT NULL DEFAULT 0,
    source VARCHAR(16) NOT NULL DEFAULT 'buy',                    -- 'buy' (procurement route)
    trigger VARCHAR(16) NOT NULL DEFAULT 'manual',                -- 'auto' | 'manual'
    snoozed_until TIMESTAMPTZ,
    qty_on_hand NUMERIC(15, 4) NOT NULL DEFAULT 0.0,              -- computed
    qty_forecast NUMERIC(15, 4) NOT NULL DEFAULT 0.0,             -- computed (on_hand - out + in over lead horizon)
    qty_to_order NUMERIC(15, 4) NOT NULL DEFAULT 0.0,             -- computed (max(max_forecast - forecast, 0), rounded to multiple)
    qty_to_order_manual NUMERIC(15, 4) NOT NULL DEFAULT 0.0,      -- manual override (Prefer to order)
    deadline_date DATE,
    active BOOLEAN NOT NULL DEFAULT true,
    company_id BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Odoo stock_orderpoint._name: unique(product_id, location_id, company_id)
CREATE UNIQUE INDEX IF NOT EXISTS idx_stock_orderpoints_product_location_company
    ON stock_orderpoints(product_id, location_id, company_id) WHERE active = true;

CREATE INDEX IF NOT EXISTS idx_stock_orderpoints_warehouse ON stock_orderpoints(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_stock_orderpoints_trigger ON stock_orderpoints(trigger) WHERE active = true;

CREATE TRIGGER trg_stock_orderpoints_updated_at
    BEFORE UPDATE ON stock_orderpoints
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 2. Landed costs header (stock.landed.cost)
CREATE TABLE IF NOT EXISTS stock_landed_costs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,               -- "LC/2026/00001"
    date DATE NOT NULL,
    state VARCHAR(16) NOT NULL DEFAULT 'draft', -- 'draft' | 'done' | 'cancel'
    picking_ids BIGINT[] NOT NULL DEFAULT '{}', -- M2M stock.picking
    amount_total NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    description TEXT,
    account_move_id BIGINT REFERENCES account_moves(id) ON DELETE SET NULL, -- the created valuation entry
    journal_id BIGINT REFERENCES account_journals(id) ON DELETE RESTRICT,   -- STJ / expense journal
    vendor_bill_id BIGINT REFERENCES account_moves(id) ON DELETE SET NULL,  -- source vendor bill (stock_landed_costs.vendor_bill_id)
    company_id BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_stock_landed_costs_state ON stock_landed_costs(state);
CREATE INDEX IF NOT EXISTS idx_stock_landed_costs_pickings ON stock_landed_costs USING GIN (picking_ids);

CREATE TRIGGER trg_stock_landed_costs_updated_at
    BEFORE UPDATE ON stock_landed_costs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 3. Landed cost lines (stock.landed.cost.lines) — expense / cost lines
CREATE TABLE IF NOT EXISTS stock_landed_cost_lines (
    id BIGSERIAL PRIMARY KEY,
    landed_cost_id BIGINT NOT NULL REFERENCES stock_landed_costs(id) ON DELETE CASCADE,
    name VARCHAR(512) NOT NULL,
    product_id BIGINT REFERENCES product_templates(id) ON DELETE SET NULL, -- cost product (non-inventory)
    account_id BIGINT REFERENCES account_accounts(id) ON DELETE RESTRICT,  -- expense account (account_expense_line)
    price_unit NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    split_method VARCHAR(24) NOT NULL DEFAULT 'equal', -- 'equal' | 'by_quantity' | 'by_current_cost_price' | 'by_weight' | 'by_volume'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_landed_cost_lines_lc ON stock_landed_cost_lines(landed_cost_id);

CREATE TRIGGER trg_stock_landed_cost_lines_updated_at
    BEFORE UPDATE ON stock_landed_cost_lines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 4. Valuation adjustment lines (stock.valuation.adjustment.lines) — computed allocation per move
CREATE TABLE IF NOT EXISTS stock_valuation_adjustment_lines (
    id BIGSERIAL PRIMARY KEY,
    landed_cost_id BIGINT NOT NULL REFERENCES stock_landed_costs(id) ON DELETE CASCADE,
    cost_line_id BIGINT REFERENCES stock_landed_cost_lines(id) ON DELETE CASCADE,
    move_id BIGINT REFERENCES stock_moves(id) ON DELETE CASCADE,
    product_id BIGINT REFERENCES product_templates(id) ON DELETE CASCADE,
    quantity NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    weight NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    volume NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    former_cost NUMERIC(15, 4) NOT NULL DEFAULT 0.0,    -- additional_landed_cost
    additional_cost NUMERIC(15, 4) NOT NULL DEFAULT 0.0, -- per unit allocation for this move
    final_cost NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    move_remaining_qty NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_valuation_adj_landed_cost ON stock_valuation_adjustment_lines(landed_cost_id);
CREATE INDEX IF NOT EXISTS idx_stock_valuation_adj_move ON stock_valuation_adjustment_lines(move_id);

CREATE TRIGGER trg_stock_valuation_adjustment_lines_updated_at
    BEFORE UPDATE ON stock_valuation_adjustment_lines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 5. Product flag for landed-cost capability (stock_landed_costs.purchase.py: landed_cost_ok)
ALTER TABLE product_templates
    ADD COLUMN IF NOT EXISTS landed_cost_ok BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS split_method_landed_cost VARCHAR(24) NOT NULL DEFAULT 'equal';

-- 6. Company default journal for landed-cost entries (stock_landed_costs res.company: lc_journal_id)
ALTER TABLE companies
    ADD COLUMN IF NOT EXISTS lc_journal_id BIGINT REFERENCES account_journals(id) ON DELETE SET NULL;

-- 7. Orderpoint sequence
CREATE SEQUENCE IF NOT EXISTS stock_orderpoint_sequence START 1;

-- 8. Landed cost sequence
CREATE SEQUENCE IF NOT EXISTS stock_landed_cost_sequence START 1;

-- 9. ACL: reorder rules & landed costs (mirror stock.* permissions in 000016)
INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'stock.orderpoint', true, false, false, false), (2::BIGINT, 'stock.orderpoint', true, true, true, true),
    (1::BIGINT, 'stock.landed.cost', true, false, false, false), (2::BIGINT, 'stock.landed.cost', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;