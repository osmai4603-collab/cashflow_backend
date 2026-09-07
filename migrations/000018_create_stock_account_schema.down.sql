-- 000018_create_stock_account_schema.down.sql

-- Backfill valuations (no-op; locations are left without valuation accounts in 000018 up)
UPDATE account_accounts SET account_stock_variation_id = NULL, account_stock_expense_id = NULL WHERE id = 5;

-- Drop seeded journal + accounts
DELETE FROM account_journals WHERE code = 'STJ';
DELETE FROM account_accounts WHERE id IN (16, 17);
SELECT setval('account_accounts_id_seq', (SELECT COALESCE(MAX(id), 1) FROM account_accounts));
SELECT setval('account_journals_id_seq', (SELECT COALESCE(MAX(id), 1) FROM account_journals));

-- Drop accounting periods
DROP TABLE IF EXISTS accounting_periods;

-- Drop product values history
DROP TABLE IF EXISTS product_values;

-- Drop COGS support fields
ALTER TABLE account_move_lines
    DROP COLUMN IF EXISTS cogs_origin_id,
    DROP COLUMN IF EXISTS display_type;

-- Drop account counterpart fields
ALTER TABLE account_accounts
    DROP COLUMN IF EXISTS account_stock_expense_id,
    DROP COLUMN IF EXISTS account_stock_variation_id;

-- Drop location valuation boundary
ALTER TABLE stock_locations
    DROP COLUMN IF EXISTS valuation_account_id;

-- Drop product valuation fields
ALTER TABLE product_templates
    DROP COLUMN IF EXISTS stock_journal_id,
    DROP COLUMN IF EXISTS price_difference_account_id,
    DROP COLUMN IF EXISTS stock_valuation_account_id,
    DROP COLUMN IF EXISTS total_value,
    DROP COLUMN IF EXISTS avg_cost,
    DROP COLUMN IF EXISTS lot_valuated,
    DROP COLUMN IF EXISTS valuation,
    DROP COLUMN IF EXISTS cost_method;

-- Drop category valuation defaults
ALTER TABLE product_categories
    DROP COLUMN IF EXISTS property_stock_journal_id,
    DROP COLUMN IF EXISTS property_price_difference_account_id,
    DROP COLUMN IF EXISTS property_stock_valuation_account_id,
    DROP COLUMN IF EXISTS property_lot_valuated,
    DROP COLUMN IF EXISTS property_valuation,
    DROP COLUMN IF EXISTS property_cost_method;

-- Drop stock_moves valuation fields
DROP INDEX IF EXISTS idx_stock_moves_valuation;
DROP INDEX IF EXISTS idx_stock_moves_account_move_id;
ALTER TABLE stock_moves
    DROP COLUMN IF EXISTS account_move_id,
    DROP COLUMN IF EXISTS remaining_value,
    DROP COLUMN IF EXISTS remaining_qty,
    DROP COLUMN IF EXISTS is_dropship,
    DROP COLUMN IF EXISTS is_out,
    DROP COLUMN IF EXISTS is_in,
    DROP COLUMN IF EXISTS standard_price,
    DROP COLUMN IF EXISTS value_manual,
    DROP COLUMN IF EXISTS value;
