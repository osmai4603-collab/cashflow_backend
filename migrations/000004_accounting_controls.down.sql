DROP TABLE IF EXISTS account_reconciliation_items;
DROP TABLE IF EXISTS account_reconciliations;
ALTER TABLE account_move_lines DROP CONSTRAINT IF EXISTS chk_line_reconciled_amount;
ALTER TABLE account_move_lines DROP COLUMN IF EXISTS reconciled_amount;
DROP TABLE IF EXISTS currency_exchange_rates;
DROP TABLE IF EXISTS subledger_event_mappings;
DROP INDEX IF EXISTS uq_accounting_configurations_singleton;
DROP TABLE IF EXISTS accounting_configurations;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS chk_accounts_cash_flow_activity;
ALTER TABLE accounts
    DROP COLUMN IF EXISTS cash_flow_activity,
    DROP COLUMN IF EXISTS is_non_cash_adjustment,
    DROP COLUMN IF EXISTS is_cash_equivalent;
