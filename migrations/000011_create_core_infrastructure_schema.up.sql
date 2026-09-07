-- 000011_create_core_infrastructure_schema.up.sql
-- Core ERP Infrastructure: Currencies, Companies, Users, Groups, Sequences, Attachments, Config Parameters

-- ═══════════════════════════════════════════════════════════════════
-- 1. Currencies (res.currency in Odoo)
-- ═══════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS res_currencies (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(10) NOT NULL UNIQUE,
    full_name VARCHAR(100) NOT NULL,
    symbol VARCHAR(10) NOT NULL,
    decimal_places SMALLINT NOT NULL DEFAULT 2 CHECK (decimal_places >= 0 AND decimal_places <= 10),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_res_currencies_name ON res_currencies(name);
CREATE INDEX IF NOT EXISTS idx_res_currencies_active ON res_currencies(active);

CREATE TRIGGER trg_res_currencies_updated_at
    BEFORE UPDATE ON res_currencies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ═══════════════════════════════════════════════════════════════════
-- 2. Currency Rates (res.currency.rate in Odoo)
-- ═══════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS res_currency_rates (
    id BIGSERIAL PRIMARY KEY,
    currency_id BIGINT NOT NULL REFERENCES res_currencies(id) ON DELETE CASCADE,
    rate NUMERIC(20, 10) NOT NULL DEFAULT 1.0 CHECK (rate > 0),
    date DATE NOT NULL,
    company_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT,
    CONSTRAINT uq_currency_rate UNIQUE (currency_id, date, company_id)
);

CREATE INDEX IF NOT EXISTS idx_currency_rates_currency_id ON res_currency_rates(currency_id);
CREATE INDEX IF NOT EXISTS idx_currency_rates_date ON res_currency_rates(date);
CREATE INDEX IF NOT EXISTS idx_currency_rates_company_id ON res_currency_rates(company_id);

CREATE TRIGGER trg_res_currency_rates_updated_at
    BEFORE UPDATE ON res_currency_rates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ═══════════════════════════════════════════════════════════════════
-- 3. Companies (res.company in Odoo)
-- ═══════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS res_companies (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    partner_id BIGINT,
    currency_id BIGINT NOT NULL REFERENCES res_currencies(id) ON DELETE RESTRICT,
    phone VARCHAR(50),
    email VARCHAR(255),
    website VARCHAR(255),
    vat VARCHAR(100),
    street VARCHAR(255),
    street2 VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(100),
    country VARCHAR(100),
    zip_code VARCHAR(20),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_res_companies_name ON res_companies(name);
CREATE INDEX IF NOT EXISTS idx_res_companies_partner_id ON res_companies(partner_id);
CREATE INDEX IF NOT EXISTS idx_res_companies_currency_id ON res_companies(currency_id);
CREATE INDEX IF NOT EXISTS idx_res_companies_active ON res_companies(active);

CREATE TRIGGER trg_res_companies_updated_at
    BEFORE UPDATE ON res_companies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add FK from res_partners to res_companies via company_id
ALTER TABLE res_partners
    ADD CONSTRAINT fk_res_partners_company
    FOREIGN KEY (company_id)
    REFERENCES res_companies(id)
    ON DELETE RESTRICT;

-- Add FK from res_companies.partner_id to res_partners
ALTER TABLE res_companies
    ADD CONSTRAINT fk_res_companies_partner
    FOREIGN KEY (partner_id)
    REFERENCES res_partners(id)
    ON DELETE SET NULL;

-- ═══════════════════════════════════════════════════════════════════
-- 4. Users (res.users in Odoo)
-- ═══════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS res_users (
    id BIGSERIAL PRIMARY KEY,
    login VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    partner_id BIGINT NOT NULL,
    company_id BIGINT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    is_superuser BOOLEAN NOT NULL DEFAULT false,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_res_users_login ON res_users(login);
CREATE INDEX IF NOT EXISTS idx_res_users_email ON res_users(email);
CREATE INDEX IF NOT EXISTS idx_res_users_partner_id ON res_users(partner_id);
CREATE INDEX IF NOT EXISTS idx_res_users_company_id ON res_users(company_id);
CREATE INDEX IF NOT EXISTS idx_res_users_active ON res_users(active);

CREATE TRIGGER trg_res_users_updated_at
    BEFORE UPDATE ON res_users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- FK from res_users.partner_id -> res_partners
ALTER TABLE res_users
    ADD CONSTRAINT fk_res_users_partner
    FOREIGN KEY (partner_id)
    REFERENCES res_partners(id)
    ON DELETE RESTRICT;

-- FK from res_users.company_id -> res_companies
ALTER TABLE res_users
    ADD CONSTRAINT fk_res_users_company
    FOREIGN KEY (company_id)
    REFERENCES res_companies(id)
    ON DELETE RESTRICT;

-- ═══════════════════════════════════════════════════════════════════
-- 5. Groups (res.groups in Odoo)
-- ═══════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS res_groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_res_groups_name ON res_groups(name);
CREATE INDEX IF NOT EXISTS idx_res_groups_category ON res_groups(category);

CREATE TRIGGER trg_res_groups_updated_at
    BEFORE UPDATE ON res_groups
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Many-to-many: Users <-> Groups
CREATE TABLE IF NOT EXISTS res_groups_users_rel (
    group_id BIGINT NOT NULL REFERENCES res_groups(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES res_users(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_groups_users_rel_user ON res_groups_users_rel(user_id);

-- ═══════════════════════════════════════════════════════════════════
-- 6. Sequences (ir.sequence in Odoo)
-- ═══════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS ir_sequences (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(100) NOT NULL UNIQUE,
    prefix VARCHAR(50),
    suffix VARCHAR(50),
    padding SMALLINT NOT NULL DEFAULT 5 CHECK (padding >= 1 AND padding <= 20),
    increment_by INTEGER NOT NULL DEFAULT 1 CHECK (increment_by >= 1),
    start_number INTEGER NOT NULL DEFAULT 1,
    current_number INTEGER NOT NULL DEFAULT 0,
    sequence_type VARCHAR(20) NOT NULL DEFAULT 'normal', -- 'normal', 'date_range'
    date_range VARCHAR(10), -- 'year', 'month', 'day' (used when sequence_type = 'date_range')
    company_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ir_sequences_code ON ir_sequences(code);
CREATE INDEX IF NOT EXISTS idx_ir_sequences_company_id ON ir_sequences(company_id);
CREATE INDEX IF NOT EXISTS idx_ir_sequences_active ON ir_sequences(active);

CREATE TRIGGER trg_ir_sequences_updated_at
    BEFORE UPDATE ON ir_sequences
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ═══════════════════════════════════════════════════════════════════
-- 7. Attachments (ir.attachment in Odoo)
-- ═══════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS ir_attachments (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    mimetype VARCHAR(255) NOT NULL DEFAULT 'application/octet-stream',
    file_size BIGINT NOT NULL DEFAULT 0 CHECK (file_size >= 0),
    checksum VARCHAR(40), -- SHA-1 hash
    storage_path VARCHAR(500),
    res_model VARCHAR(100),
    res_id BIGINT,
    description TEXT,
    company_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_ir_attachments_name ON ir_attachments(name);
CREATE INDEX IF NOT EXISTS idx_ir_attachments_checksum ON ir_attachments(checksum);
CREATE INDEX IF NOT EXISTS idx_ir_attachments_res_model_id ON ir_attachments(res_model, res_id);
CREATE INDEX IF NOT EXISTS idx_ir_attachments_company_id ON ir_attachments(company_id);
CREATE INDEX IF NOT EXISTS idx_ir_attachments_active ON ir_attachments(active);

CREATE TRIGGER trg_ir_attachments_updated_at
    BEFORE UPDATE ON ir_attachments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ═══════════════════════════════════════════════════════════════════
-- 8. System Parameters (ir.config_parameter in Odoo)
-- ═══════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS ir_config_parameters (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(255) NOT NULL UNIQUE,
    value TEXT NOT NULL DEFAULT '',
    company_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ir_config_parameters_key ON ir_config_parameters(key);
CREATE INDEX IF NOT EXISTS idx_ir_config_parameters_company_id ON ir_config_parameters(company_id);

CREATE TRIGGER trg_ir_config_parameters_updated_at
    BEFORE UPDATE ON ir_config_parameters
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ═══════════════════════════════════════════════════════════════════
-- 9. Seed Data
-- ═══════════════════════════════════════════════════════════════════

-- Default currencies
INSERT INTO res_currencies (id, name, full_name, symbol, decimal_places) VALUES
    (1, 'USD', 'United States Dollar', '$', 2),
    (2, 'EUR', 'Euro', '€', 2),
    (3, 'SAR', 'Saudi Riyal', 'ر.س', 2)
ON CONFLICT (name) DO NOTHING;

SELECT setval('res_currencies_id_seq', (SELECT COALESCE(MAX(id), 0) FROM res_currencies));

-- Default currency rates (base date)
INSERT INTO res_currency_rates (currency_id, rate, date) VALUES
    (1, 1.0, CURRENT_DATE),
    (2, 0.92, CURRENT_DATE),
    (3, 3.75, CURRENT_DATE)
ON CONFLICT (currency_id, date, company_id) DO NOTHING;

-- Default partner for admin (res_partners already exists)
INSERT INTO res_partners (name, email, type, is_customer, active, created_at, updated_at)
VALUES ('Administrator', 'admin@example.com', 'individual', false, true, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- Get the admin partner ID (either newly inserted or existing)
-- Default company
INSERT INTO res_companies (id, name, partner_id, currency_id, email, active)
SELECT 1, 'Main Company',
    (SELECT id FROM res_partners WHERE email = 'admin@example.com' LIMIT 1),
    1,
    'admin@example.com',
    true
WHERE NOT EXISTS (SELECT 1 FROM res_companies WHERE id = 1)
ON CONFLICT DO NOTHING;

SELECT setval('res_companies_id_seq', (SELECT COALESCE(MAX(id), 0) FROM res_companies));

-- Default admin user (password: admin123 - bcrypt hash of "admin123")
INSERT INTO res_users (id, login, email, name, password_hash, partner_id, company_id, is_superuser, active)
SELECT 1, 'admin', 'admin@example.com', 'Administrator',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    (SELECT id FROM res_partners WHERE email = 'admin@example.com' LIMIT 1),
    1,
    true,
    true
WHERE NOT EXISTS (SELECT 1 FROM res_users WHERE id = 1)
ON CONFLICT DO NOTHING;

SELECT setval('res_users_id_seq', (SELECT COALESCE(MAX(id), 0) FROM res_users));

-- Default groups
INSERT INTO res_groups (id, name, category) VALUES
    (1, 'User', 'internal'),
    (2, 'Administrator', 'internal')
ON CONFLICT DO NOTHING;

SELECT setval('res_groups_id_seq', (SELECT COALESCE(MAX(id), 0) FROM res_groups));

-- Assign Administrator group to admin user
INSERT INTO res_groups_users_rel (group_id, user_id) VALUES (2, 1)
ON CONFLICT DO NOTHING;

-- Standard document sequences
INSERT INTO ir_sequences (name, code, prefix, suffix, padding, start_number, current_number, sequence_type, company_id) VALUES
    ('Customer Invoices', 'account.move.customer_invoice', 'INV/', '', 5, 1, 0, 'normal', 1),
    ('Vendor Bills', 'account.move.vendor_bill', 'BILL/', '', 5, 1, 0, 'normal', 1),
    ('Sales Orders', 'sale.order', 'SO/', '', 5, 1, 0, 'normal', 1),
    ('Purchase Orders', 'purchase.order', 'PO/', '', 5, 1, 0, 'normal', 1),
    ('Stock Pickings', 'stock.picking', 'WH/OUT/', '', 5, 1, 0, 'normal', 1),
    ('Payments', 'account.payment', 'PAY/', '', 5, 1, 0, 'normal', 1),
    ('CRM Leads', 'crm.lead', 'LEAD/', '', 5, 1, 0, 'normal', 1),
    ('HR Employees', 'hr.employee', 'EMP/', '', 5, 1, 0, 'normal', 1),
    ('Product Internal Reference', 'product.product', 'PRD/', '', 5, 1, 0, 'normal', 1),
    ('Attachments', 'ir.attachment', 'ATT/', '', 5, 1, 0, 'normal', 1)
ON CONFLICT (code) DO NOTHING;

-- System configuration parameters
INSERT INTO ir_config_parameters (key, value, company_id) VALUES
    ('system.name', 'CashFlow ERP', 1),
    ('system.version', '11.0.0', 1),
    ('default.currency_id', '1', 1),
    ('default.company_id', '1', 1)
ON CONFLICT (key) DO NOTHING;

-- ═══════════════════════════════════════════════════════════════════
-- 10. Add Foreign Keys on existing tables (company_id -> res_companies)
-- ═══════════════════════════════════════════════════════════════════

ALTER TABLE product_templates
    ADD CONSTRAINT fk_product_templates_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE account_accounts
    ADD CONSTRAINT fk_account_accounts_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE sale_orders
    ADD CONSTRAINT fk_sale_orders_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE purchase_orders
    ADD CONSTRAINT fk_purchase_orders_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE stock_locations
    ADD CONSTRAINT fk_stock_locations_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE stock_warehouses
    ADD CONSTRAINT fk_stock_warehouses_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE stock_pickings
    ADD CONSTRAINT fk_stock_pickings_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE stock_quants
    ADD CONSTRAINT fk_stock_quants_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE crm_stages
    ADD CONSTRAINT fk_crm_stages_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE crm_leads
    ADD CONSTRAINT fk_crm_leads_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE account_payments
    ADD CONSTRAINT fk_account_payments_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE hr_departments
    ADD CONSTRAINT fk_hr_departments_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE hr_jobs
    ADD CONSTRAINT fk_hr_jobs_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE hr_employees
    ADD CONSTRAINT fk_hr_employees_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE hr_leave_allocations
    ADD CONSTRAINT fk_hr_leave_allocations_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;

ALTER TABLE hr_leave_requests
    ADD CONSTRAINT fk_hr_leave_requests_company
    FOREIGN KEY (company_id) REFERENCES res_companies(id) ON DELETE RESTRICT;
