-- 000009_create_payments_schema.up.sql
-- Payments & Reconciliations schema: Payments, sequence, and reconciliation link table

-- 1. Sequence for Payments (PAY/YYYY/NNNNN)
CREATE SEQUENCE IF NOT EXISTS account_payment_seq START WITH 1 INCREMENT BY 1;

-- 2. Payments (account.payment in Odoo)
CREATE TABLE IF NOT EXISTS account_payments (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL DEFAULT '/',
    payment_type VARCHAR(20) NOT NULL, -- 'inbound', 'outbound'
    partner_type VARCHAR(20) NOT NULL DEFAULT 'customer', -- 'customer', 'supplier'
    partner_id BIGINT NOT NULL REFERENCES res_partners(id) ON DELETE RESTRICT,
    amount NUMERIC(15, 4) NOT NULL CHECK (amount > 0),
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    payment_method VARCHAR(50) NOT NULL DEFAULT 'cash', -- 'cash', 'bank_transfer', 'check'
    journal_id BIGINT NOT NULL REFERENCES account_journals(id) ON DELETE RESTRICT,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    state VARCHAR(20) NOT NULL DEFAULT 'draft', -- 'draft', 'posted', 'reconciled', 'cancelled'
    ref VARCHAR(255),
    move_id BIGINT REFERENCES account_moves(id) ON DELETE SET NULL,
    reconciled_amount NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    residual_amount NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    company_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_account_payments_name ON account_payments(name);
CREATE INDEX IF NOT EXISTS idx_account_payments_partner_id ON account_payments(partner_id);
CREATE INDEX IF NOT EXISTS idx_account_payments_type ON account_payments(payment_type);
CREATE INDEX IF NOT EXISTS idx_account_payments_state ON account_payments(state);
CREATE INDEX IF NOT EXISTS idx_account_payments_journal_id ON account_payments(journal_id);
CREATE INDEX IF NOT EXISTS idx_account_payments_date ON account_payments(date);
CREATE INDEX IF NOT EXISTS idx_account_payments_active ON account_payments(active);

CREATE TRIGGER trg_account_payments_updated_at
    BEFORE UPDATE ON account_payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 3. Payment Invoices Reconciliation (account.payment matching with account.move)
CREATE TABLE IF NOT EXISTS account_payment_reconciliations (
    id BIGSERIAL PRIMARY KEY,
    payment_id BIGINT NOT NULL REFERENCES account_payments(id) ON DELETE CASCADE,
    invoice_id BIGINT NOT NULL REFERENCES account_moves(id) ON DELETE RESTRICT,
    amount NUMERIC(15, 4) NOT NULL CHECK (amount > 0),
    reconciled_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_payment_reconciliations_payment ON account_payment_reconciliations(payment_id);
CREATE INDEX IF NOT EXISTS idx_account_payment_reconciliations_invoice ON account_payment_reconciliations(invoice_id);
