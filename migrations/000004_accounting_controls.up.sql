ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS is_cash_equivalent BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS is_non_cash_adjustment BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS cash_flow_activity VARCHAR(20) NOT NULL DEFAULT 'unclassified';

ALTER TABLE accounts ADD CONSTRAINT chk_accounts_cash_flow_activity
    CHECK (cash_flow_activity IN ('operating', 'investing', 'financing', 'none', 'unclassified'));

CREATE TABLE IF NOT EXISTS accounting_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    retained_earnings_account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    fx_gain_account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    fx_loss_account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    fx_reconciliation_journal_id UUID NOT NULL REFERENCES journals(id) ON DELETE RESTRICT,
    functional_currency VARCHAR(3) NOT NULL DEFAULT 'SAR',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_accounting_configurations_singleton
    ON accounting_configurations ((TRUE));

CREATE TABLE IF NOT EXISTS subledger_event_mappings (
    event_type VARCHAR(100) PRIMARY KEY,
    journal_id UUID NOT NULL REFERENCES journals(id) ON DELETE RESTRICT,
    debit_account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    credit_account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    description VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS currency_exchange_rates (
    currency VARCHAR(3) NOT NULL,
    base_currency VARCHAR(3) NOT NULL,
    rate_date DATE NOT NULL,
    rate NUMERIC(28, 10) NOT NULL CHECK (rate > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (currency, base_currency, rate_date),
    CONSTRAINT chk_currency_exchange_rate_pair CHECK (currency <> base_currency)
);

CREATE INDEX IF NOT EXISTS idx_currency_exchange_rates_lookup
    ON currency_exchange_rates (currency, base_currency, rate_date DESC);

INSERT INTO currency_exchange_rates (currency, base_currency, rate_date, rate)
VALUES ('USD', 'SAR', DATE '1986-06-01', 3.750000)
ON CONFLICT (currency, base_currency, rate_date) DO NOTHING;

CREATE TABLE IF NOT EXISTS account_reconciliations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    currency_id VARCHAR(3) NOT NULL,
    realized_fx_move_id UUID REFERENCES account_moves(id) ON DELETE RESTRICT,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS account_reconciliation_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reconciliation_id UUID NOT NULL REFERENCES account_reconciliations(id) ON DELETE CASCADE,
    move_line_id UUID NOT NULL REFERENCES account_move_lines(id) ON DELETE RESTRICT,
    allocated_amount NUMERIC(28, 4) NOT NULL CHECK (allocated_amount > 0),
    currency_id VARCHAR(3) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_reconciliation_line UNIQUE (reconciliation_id, move_line_id)
);

ALTER TABLE account_move_lines
    ADD COLUMN IF NOT EXISTS reconciled_amount NUMERIC(28, 4) NOT NULL DEFAULT 0;

ALTER TABLE account_move_lines
    ADD CONSTRAINT chk_line_reconciled_amount CHECK (reconciled_amount >= 0);

CREATE INDEX IF NOT EXISTS idx_reconciliation_items_line ON account_reconciliation_items (move_line_id);
CREATE INDEX IF NOT EXISTS idx_reconciliation_date ON account_reconciliations (date);
