-- 000017_create_bank_statements_schema.up.sql
-- Bank Statements & Reconciliation: statements, lines, partial/full reconciles,
-- reconcile models (Odoo 19 design), cash rounding, and reconciliation fields on move lines.

-- 1. Bank Statements (account.bank.statement)
CREATE TABLE IF NOT EXISTS account_bank_statements (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL DEFAULT '/',
    journal_id BIGINT NOT NULL REFERENCES account_journals(id) ON DELETE RESTRICT,
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    balance_start NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    balance_end NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    balance_end_real NUMERIC(15, 4),
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    state VARCHAR(20) NOT NULL DEFAULT 'open', -- 'open', 'confirm'
    is_complete BOOLEAN NOT NULL DEFAULT false,
    is_valid BOOLEAN NOT NULL DEFAULT false,
    problem_description TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_account_bank_statements_journal_id ON account_bank_statements(journal_id);
CREATE INDEX IF NOT EXISTS idx_account_bank_statements_date ON account_bank_statements(date);
CREATE INDEX IF NOT EXISTS idx_account_bank_statements_state ON account_bank_statements(state);
CREATE INDEX IF NOT EXISTS idx_account_bank_statements_active ON account_bank_statements(active);

CREATE TRIGGER trg_account_bank_statements_updated_at
    BEFORE UPDATE ON account_bank_statements
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 2. Bank Statement Lines (account.bank.statement.line)
CREATE TABLE IF NOT EXISTS account_bank_statement_lines (
    id BIGSERIAL PRIMARY KEY,
    statement_id BIGINT NOT NULL REFERENCES account_bank_statements(id) ON DELETE CASCADE,
    name VARCHAR(500) NOT NULL DEFAULT '/',
    ref VARCHAR(255),
    sequence INT NOT NULL DEFAULT 1,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    amount NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    amount_currency NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    account_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL,
    move_id BIGINT REFERENCES account_moves(id) ON DELETE SET NULL,
    journal_id BIGINT REFERENCES account_journals(id) ON DELETE RESTRICT,
    checked BOOLEAN NOT NULL DEFAULT false,
    running_balance NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    amount_residual NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    reconciled BOOLEAN NOT NULL DEFAULT false,
    matching_number VARCHAR(64),
    internal_index VARCHAR(100),
    import_batch_id VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bsl_statement_id ON account_bank_statement_lines(statement_id);
CREATE INDEX IF NOT EXISTS idx_bsl_move_id ON account_bank_statement_lines(move_id);
CREATE INDEX IF NOT EXISTS idx_bsl_partner_id ON account_bank_statement_lines(partner_id);
CREATE INDEX IF NOT EXISTS idx_bsl_account_id ON account_bank_statement_lines(account_id);
CREATE INDEX IF NOT EXISTS idx_bsl_reconciled ON account_bank_statement_lines(reconciled);
CREATE INDEX IF NOT EXISTS idx_bsl_date ON account_bank_statement_lines(date);
CREATE INDEX IF NOT EXISTS idx_bsl_sequence ON account_bank_statement_lines(statement_id, sequence);

CREATE TRIGGER trg_account_bank_statement_lines_updated_at
    BEFORE UPDATE ON account_bank_statement_lines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 3. Partial Reconciles (account.partial.reconcile)
CREATE TABLE IF NOT EXISTS account_partial_reconciles (
    id BIGSERIAL PRIMARY KEY,
    debit_move_id BIGINT REFERENCES account_moves(id) ON DELETE CASCADE,
    credit_move_id BIGINT REFERENCES account_moves(id) ON DELETE CASCADE,
    debit_line_id BIGINT NOT NULL REFERENCES account_move_lines(id) ON DELETE CASCADE,
    credit_line_id BIGINT NOT NULL REFERENCES account_move_lines(id) ON DELETE CASCADE,
    amount NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    amount_currency NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pr_debit_line ON account_partial_reconciles(debit_line_id);
CREATE INDEX IF NOT EXISTS idx_pr_credit_line ON account_partial_reconciles(credit_line_id);
CREATE INDEX IF NOT EXISTS idx_pr_debit_move ON account_partial_reconciles(debit_move_id);
CREATE INDEX IF NOT EXISTS idx_pr_credit_move ON account_partial_reconciles(credit_move_id);

CREATE TRIGGER trg_account_partial_reconciles_updated_at
    BEFORE UPDATE ON account_partial_reconciles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 4. Full Reconciles (account.full.reconcile)
CREATE TABLE IF NOT EXISTS account_full_reconciles (
    id BIGSERIAL PRIMARY KEY,
    matching_number VARCHAR(64) NOT NULL UNIQUE,
    exchange_move_id BIGINT REFERENCES account_moves(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_account_full_reconciles_updated_at
    BEFORE UPDATE ON account_full_reconciles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 5. Reconcile Models (account.reconcile.model - Odoo 19 design, G3)
CREATE TABLE IF NOT EXISTS account_reconcile_models (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    sequence INT NOT NULL DEFAULT 10,
    is_auto_reconcile BOOLEAN NOT NULL DEFAULT false,
    match_nature VARCHAR(20) NOT NULL DEFAULT 'both', -- 'both', 'money_in', 'money_out'
    match_amount VARCHAR(20), -- 'lower', 'greater', 'between'
    match_amount_min NUMERIC(15, 4),
    match_amount_max NUMERIC(15, 4),
    match_label VARCHAR(20), -- 'contains', 'not_contains', 'match_regex'
    match_label_param VARCHAR(255),
    match_journal_ids BIGINT[] DEFAULT '{}',
    match_partner_ids BIGINT[] DEFAULT '{}',
    mapped_partner_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_reconcile_models_active ON account_reconcile_models(active);
CREATE INDEX IF NOT EXISTS idx_account_reconcile_models_auto ON account_reconcile_models(is_auto_reconcile);

CREATE TRIGGER trg_account_reconcile_models_updated_at
    BEFORE UPDATE ON account_reconcile_models
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS account_reconcile_model_lines (
    id BIGSERIAL PRIMARY KEY,
    reconcile_model_id BIGINT NOT NULL REFERENCES account_reconcile_models(id) ON DELETE CASCADE,
    amount_type VARCHAR(20) NOT NULL DEFAULT 'fixed', -- 'fixed', 'percentage', 'percentage_st_line', 'regex'
    amount VARCHAR(255) NOT NULL DEFAULT '0',
    account_id BIGINT REFERENCES account_accounts(id) ON DELETE RESTRICT,
    label VARCHAR(255),
    tax_ids BIGINT[] DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_rml_model_id ON account_reconcile_model_lines(reconcile_model_id);
CREATE INDEX IF NOT EXISTS idx_rml_account_id ON account_reconcile_model_lines(account_id);

-- 6. Cash Rounding (account.cash.rounding - CRUD scope, G6)
CREATE TABLE IF NOT EXISTS account_cash_roundings (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    rounding_method VARCHAR(20) NOT NULL DEFAULT 'HALF-UP', -- 'UP', 'DOWN', 'HALF-UP', 'HALF-DOWN'
    rounding NUMERIC(15, 4) NOT NULL DEFAULT 0.01,
    strategy VARCHAR(20) NOT NULL DEFAULT 'add_invoice_line', -- 'add_invoice_line', 'biggest_tax'
    profit_account_id BIGINT REFERENCES account_accounts(id) ON DELETE RESTRICT,
    loss_account_id BIGINT REFERENCES account_accounts(id) ON DELETE RESTRICT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_cash_roundings_active ON account_cash_roundings(active);

CREATE TRIGGER trg_account_cash_roundings_updated_at
    BEFORE UPDATE ON account_cash_roundings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 7. Extend account_move_lines with reconciliation state (G4)
ALTER TABLE account_move_lines
    ADD COLUMN IF NOT EXISTS reconcile BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS reconciled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS amount_residual NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    ADD COLUMN IF NOT EXISTS matching_number VARCHAR(64),
    ADD COLUMN IF NOT EXISTS statement_line_id BIGINT REFERENCES account_bank_statement_lines(id) ON DELETE SET NULL;

-- Backfill reconciliation defaults from the chart of accounts for historical lines
UPDATE account_move_lines l
SET reconcile = a.reconcile,
    reconciled = (a.reconcile = false),
    amount_residual = CASE WHEN a.reconcile THEN ABS(l.balance) ELSE 0 END
FROM account_accounts a
WHERE a.id = l.account_id;

CREATE INDEX IF NOT EXISTS idx_account_move_lines_reconcile_state
    ON account_move_lines (reconcile, reconciled, partner_id, amount_residual);
CREATE INDEX IF NOT EXISTS idx_account_move_lines_statement_line_id
    ON account_move_lines (statement_line_id);
CREATE INDEX IF NOT EXISTS idx_account_move_lines_matching_number
    ON account_move_lines (matching_number);