-- 000018_create_stock_account_schema.up.sql
-- Phase 12: Stock-Account Integration (Odoo 19.0 stock_account reference)
-- - Valuation fields merged into stock_moves (value, standard_price, is_in/out, account_move_id)
-- - cost_method / valuation / lot_valuated on products, categories and companies
-- - valuation_account_id on stock locations
-- - Stock Variation / Price Difference accounts + Stock Journal (STJ)
-- - product_values history table (equivalent to product.value)
-- - accounting_periods table for periodic (closing) valuation

-- 1. Extend stock_moves with valuation fields (integrated into the move, per Odoo 19)
ALTER TABLE stock_moves
    ADD COLUMN IF NOT EXISTS value NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    ADD COLUMN IF NOT EXISTS value_manual NUMERIC(15, 4),
    ADD COLUMN IF NOT EXISTS standard_price NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    ADD COLUMN IF NOT EXISTS is_in BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_out BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_dropship BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS remaining_qty NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    ADD COLUMN IF NOT EXISTS remaining_value NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    ADD COLUMN IF NOT EXISTS account_move_id BIGINT REFERENCES account_moves(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_stock_moves_account_move_id ON stock_moves(account_move_id);
CREATE INDEX IF NOT EXISTS idx_stock_moves_valuation ON stock_moves(value) WHERE is_in OR is_out;

-- 2. Extend product_categories with valuation defaults (company_templates / company_dependent)
ALTER TABLE product_categories
    ADD COLUMN IF NOT EXISTS property_cost_method VARCHAR(10),
    ADD COLUMN IF NOT EXISTS property_valuation VARCHAR(10),
    ADD COLUMN IF NOT EXISTS property_lot_valuated BOOLEAN,
    ADD COLUMN IF NOT EXISTS property_stock_valuation_account_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS property_price_difference_account_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS property_stock_journal_id BIGINT REFERENCES account_journals(id) ON DELETE SET NULL;

-- 3. Extend product_templates with valuation fields (resolved from category/company at runtime)
ALTER TABLE product_templates
    ADD COLUMN IF NOT EXISTS cost_method VARCHAR(10) NOT NULL DEFAULT 'standard', -- 'standard', 'fifo', 'average'
    ADD COLUMN IF NOT EXISTS valuation VARCHAR(10) NOT NULL DEFAULT 'real_time',  -- 'real_time', 'periodic'
    ADD COLUMN IF NOT EXISTS lot_valuated BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS avg_cost NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    ADD COLUMN IF NOT EXISTS total_value NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    ADD COLUMN IF NOT EXISTS stock_valuation_account_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS price_difference_account_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS stock_journal_id BIGINT REFERENCES account_journals(id) ON DELETE SET NULL;

-- 4. Extend stock_locations with valuation boundary account
ALTER TABLE stock_locations
    ADD COLUMN IF NOT EXISTS valuation_account_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL;

-- 5. Extend account_accounts with stock-variation / stock-expense counterpart fields
ALTER TABLE account_accounts
    ADD COLUMN IF NOT EXISTS account_stock_variation_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS account_stock_expense_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL;

-- 6. Extend account_move_lines with COGS and landed-cost support fields (Anglo-Saxon)
ALTER TABLE account_move_lines
    ADD COLUMN IF NOT EXISTS display_type VARCHAR(20),
    ADD COLUMN IF NOT EXISTS cogs_origin_id BIGINT REFERENCES account_move_lines(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS is_landed_costs_line BOOLEAN NOT NULL DEFAULT false;

-- 7. Product Values history table (equivalent of product.value / old stock.valuation.layer history)
CREATE TABLE IF NOT EXISTS product_values (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT REFERENCES product_templates(id) ON DELETE CASCADE,
    lot_id BIGINT,
    move_id BIGINT REFERENCES stock_moves(id) ON DELETE SET NULL,
    value NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    company_id BIGINT,
    date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id BIGINT,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_product_values_product_id ON product_values(product_id);
CREATE INDEX IF NOT EXISTS idx_product_values_move_id ON product_values(move_id);
CREATE INDEX IF NOT EXISTS idx_product_values_date ON product_values(date);

CREATE TRIGGER trg_product_values_updated_at
    BEFORE UPDATE ON product_values
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 8. Accounting Periods for periodic (closing) valuation
CREATE TABLE IF NOT EXISTS accounting_periods (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    date_from DATE NOT NULL,
    date_to DATE NOT NULL,
    state VARCHAR(16) NOT NULL DEFAULT 'open', -- 'open', 'closed'
    journal_id BIGINT REFERENCES account_journals(id) ON DELETE RESTRICT,
    account_move_id BIGINT REFERENCES account_moves(id) ON DELETE SET NULL,
    company_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_accounting_periods_dates ON accounting_periods(date_from, date_to);
CREATE INDEX IF NOT EXISTS idx_accounting_periods_state ON accounting_periods(state);

CREATE TRIGGER trg_accounting_periods_updated_at
    BEFORE UPDATE ON accounting_periods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 9. Seed: Stock Variation account (counterpart of Inventory at closing)
INSERT INTO account_accounts (id, code, name, type, reconcile, currency, active) VALUES
(16, '140100', 'Stock Variation', 'asset_current', false, 'USD', true),
(17, '510000', 'Price Difference', 'expense_direct_cost', false, 'USD', true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('account_accounts_id_seq', (SELECT COALESCE(MAX(id), 1) FROM account_accounts));

-- 10. Seed: Stock Journal (STJ, general type) for valuation entries
INSERT INTO account_journals (id, name, code, type, default_account_id, suspense_account_id, sequence_prefix, next_number, active) VALUES
(6, 'Stock Operations', 'STJ', 'general', 5, NULL, 'STJ/%Y/', 1, true)
ON CONFLICT (code) DO NOTHING;

SELECT setval('account_journals_id_seq', (SELECT COALESCE(MAX(id), 1) FROM account_journals));

-- 11. Wire default stock valuation account on the Inventory account pair
-- Inventory account (id=5) points to Stock Variation (id=16) for periodic closing
UPDATE account_accounts
SET account_stock_variation_id = 16,
    account_stock_expense_id = 12 -- COGS
WHERE id = 5;

-- 12. Backfill valuation defaults on seeded stock locations (WH/Stock internal boundary)
-- Odoo 19 does NOT assign valuation_account_id to plain internal stock locations by default;
-- only valued boundaries (transit / production / cost locations) carry one. Leave them NULL so
-- standard receipts/deliveries are valued at invoice-time (Anglo-Saxon) rather than at validate.
