-- 000054_stock_account_config.up.sql
-- Phase 1 Glue: Stock-Accounting Configuration (Odoo 19 stock_account properties)

CREATE TABLE IF NOT EXISTS stock_account_config (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE CASCADE,
    product_category_id BIGINT REFERENCES product_categories(id) ON DELETE CASCADE,
    stock_valuation_account_id BIGINT NOT NULL REFERENCES account_accounts(id) ON DELETE RESTRICT,
    stock_input_account_id BIGINT NOT NULL REFERENCES account_accounts(id) ON DELETE RESTRICT,
    stock_output_account_id BIGINT NOT NULL REFERENCES account_accounts(id) ON DELETE RESTRICT,
    stock_journal_id BIGINT NOT NULL REFERENCES account_journals(id) ON DELETE RESTRICT,
    price_diff_account_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add backorder_of_id to stock_pickings
ALTER TABLE stock_pickings ADD COLUMN IF NOT EXISTS backorder_of_id BIGINT REFERENCES stock_pickings(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_stock_pickings_backorder_of_id ON stock_pickings(backorder_of_id) WHERE backorder_of_id IS NOT NULL;

-- Unique per company and product_category (NULL category = company default)
CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_account_config_cat 
    ON stock_account_config (company_id, product_category_id) 
    WHERE product_category_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_account_config_company_default 
    ON stock_account_config (company_id) 
    WHERE product_category_id IS NULL;

-- Seed default company stock account config:
-- Valuation account: Inventory (id=5)
-- Input account: Stock Variation / Interim Received (id=16)
-- Output account: COGS (id=12)
-- Stock journal: STJ / Stock Operations (id=6)
-- Price diff account: Price Difference (id=17)
INSERT INTO stock_account_config (
    company_id, product_category_id, stock_valuation_account_id, stock_input_account_id,
    stock_output_account_id, stock_journal_id, price_diff_account_id
) VALUES (
    1, NULL, 5, 16, 12, 6, 17
) ON CONFLICT DO NOTHING;
