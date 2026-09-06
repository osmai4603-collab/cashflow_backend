-- 000004_create_accounting_schema.up.sql
-- Core Accounting schema: Chart of accounts, journals, taxes, payment terms, account moves, and move lines

-- 1. Chart of Accounts (account.account)
CREATE TABLE IF NOT EXISTS account_accounts (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'asset_receivable', 'asset_cash', 'asset_current', 'asset_non_current', 'liability_payable', 'liability_current', 'liability_non_current', 'equity', 'income', 'income_other', 'expense', 'expense_depreciation', 'expense_direct_cost'
    reconcile BOOLEAN NOT NULL DEFAULT false,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    parent_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL,
    company_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_account_accounts_code ON account_accounts(code);
CREATE INDEX IF NOT EXISTS idx_account_accounts_type ON account_accounts(type);
CREATE INDEX IF NOT EXISTS idx_account_accounts_active ON account_accounts(active);
CREATE INDEX IF NOT EXISTS idx_account_accounts_parent_id ON account_accounts(parent_id);

CREATE TRIGGER trg_account_accounts_updated_at
    BEFORE UPDATE ON account_accounts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed Standard Chart of Accounts
INSERT INTO account_accounts (id, code, name, type, reconcile, currency, active) VALUES
(1, '101000', 'Cash on Hand', 'asset_cash', false, 'USD', true),
(2, '102000', 'Bank Account', 'asset_cash', false, 'USD', true),
(3, '120000', 'Accounts Receivable', 'asset_receivable', true, 'USD', true),
(4, '130000', 'VAT Input (Tax Receivable)', 'asset_current', false, 'USD', true),
(5, '140000', 'Inventory', 'asset_current', false, 'USD', true),
(6, '210000', 'Accounts Payable', 'liability_payable', true, 'USD', true),
(7, '220000', 'VAT Output (Tax Payable)', 'liability_current', false, 'USD', true),
(8, '300000', 'Capital / Equity', 'equity', false, 'USD', true),
(9, '320000', 'Retained Earnings', 'equity', false, 'USD', true),
(10, '400000', 'Product Sales Revenue', 'income', false, 'USD', true),
(11, '410000', 'Service Revenue', 'income', false, 'USD', true),
(12, '500000', 'Cost of Goods Sold', 'expense_direct_cost', false, 'USD', true),
(13, '600000', 'Operating Expenses', 'expense', false, 'USD', true),
(14, '610000', 'Salaries and Wages', 'expense', false, 'USD', true),
(15, '999999', 'Undistributed Profits/Losses', 'equity', false, 'USD', true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('account_accounts_id_seq', (SELECT COALESCE(MAX(id), 1) FROM account_accounts));

-- 2. Journals (account.journal)
CREATE TABLE IF NOT EXISTS account_journals (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(20) NOT NULL UNIQUE,
    type VARCHAR(30) NOT NULL, -- 'sale', 'purchase', 'cash', 'bank', 'general'
    default_account_id BIGINT REFERENCES account_accounts(id) ON DELETE RESTRICT,
    suspense_account_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL,
    sequence_prefix VARCHAR(20) NOT NULL,
    next_number INT NOT NULL DEFAULT 1,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_journals_code ON account_journals(code);
CREATE INDEX IF NOT EXISTS idx_account_journals_type ON account_journals(type);
CREATE INDEX IF NOT EXISTS idx_account_journals_active ON account_journals(active);

CREATE TRIGGER trg_account_journals_updated_at
    BEFORE UPDATE ON account_journals
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed Standard Journals
INSERT INTO account_journals (id, name, code, type, default_account_id, suspense_account_id, sequence_prefix, next_number, active) VALUES
(1, 'Customer Invoices', 'INV', 'sale', 10, NULL, 'INV/%Y/', 1, true),
(2, 'Vendor Bills', 'BILL', 'purchase', 13, NULL, 'BILL/%Y/', 1, true),
(3, 'Bank', 'BNK1', 'bank', 2, 2, 'BNK1/%Y/', 1, true),
(4, 'Cash', 'CSH1', 'cash', 1, 1, 'CSH1/%Y/', 1, true),
(5, 'Miscellaneous Operations', 'MISC', 'general', NULL, NULL, 'MISC/%Y/', 1, true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('account_journals_id_seq', (SELECT COALESCE(MAX(id), 1) FROM account_journals));

-- 3. Taxes (account.tax)
CREATE TABLE IF NOT EXISTS account_taxes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'percent', -- 'percent', 'fixed'
    type_tax_use VARCHAR(20) NOT NULL DEFAULT 'sale', -- 'sale', 'purchase', 'none'
    amount NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    account_id BIGINT REFERENCES account_accounts(id) ON DELETE RESTRICT,
    refund_account_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL,
    price_include BOOLEAN NOT NULL DEFAULT false,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_taxes_use ON account_taxes(type_tax_use);
CREATE INDEX IF NOT EXISTS idx_account_taxes_active ON account_taxes(active);

CREATE TRIGGER trg_account_taxes_updated_at
    BEFORE UPDATE ON account_taxes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed Standard Taxes (e.g. 15% VAT)
INSERT INTO account_taxes (id, name, type, type_tax_use, amount, account_id, refund_account_id, price_include, active) VALUES
(1, '15% Sales VAT', 'percent', 'sale', 15.0000, 7, 7, false, true),
(2, '15% Purchase VAT', 'percent', 'purchase', 15.0000, 4, 4, false, true),
(3, '15% VAT Included', 'percent', 'sale', 15.0000, 7, 7, true, true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('account_taxes_id_seq', (SELECT COALESCE(MAX(id), 1) FROM account_taxes));

-- 4. Payment Terms (account.payment.term)
CREATE TABLE IF NOT EXISTS account_payment_terms (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    note TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_payment_terms_active ON account_payment_terms(active);

CREATE TRIGGER trg_account_payment_terms_updated_at
    BEFORE UPDATE ON account_payment_terms
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS account_payment_term_lines (
    id BIGSERIAL PRIMARY KEY,
    payment_term_id BIGINT NOT NULL REFERENCES account_payment_terms(id) ON DELETE CASCADE,
    value_type VARCHAR(20) NOT NULL DEFAULT 'balance', -- 'balance', 'percent', 'fixed'
    value_amount NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    days INT NOT NULL DEFAULT 0,
    day_of_month INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_account_payment_term_lines_term_id ON account_payment_term_lines(payment_term_id);

-- Seed Standard Payment Terms
INSERT INTO account_payment_terms (id, name, note, active) VALUES
(1, 'Immediate Payment', 'Payment due immediately on issuance', true),
(2, '15 Days', 'Payment due within 15 calendar days', true),
(3, '30 Days', 'Payment due within 30 calendar days', true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO account_payment_term_lines (payment_term_id, value_type, value_amount, days) VALUES
(1, 'balance', 0, 0),
(2, 'balance', 0, 15),
(3, 'balance', 0, 30);

SELECT setval('account_payment_terms_id_seq', (SELECT COALESCE(MAX(id), 1) FROM account_payment_terms));

-- 5. Account Moves / Invoices (account.move)
CREATE TABLE IF NOT EXISTS account_moves (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL DEFAULT '/',
    move_type VARCHAR(30) NOT NULL DEFAULT 'entry', -- 'entry', 'out_invoice', 'out_refund', 'in_invoice', 'in_refund'
    journal_id BIGINT NOT NULL REFERENCES account_journals(id) ON DELETE RESTRICT,
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    invoice_date DATE,
    invoice_date_due DATE,
    payment_term_id BIGINT REFERENCES account_payment_terms(id) ON DELETE SET NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'draft', -- 'draft', 'posted', 'cancel'
    payment_state VARCHAR(20) NOT NULL DEFAULT 'not_paid', -- 'not_paid', 'in_payment', 'paid', 'partial', 'reversed'
    amount_untaxed NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    amount_tax NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    amount_total NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    amount_residual NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    ref VARCHAR(255),
    reversed_entry_id BIGINT REFERENCES account_moves(id) ON DELETE SET NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_account_moves_name ON account_moves(name);
CREATE INDEX IF NOT EXISTS idx_account_moves_move_type ON account_moves(move_type);
CREATE INDEX IF NOT EXISTS idx_account_moves_state ON account_moves(state);
CREATE INDEX IF NOT EXISTS idx_account_moves_date ON account_moves(date);
CREATE INDEX IF NOT EXISTS idx_account_moves_partner_id ON account_moves(partner_id);
CREATE INDEX IF NOT EXISTS idx_account_moves_journal_id ON account_moves(journal_id);
CREATE INDEX IF NOT EXISTS idx_account_moves_active ON account_moves(active);

CREATE TRIGGER trg_account_moves_updated_at
    BEFORE UPDATE ON account_moves
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 6. Account Move Lines (account.move.line)
CREATE TABLE IF NOT EXISTS account_move_lines (
    id BIGSERIAL PRIMARY KEY,
    move_id BIGINT NOT NULL REFERENCES account_moves(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES account_accounts(id) ON DELETE RESTRICT,
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    product_id BIGINT REFERENCES product_templates(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    quantity NUMERIC(15, 4) NOT NULL DEFAULT 1.0,
    price_unit NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    discount NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    debit NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    credit NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    balance NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    tax_ids BIGINT[] DEFAULT '{}',
    tax_amount NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_move_lines_move_id ON account_move_lines(move_id);
CREATE INDEX IF NOT EXISTS idx_account_move_lines_account_id ON account_move_lines(account_id);
CREATE INDEX IF NOT EXISTS idx_account_move_lines_partner_id ON account_move_lines(partner_id);

CREATE TRIGGER trg_account_move_lines_updated_at
    BEFORE UPDATE ON account_move_lines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
