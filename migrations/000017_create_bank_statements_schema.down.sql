-- 000017_create_bank_statements_schema.down.sql

-- Drop reconciliation state from move lines
DROP INDEX IF EXISTS idx_account_move_lines_reconcile_state;
DROP INDEX IF EXISTS idx_account_move_lines_statement_line_id;
DROP INDEX IF EXISTS idx_account_move_lines_matching_number;

ALTER TABLE account_move_lines
    DROP COLUMN IF EXISTS statement_line_id,
    DROP COLUMN IF EXISTS matching_number,
    DROP COLUMN IF EXISTS amount_residual,
    DROP COLUMN IF EXISTS reconciled,
    DROP COLUMN IF EXISTS reconcile;

-- Drop cash rounding
DROP TABLE IF EXISTS account_cash_roundings;

-- Drop reconcile models
DROP TABLE IF EXISTS account_reconcile_model_lines;
DROP TABLE IF EXISTS account_reconcile_models;

-- Drop reconciles
DROP TABLE IF EXISTS account_full_reconciles;
DROP TABLE IF EXISTS account_partial_reconciles;

-- Drop bank statement lines / statements
DROP TABLE IF EXISTS account_bank_statement_lines;
DROP TABLE IF EXISTS account_bank_statements;