-- ===========================================================================
-- Source: 000001_init_schema.up.sql
-- ===========================================================================
-- 000001_init_schema.up.sql
-- Base database configuration, extensions, and standard ERP triggers

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";

-- Standard automated trigger function for touch-updating updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- ===========================================================================
-- Standard Geopolitical tables (referenced by delivery_carrier)
-- ===========================================================================

CREATE TABLE IF NOT EXISTS res_country (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(2) NOT NULL UNIQUE,
    active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS res_country_state (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(10) NOT NULL,
    country_id BIGINT NOT NULL REFERENCES res_country(id) ON DELETE CASCADE,
    active BOOLEAN DEFAULT TRUE,
    CONSTRAINT uq_res_country_state_code UNIQUE (country_id, code)
);

-- Seed basic countries
INSERT INTO res_country (id, name, code) VALUES
(1, 'United States', 'US'),
(2, 'Saudi Arabia', 'SA'),
(3, 'United Kingdom', 'GB'),
(4, 'Germany', 'DE'),
(5, 'France', 'FR')
ON CONFLICT (id) DO NOTHING;

SELECT setval('res_country_id_seq', (SELECT COALESCE(MAX(id), 1) FROM res_country));


-- ===========================================================================
-- Source: 000002_create_partners_table.up.sql
-- ===========================================================================
-- 000002_create_partners_table.up.sql
-- Res Partners table representing contacts, customers, suppliers, and companies

CREATE TABLE IF NOT EXISTS res_partners (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email CITEXT,
    phone VARCHAR(50),
    mobile VARCHAR(50),
    type VARCHAR(20) NOT NULL DEFAULT 'individual',
    is_customer BOOLEAN NOT NULL DEFAULT true,
    is_supplier BOOLEAN NOT NULL DEFAULT false,
    vat_number VARCHAR(50),
    website VARCHAR(255),
    company_id BIGINT,
    parent_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    street VARCHAR(255),
    street2 VARCHAR(255),
    state_id BIGINT REFERENCES res_country_state(id),
    country_id BIGINT REFERENCES res_country(id),
    zip_code VARCHAR(20),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

-- Indexes for performant searching and filtering
CREATE INDEX IF NOT EXISTS idx_partners_name ON res_partners(name);
CREATE INDEX IF NOT EXISTS idx_partners_email ON res_partners(email);
CREATE INDEX IF NOT EXISTS idx_partners_active ON res_partners(active);
CREATE INDEX IF NOT EXISTS idx_partners_is_customer ON res_partners(is_customer) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_partners_is_supplier ON res_partners(is_supplier) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_partners_company_id ON res_partners(company_id);
CREATE INDEX IF NOT EXISTS idx_partners_parent_id ON res_partners(parent_id);

-- Trigger for auto-updating updated_at timestamp
CREATE TRIGGER trg_partners_updated_at
    BEFORE UPDATE ON res_partners
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();


-- ===========================================================================
-- Source: 000003_create_products_schema.up.sql
-- ===========================================================================
-- 000003_create_products_schema.up.sql
-- Product catalog, categories, units of measure, variants, and pricelists schema

-- 1. Units of Measure (UoM)
CREATE TABLE IF NOT EXISTS uom_uoms (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL, -- 'unit', 'weight', 'volume', 'length', 'time'
    ratio NUMERIC(15, 6) NOT NULL DEFAULT 1.0,
    rounding NUMERIC(15, 6) NOT NULL DEFAULT 0.001,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_uom_name ON uom_uoms(name);
CREATE INDEX IF NOT EXISTS idx_uom_category ON uom_uoms(category);
CREATE INDEX IF NOT EXISTS idx_uom_active ON uom_uoms(active);

CREATE TRIGGER trg_uom_updated_at
    BEFORE UPDATE ON uom_uoms
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed standard Units of Measure
INSERT INTO uom_uoms (id, name, category, ratio, rounding, active) VALUES
(1, 'Units', 'unit', 1.0, 0.001, true),
(2, 'Dozens', 'unit', 12.0, 0.001, true),
(3, 'kg', 'weight', 1.0, 0.001, true),
(4, 'g', 'weight', 0.001, 0.001, true),
(5, 'Liters', 'volume', 1.0, 0.001, true),
(6, 'Hours', 'time', 1.0, 0.01, true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('uom_uoms_id_seq', (SELECT COALESCE(MAX(id), 1) FROM uom_uoms));

-- 2. Product Categories
CREATE TABLE IF NOT EXISTS product_categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    parent_id BIGINT REFERENCES product_categories(id) ON DELETE SET NULL,
    complete_name VARCHAR(500) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_product_categories_name ON product_categories(name);
CREATE INDEX IF NOT EXISTS idx_product_categories_parent_id ON product_categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_product_categories_active ON product_categories(active);

CREATE TRIGGER trg_product_categories_updated_at
    BEFORE UPDATE ON product_categories
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed standard Root Category
INSERT INTO product_categories (id, name, parent_id, complete_name, active) VALUES
(1, 'All', NULL, 'All', true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('product_categories_id_seq', (SELECT COALESCE(MAX(id), 1) FROM product_categories));

-- 3. Product Templates (Master Product)
CREATE TABLE IF NOT EXISTS product_templates (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'consu', -- 'consu' (goods), 'service', 'combo'
    category_id BIGINT REFERENCES product_categories(id) ON DELETE RESTRICT,
    internal_ref VARCHAR(100), -- SKU
    barcode VARCHAR(100),
    sale_price NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    cost_price NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    uom_id BIGINT REFERENCES uom_uoms(id) ON DELETE RESTRICT,
    sale_ok BOOLEAN NOT NULL DEFAULT true,
    purchase_ok BOOLEAN NOT NULL DEFAULT true,
    weight NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    volume NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    description TEXT,
    company_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_product_templates_name ON product_templates(name);
CREATE INDEX IF NOT EXISTS idx_product_templates_internal_ref ON product_templates(internal_ref);
CREATE INDEX IF NOT EXISTS idx_product_templates_barcode ON product_templates(barcode);
CREATE INDEX IF NOT EXISTS idx_product_templates_category_id ON product_templates(category_id);
CREATE INDEX IF NOT EXISTS idx_product_templates_active ON product_templates(active);
CREATE INDEX IF NOT EXISTS idx_product_templates_sale_ok ON product_templates(sale_ok) WHERE active = true;

CREATE TRIGGER trg_product_templates_updated_at
    BEFORE UPDATE ON product_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 4. Product Attributes & Values
CREATE TABLE IF NOT EXISTS product_attributes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    sequence INT NOT NULL DEFAULT 10,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS product_attribute_values (
    id BIGSERIAL PRIMARY KEY,
    attribute_id BIGINT NOT NULL REFERENCES product_attributes(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    sequence INT NOT NULL DEFAULT 10,
    extra_price NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_attribute_values_attr_id ON product_attribute_values(attribute_id);

-- 5. Product Variants (Specific Variant Instance)
CREATE TABLE IF NOT EXISTS product_variants (
    id BIGSERIAL PRIMARY KEY,
    template_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE CASCADE,
    sku VARCHAR(100),
    barcode VARCHAR(100),
    extra_price NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_product_variants_template_id ON product_variants(template_id);
CREATE INDEX IF NOT EXISTS idx_product_variants_sku ON product_variants(sku);
CREATE INDEX IF NOT EXISTS idx_product_variants_barcode ON product_variants(barcode);
CREATE INDEX IF NOT EXISTS idx_product_variants_active ON product_variants(active);

CREATE TRIGGER trg_product_variants_updated_at
    BEFORE UPDATE ON product_variants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 6. Product Variant Attributes Mapping
CREATE TABLE IF NOT EXISTS product_variant_attributes (
    variant_id BIGINT NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    attribute_value_id BIGINT NOT NULL REFERENCES product_attribute_values(id) ON DELETE CASCADE,
    PRIMARY KEY (variant_id, attribute_value_id)
);

-- 7. Pricelists and Items
CREATE TABLE IF NOT EXISTS product_pricelists (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_product_pricelists_active ON product_pricelists(active);

CREATE TRIGGER trg_product_pricelists_updated_at
    BEFORE UPDATE ON product_pricelists
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed standard Public Pricelist
INSERT INTO product_pricelists (id, name, currency, active) VALUES
(1, 'Public Pricelist', 'USD', true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('product_pricelists_id_seq', (SELECT COALESCE(MAX(id), 1) FROM product_pricelists));

CREATE TABLE IF NOT EXISTS product_pricelist_items (
    id BIGSERIAL PRIMARY KEY,
    pricelist_id BIGINT NOT NULL REFERENCES product_pricelists(id) ON DELETE CASCADE,
    applied_on VARCHAR(20) NOT NULL DEFAULT 'all', -- 'all', 'category', 'template', 'variant'
    category_id BIGINT REFERENCES product_categories(id) ON DELETE CASCADE,
    template_id BIGINT REFERENCES product_templates(id) ON DELETE CASCADE,
    variant_id BIGINT REFERENCES product_variants(id) ON DELETE CASCADE,
    min_quantity NUMERIC(15, 4) NOT NULL DEFAULT 1.0,
    compute_price VARCHAR(20) NOT NULL DEFAULT 'fixed', -- 'fixed', 'percentage', 'formula'
    fixed_price NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    percent_price NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    date_start TIMESTAMPTZ,
    date_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pricelist_items_pricelist_id ON product_pricelist_items(pricelist_id);
CREATE INDEX IF NOT EXISTS idx_pricelist_items_category_id ON product_pricelist_items(category_id);
CREATE INDEX IF NOT EXISTS idx_pricelist_items_template_id ON product_pricelist_items(template_id);
CREATE INDEX IF NOT EXISTS idx_pricelist_items_variant_id ON product_pricelist_items(variant_id);

CREATE TRIGGER trg_product_pricelist_items_updated_at
    BEFORE UPDATE ON product_pricelist_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();


-- ===========================================================================
-- Source: 000004_create_accounting_schema.up.sql
-- ===========================================================================
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
    tag_ids JSONB,
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


-- ===========================================================================
-- Source: 000005_create_sales_schema.up.sql
-- ===========================================================================
-- 000005_create_sales_schema.up.sql
-- Sales Order schema: Quotations, Sale Orders, Lines, and Invoice Junction

-- 1. Sequence for Sales Orders
CREATE SEQUENCE IF NOT EXISTS sale_order_seq START 1;

-- 2. Sale Orders (sale.order in Odoo)
CREATE TABLE IF NOT EXISTS sale_orders (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    partner_id BIGINT NOT NULL REFERENCES res_partners(id) ON DELETE RESTRICT,
    date_order TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    validity_date TIMESTAMPTZ,
    state VARCHAR(20) NOT NULL DEFAULT 'draft', -- 'draft', 'sent', 'sale', 'done', 'cancel'
    invoice_status VARCHAR(20) NOT NULL DEFAULT 'no', -- 'no', 'to_invoice', 'invoiced'
    pricelist_id BIGINT REFERENCES product_pricelists(id) ON DELETE SET NULL,
    payment_term_id BIGINT REFERENCES account_payment_terms(id) ON DELETE SET NULL,
    user_id BIGINT,
    company_id BIGINT,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    note TEXT,
    amount_untaxed NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    amount_tax NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    amount_total NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    tag_ids JSONB,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_sale_orders_name ON sale_orders(name);
CREATE INDEX IF NOT EXISTS idx_sale_orders_partner_id ON sale_orders(partner_id);
CREATE INDEX IF NOT EXISTS idx_sale_orders_state ON sale_orders(state);
CREATE INDEX IF NOT EXISTS idx_sale_orders_invoice_status ON sale_orders(invoice_status);
CREATE INDEX IF NOT EXISTS idx_sale_orders_date_order ON sale_orders(date_order);
CREATE INDEX IF NOT EXISTS idx_sale_orders_active ON sale_orders(active);

CREATE TRIGGER trg_sale_orders_updated_at
    BEFORE UPDATE ON sale_orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 3. Sale Order Lines (sale.order.line in Odoo)
CREATE TABLE IF NOT EXISTS sale_order_lines (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES sale_orders(id) ON DELETE CASCADE,
    sequence INT NOT NULL DEFAULT 10,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    product_uom_qty NUMERIC(15, 4) NOT NULL DEFAULT 1.0000,
    product_uom BIGINT REFERENCES uom_uoms(id) ON DELETE SET NULL,
    price_unit NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    discount NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    tax_ids BIGINT[] DEFAULT '{}',
    price_subtotal NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    price_tax NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    price_total NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    qty_delivered NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    qty_invoiced NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sale_order_lines_order_id ON sale_order_lines(order_id);
CREATE INDEX IF NOT EXISTS idx_sale_order_lines_product_id ON sale_order_lines(product_id);

CREATE TRIGGER trg_sale_order_lines_updated_at
    BEFORE UPDATE ON sale_order_lines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 4. Sale Order Invoices Junction
CREATE TABLE IF NOT EXISTS sale_order_invoices (
    order_id BIGINT NOT NULL REFERENCES sale_orders(id) ON DELETE CASCADE,
    move_id BIGINT NOT NULL REFERENCES account_moves(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (order_id, move_id)
);

CREATE INDEX IF NOT EXISTS idx_sale_order_invoices_move_id ON sale_order_invoices(move_id);


-- ===========================================================================
-- Source: 000006_create_purchase_schema.up.sql
-- ===========================================================================
-- 000006_create_purchase_schema.up.sql
-- Purchase Order schema: RFQ, Purchase Orders, Lines, and Bill Junction

-- 1. Sequence for Purchase Orders
CREATE SEQUENCE IF NOT EXISTS purchase_order_seq START 1;

-- 2. Purchase Orders (purchase.order in Odoo)
CREATE TABLE IF NOT EXISTS purchase_orders (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    partner_id BIGINT NOT NULL REFERENCES res_partners(id) ON DELETE RESTRICT,
    date_order TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    date_planned TIMESTAMPTZ,
    state VARCHAR(20) NOT NULL DEFAULT 'draft', -- 'draft', 'sent', 'purchase', 'done', 'cancel'
    invoice_status VARCHAR(20) NOT NULL DEFAULT 'no', -- 'no', 'to_invoice', 'invoiced'
    payment_term_id BIGINT REFERENCES account_payment_terms(id) ON DELETE SET NULL,
    user_id BIGINT,
    company_id BIGINT,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    note TEXT,
    amount_untaxed NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    amount_tax NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    amount_total NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    tag_ids JSONB,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_name ON purchase_orders(name);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_partner_id ON purchase_orders(partner_id);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_state ON purchase_orders(state);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_invoice_status ON purchase_orders(invoice_status);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_date_order ON purchase_orders(date_order);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_active ON purchase_orders(active);

CREATE TRIGGER trg_purchase_orders_updated_at
    BEFORE UPDATE ON purchase_orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 3. Purchase Order Lines (purchase.order.line in Odoo)
CREATE TABLE IF NOT EXISTS purchase_order_lines (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    sequence INT NOT NULL DEFAULT 10,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    product_qty NUMERIC(15, 4) NOT NULL DEFAULT 1.0000,
    product_uom BIGINT REFERENCES uom_uoms(id) ON DELETE SET NULL,
    price_unit NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    discount NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    tax_ids BIGINT[] DEFAULT '{}',
    price_subtotal NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    price_tax NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    price_total NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    qty_received NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    qty_invoiced NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_order_lines_order_id ON purchase_order_lines(order_id);
CREATE INDEX IF NOT EXISTS idx_purchase_order_lines_product_id ON purchase_order_lines(product_id);

CREATE TRIGGER trg_purchase_order_lines_updated_at
    BEFORE UPDATE ON purchase_order_lines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 4. Purchase Order Invoices (Vendor Bills) Junction
CREATE TABLE IF NOT EXISTS purchase_order_invoices (
    order_id BIGINT NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    move_id BIGINT NOT NULL REFERENCES account_moves(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (order_id, move_id)
);

CREATE INDEX IF NOT EXISTS idx_purchase_order_invoices_move_id ON purchase_order_invoices(move_id);


-- ===========================================================================
-- Source: 000007_create_stock_schema.up.sql
-- ===========================================================================
-- 000007_create_stock_schema.up.sql
-- Inventory & Stock Management Schema: Locations, Warehouses, Pickings, Moves, and Quants

-- 1. Sequences for Stock Pickings
CREATE SEQUENCE IF NOT EXISTS stock_picking_in_seq START 1;
CREATE SEQUENCE IF NOT EXISTS stock_picking_out_seq START 1;
CREATE SEQUENCE IF NOT EXISTS stock_picking_int_seq START 1;

-- 2. Stock Locations (stock.location in Odoo)
CREATE TABLE IF NOT EXISTS stock_locations (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    complete_name VARCHAR(255) NOT NULL,
    usage VARCHAR(32) NOT NULL DEFAULT 'internal', -- 'internal', 'supplier', 'customer', 'inventory', 'transit', 'production', 'view'
    parent_id BIGINT REFERENCES stock_locations(id) ON DELETE RESTRICT,
    scrap_location BOOLEAN NOT NULL DEFAULT false,
    return_location BOOLEAN NOT NULL DEFAULT false,
    company_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_stock_locations_usage ON stock_locations(usage);
CREATE INDEX IF NOT EXISTS idx_stock_locations_parent_id ON stock_locations(parent_id);
CREATE INDEX IF NOT EXISTS idx_stock_locations_active ON stock_locations(active);

CREATE TRIGGER trg_stock_locations_updated_at
    BEFORE UPDATE ON stock_locations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed Default Standard Locations
INSERT INTO stock_locations (id, name, complete_name, usage, parent_id, active)
VALUES 
    (1, 'Partner Locations', 'Partner Locations', 'view', NULL, true),
    (2, 'Vendors', 'Partner Locations/Vendors', 'supplier', 1, true),
    (3, 'Customers', 'Partner Locations/Customers', 'customer', 1, true),
    (4, 'Virtual Locations', 'Virtual Locations', 'view', NULL, true),
    (5, 'Inventory adjustment', 'Virtual Locations/Inventory adjustment', 'inventory', 4, true),
    (6, 'Scrap', 'Virtual Locations/Scrap', 'inventory', 4, true),
    (7, 'WH', 'WH', 'view', NULL, true),
    (8, 'Stock', 'WH/Stock', 'internal', 7, true),
    (9, 'Input', 'WH/Input', 'internal', 7, true),
    (10, 'Output', 'WH/Output', 'internal', 7, true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('stock_locations_id_seq', (SELECT COALESCE(MAX(id), 1) FROM stock_locations));

-- 3. Stock Warehouses (stock.warehouse in Odoo)
CREATE TABLE IF NOT EXISTS stock_warehouses (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    code VARCHAR(16) NOT NULL UNIQUE,
    company_id BIGINT,
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    view_location_id BIGINT REFERENCES stock_locations(id) ON DELETE RESTRICT,
    lot_stock_id BIGINT REFERENCES stock_locations(id) ON DELETE RESTRICT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_stock_warehouses_code ON stock_warehouses(code);
CREATE INDEX IF NOT EXISTS idx_stock_warehouses_active ON stock_warehouses(active);

CREATE TRIGGER trg_stock_warehouses_updated_at
    BEFORE UPDATE ON stock_warehouses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed Default Warehouse
INSERT INTO stock_warehouses (id, name, code, view_location_id, lot_stock_id, active)
VALUES (1, 'Main Warehouse', 'WH', 7, 8, true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('stock_warehouses_id_seq', (SELECT COALESCE(MAX(id), 1) FROM stock_warehouses));

-- 4. Stock Pickings (stock.picking in Odoo: Receipts, Deliveries, Internal Transfers)
CREATE TABLE IF NOT EXISTS stock_pickings (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    picking_type VARCHAR(20) NOT NULL, -- 'incoming', 'outgoing', 'internal'
    state VARCHAR(20) NOT NULL DEFAULT 'draft', -- 'draft', 'waiting', 'confirmed', 'assigned', 'done', 'cancel'
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    location_id BIGINT NOT NULL REFERENCES stock_locations(id) ON DELETE RESTRICT,
    location_dest_id BIGINT NOT NULL REFERENCES stock_locations(id) ON DELETE RESTRICT,
    scheduled_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    date_done TIMESTAMPTZ,
    origin VARCHAR(128),
    source_order_id BIGINT,
    company_id BIGINT,
    note TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_stock_pickings_name ON stock_pickings(name);
CREATE INDEX IF NOT EXISTS idx_stock_pickings_picking_type ON stock_pickings(picking_type);
CREATE INDEX IF NOT EXISTS idx_stock_pickings_state ON stock_pickings(state);
CREATE INDEX IF NOT EXISTS idx_stock_pickings_partner_id ON stock_pickings(partner_id);
CREATE INDEX IF NOT EXISTS idx_stock_pickings_location_id ON stock_pickings(location_id);
CREATE INDEX IF NOT EXISTS idx_stock_pickings_location_dest_id ON stock_pickings(location_dest_id);
CREATE INDEX IF NOT EXISTS idx_stock_pickings_origin ON stock_pickings(origin);
CREATE INDEX IF NOT EXISTS idx_stock_pickings_active ON stock_pickings(active);

CREATE TRIGGER trg_stock_pickings_updated_at
    BEFORE UPDATE ON stock_pickings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 5. Stock Moves (stock.move in Odoo)
CREATE TABLE IF NOT EXISTS stock_moves (
    id BIGSERIAL PRIMARY KEY,
    picking_id BIGINT REFERENCES stock_pickings(id) ON DELETE CASCADE,
    sequence INT NOT NULL DEFAULT 10,
    name VARCHAR(255) NOT NULL,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    product_uom BIGINT REFERENCES uom_uoms(id) ON DELETE SET NULL,
    product_qty NUMERIC(15, 4) NOT NULL DEFAULT 1.0000,
    quantity_done NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    location_id BIGINT NOT NULL REFERENCES stock_locations(id) ON DELETE RESTRICT,
    location_dest_id BIGINT NOT NULL REFERENCES stock_locations(id) ON DELETE RESTRICT,
    state VARCHAR(20) NOT NULL DEFAULT 'draft', -- 'draft', 'waiting', 'confirmed', 'assigned', 'done', 'cancel'
    sale_line_id BIGINT REFERENCES sale_order_lines(id) ON DELETE SET NULL,
    purchase_line_id BIGINT REFERENCES purchase_order_lines(id) ON DELETE SET NULL,
    date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_moves_picking_id ON stock_moves(picking_id);
CREATE INDEX IF NOT EXISTS idx_stock_moves_product_id ON stock_moves(product_id);
CREATE INDEX IF NOT EXISTS idx_stock_moves_location_id ON stock_moves(location_id);
CREATE INDEX IF NOT EXISTS idx_stock_moves_location_dest_id ON stock_moves(location_dest_id);
CREATE INDEX IF NOT EXISTS idx_stock_moves_state ON stock_moves(state);

CREATE TRIGGER trg_stock_moves_updated_at
    BEFORE UPDATE ON stock_moves
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 6. Stock Quants (stock.quant in Odoo: Physical On-Hand Stock Levels)
CREATE TABLE IF NOT EXISTS stock_quants (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    location_id BIGINT NOT NULL REFERENCES stock_locations(id) ON DELETE RESTRICT,
    quantity NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    reserved_quantity NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    company_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_stock_quants_product_location UNIQUE (product_id, location_id)
);

CREATE INDEX IF NOT EXISTS idx_stock_quants_product_id ON stock_quants(product_id);
CREATE INDEX IF NOT EXISTS idx_stock_quants_location_id ON stock_quants(location_id);

CREATE TRIGGER trg_stock_quants_updated_at
    BEFORE UPDATE ON stock_quants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();


-- ===========================================================================
-- Source: 000008_create_crm_schema.up.sql
-- ===========================================================================
-- 000008_create_crm_schema.up.sql
-- CRM & Sales Pipeline Schema: Stages, Lost Reasons, Tags, Leads & Opportunities

-- 1. Stages (crm.stage in Odoo)
CREATE TABLE IF NOT EXISTS crm_stages (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    sequence INT NOT NULL DEFAULT 10,
    is_won BOOLEAN NOT NULL DEFAULT false,
    is_closed BOOLEAN NOT NULL DEFAULT false,
    fold BOOLEAN NOT NULL DEFAULT false,
    requirements TEXT,
    company_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_crm_stages_sequence ON crm_stages(sequence);
CREATE INDEX IF NOT EXISTS idx_crm_stages_active ON crm_stages(active);

CREATE TRIGGER trg_crm_stages_updated_at
    BEFORE UPDATE ON crm_stages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed Default Pipeline Stages
INSERT INTO crm_stages (id, name, sequence, is_won, is_closed, fold, active)
VALUES 
    (1, 'New', 10, false, false, false, true),
    (2, 'Qualified', 20, false, false, false, true),
    (3, 'Proposition', 30, false, false, false, true),
    (4, 'Won', 40, true, true, false, true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('crm_stages_id_seq', (SELECT COALESCE(MAX(id), 1) FROM crm_stages));

-- 2. Lost Reasons (crm.lost.reason in Odoo)
CREATE TABLE IF NOT EXISTS crm_lost_reasons (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_crm_lost_reasons_active ON crm_lost_reasons(active);

CREATE TRIGGER trg_crm_lost_reasons_updated_at
    BEFORE UPDATE ON crm_lost_reasons
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed Default Lost Reasons
INSERT INTO crm_lost_reasons (id, name, active)
VALUES 
    (1, 'Too expensive', true),
    (2, 'We don''t have people/skills', true),
    (3, 'Not enough features', true),
    (4, 'Lost to competitor', true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('crm_lost_reasons_id_seq', (SELECT COALESCE(MAX(id), 1) FROM crm_lost_reasons));

-- 3. CRM Tags (crm.tag in Odoo) - Removed in favor of JSONB tag_ids


-- 4. Leads & Opportunities (crm.lead in Odoo)
CREATE TABLE IF NOT EXISTS crm_leads (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'lead', -- 'lead', 'opportunity'
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    partner_name VARCHAR(255),
    contact_name VARCHAR(255),
    email_from CITEXT,
    phone VARCHAR(50),
    stage_id BIGINT NOT NULL REFERENCES crm_stages(id) ON DELETE RESTRICT,
    salesperson_id BIGINT,
    expected_revenue NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    prorated_revenue NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    probability NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    source VARCHAR(64),
    priority VARCHAR(20) NOT NULL DEFAULT '1', -- '0'=Low, '1'=Medium, '2'=High, '3'=Very High
    lost_reason_id BIGINT REFERENCES crm_lost_reasons(id) ON DELETE SET NULL,
    lost_feedback TEXT,
    date_deadline TIMESTAMPTZ,
    date_closed TIMESTAMPTZ,
    date_conversion TIMESTAMPTZ,
    notes TEXT,
    company_id BIGINT,
    tag_ids JSONB,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_crm_leads_name ON crm_leads(name);


-- ===========================================================================
-- Source: 000009_create_payments_schema.up.sql
-- ===========================================================================
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


-- ===========================================================================
-- Source: 000010_create_hr_schema.up.sql
-- ===========================================================================
-- 000010_create_hr_schema.up.sql
-- Human Resources (HR) Module: Departments, Jobs, Employees, Leave Allocations, and Leave Requests

-- 1. Departments (hr.department in Odoo)
CREATE TABLE IF NOT EXISTS hr_departments (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    complete_name VARCHAR(500),
    parent_id BIGINT REFERENCES hr_departments(id) ON DELETE SET NULL,
    manager_id BIGINT, -- Foreign key to hr_employees(id) added below
    company_id BIGINT,
    color INTEGER DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_hr_departments_name ON hr_departments(name);
CREATE INDEX IF NOT EXISTS idx_hr_departments_parent_id ON hr_departments(parent_id);
CREATE INDEX IF NOT EXISTS idx_hr_departments_active ON hr_departments(active);

CREATE TRIGGER trg_hr_departments_updated_at
    BEFORE UPDATE ON hr_departments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 2. Jobs / Job Positions (hr.job in Odoo)
CREATE TABLE IF NOT EXISTS hr_jobs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    department_id BIGINT REFERENCES hr_departments(id) ON DELETE SET NULL,
    description TEXT,
    expected_employees INTEGER NOT NULL DEFAULT 1 CHECK (expected_employees >= 0),
    no_of_employee INTEGER NOT NULL DEFAULT 0 CHECK (no_of_employee >= 0),
    company_id BIGINT,
    tag_ids JSONB,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_hr_jobs_name ON hr_jobs(name);
CREATE INDEX IF NOT EXISTS idx_hr_jobs_dept ON hr_jobs(department_id);
CREATE INDEX IF NOT EXISTS idx_hr_jobs_active ON hr_jobs(active);

CREATE TRIGGER trg_hr_jobs_updated_at
    BEFORE UPDATE ON hr_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 3. Employees (hr.employee in Odoo)
CREATE TABLE IF NOT EXISTS hr_employees (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    department_id BIGINT REFERENCES hr_departments(id) ON DELETE SET NULL,
    job_id BIGINT REFERENCES hr_jobs(id) ON DELETE SET NULL,
    job_title VARCHAR(255),
    manager_id BIGINT REFERENCES hr_employees(id) ON DELETE SET NULL,
    work_email VARCHAR(255),
    work_phone VARCHAR(50),
    work_location VARCHAR(255),
    hire_date DATE,
    gender VARCHAR(20) DEFAULT 'other',
    marital_status VARCHAR(20) DEFAULT 'single',
    identification_id VARCHAR(100),
    bank_account_no VARCHAR(100),
    company_id BIGINT,
    tag_ids JSONB,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_hr_employees_name ON hr_employees(name);
CREATE INDEX IF NOT EXISTS idx_hr_employees_partner_id ON hr_employees(partner_id);
CREATE INDEX IF NOT EXISTS idx_hr_employees_dept ON hr_employees(department_id);
CREATE INDEX IF NOT EXISTS idx_hr_employees_job ON hr_employees(job_id);
CREATE INDEX IF NOT EXISTS idx_hr_employees_manager ON hr_employees(manager_id);
CREATE INDEX IF NOT EXISTS idx_hr_employees_email ON hr_employees(work_email);
CREATE INDEX IF NOT EXISTS idx_hr_employees_active ON hr_employees(active);

CREATE TRIGGER trg_hr_employees_updated_at
    BEFORE UPDATE ON hr_employees
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add circular foreign key for Department Manager
ALTER TABLE hr_departments
    ADD CONSTRAINT fk_hr_departments_manager
    FOREIGN KEY (manager_id)
    REFERENCES hr_employees(id)
    ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_hr_departments_manager_id ON hr_departments(manager_id);

-- 4. Leave Allocations (hr.leave.allocation in Odoo)
CREATE TABLE IF NOT EXISTS hr_leave_allocations (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL DEFAULT '/',
    employee_id BIGINT NOT NULL REFERENCES hr_employees(id) ON DELETE CASCADE,
    leave_type VARCHAR(50) NOT NULL DEFAULT 'annual', -- 'annual', 'sick', 'unpaid', 'emergency', etc.
    allocated_days NUMERIC(5, 2) NOT NULL CHECK (allocated_days >= 0),
    year INTEGER NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'approved', -- 'draft', 'approved', 'cancelled'
    notes TEXT,
    company_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_hr_allocations_emp ON hr_leave_allocations(employee_id);
CREATE INDEX IF NOT EXISTS idx_hr_allocations_type_year ON hr_leave_allocations(leave_type, year);
CREATE INDEX IF NOT EXISTS idx_hr_allocations_state ON hr_leave_allocations(state);

CREATE TRIGGER trg_hr_leave_allocations_updated_at
    BEFORE UPDATE ON hr_leave_allocations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 5. Leave Requests (hr.leave in Odoo)
CREATE TABLE IF NOT EXISTS hr_leave_requests (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL DEFAULT '/',
    employee_id BIGINT NOT NULL REFERENCES hr_employees(id) ON DELETE CASCADE,
    leave_type VARCHAR(50) NOT NULL DEFAULT 'annual',
    date_from DATE NOT NULL,
    date_to DATE NOT NULL,
    days NUMERIC(5, 2) NOT NULL CHECK (days > 0),
    state VARCHAR(20) NOT NULL DEFAULT 'draft', -- 'draft', 'confirm', 'validate', 'refuse', 'cancelled'
    description TEXT,
    approver_id BIGINT REFERENCES hr_employees(id) ON DELETE SET NULL,
    refusal_reason TEXT,
    company_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT,
    CONSTRAINT chk_hr_leave_dates CHECK (date_to >= date_from)
);

CREATE INDEX IF NOT EXISTS idx_hr_leave_requests_emp ON hr_leave_requests(employee_id);
CREATE INDEX IF NOT EXISTS idx_hr_leave_requests_dates ON hr_leave_requests(date_from, date_to);
CREATE INDEX IF NOT EXISTS idx_hr_leave_requests_state ON hr_leave_requests(state);
CREATE INDEX IF NOT EXISTS idx_hr_leave_requests_type ON hr_leave_requests(leave_type);

CREATE TRIGGER trg_hr_leave_requests_updated_at
    BEFORE UPDATE ON hr_leave_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();


-- ===========================================================================
-- Source: 000011_create_core_infrastructure_schema.up.sql
-- ===========================================================================
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
    state_id BIGINT REFERENCES res_country_state(id),
    country_id BIGINT REFERENCES res_country(id),
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
    group_type VARCHAR(50),
    color INT,
    company_id BIGINT REFERENCES res_companies(id),
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


-- ===========================================================================
-- Source: 000012_create_analytic_schema.up.sql
-- ===========================================================================
-- 000012_create_analytic_schema.up.sql
-- Analytic Accounting schema (Phase 10) built against Odoo 19.0 addons/analytic:
--   plans + applicability rules (G2), JSONB distribution (G3), accounts, lines,
--   automatic distribution models (G7), and the project-plan config (G8).

-- 1. Analytic Plans (account.analytic.plan)
CREATE TABLE IF NOT EXISTS account_analytic_plan (
    id                      BIGSERIAL PRIMARY KEY,
    name                    TEXT NOT NULL,
    description             TEXT,
    parent_id               BIGINT REFERENCES account_analytic_plan(id) ON DELETE CASCADE,
    parent_path             TEXT,
    root_id                 BIGINT,
    sequence                INT DEFAULT 10,
    color                   INT,
    default_applicability   TEXT NOT NULL DEFAULT 'optional', -- optional|mandatory|unavailable
    active                  BOOLEAN DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              BIGINT,
    updated_by              BIGINT
);

CREATE INDEX IF NOT EXISTS idx_analytic_plan_parent ON account_analytic_plan(parent_id);
CREATE INDEX IF NOT EXISTS idx_analytic_plan_root ON account_analytic_plan(root_id);
CREATE INDEX IF NOT EXISTS idx_analytic_plan_active ON account_analytic_plan(active);

CREATE TRIGGER trg_account_analytic_plan_updated_at
    BEFORE UPDATE ON account_analytic_plan
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 2. Applicability Rules (account.analytic.applicability - G2)
CREATE TABLE IF NOT EXISTS account_analytic_applicability (
    id                BIGSERIAL PRIMARY KEY,
    analytic_plan_id  BIGINT NOT NULL REFERENCES account_analytic_plan(id) ON DELETE CASCADE,
    business_domain   TEXT NOT NULL DEFAULT 'general', -- general|sale_order|purchase_order|expense
    applicability     TEXT NOT NULL,                   -- optional|mandatory|unavailable
    company_id        BIGINT,                          -- NULL = applies to all companies
    sequence          INT DEFAULT 10,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by        BIGINT,
    updated_by        BIGINT
);

CREATE INDEX IF NOT EXISTS idx_analytic_applicability_plan ON account_analytic_applicability(analytic_plan_id);
CREATE INDEX IF NOT EXISTS idx_analytic_applicability_domain ON account_analytic_applicability(business_domain);
CREATE INDEX IF NOT EXISTS idx_analytic_applicability_company ON account_analytic_applicability(company_id);

CREATE TRIGGER trg_account_analytic_applicability_updated_at
    BEFORE UPDATE ON account_analytic_applicability
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 3. Analytic Accounts (account.analytic.account)
CREATE TABLE IF NOT EXISTS account_analytic_account (
    id             BIGSERIAL PRIMARY KEY,
    name           TEXT NOT NULL,
    code           TEXT,
    plan_id        BIGINT NOT NULL REFERENCES account_analytic_plan(id) ON DELETE RESTRICT,
    root_plan_id   BIGINT,
    partner_id     BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    color          INT,
    company_id     BIGINT,             -- NULL = shared across companies
    active         BOOLEAN DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by     BIGINT,
    updated_by     BIGINT
);

CREATE INDEX IF NOT EXISTS idx_analytic_account_plan ON account_analytic_account(plan_id);
CREATE INDEX IF NOT EXISTS idx_analytic_account_partner ON account_analytic_account(partner_id);
CREATE INDEX IF NOT EXISTS idx_analytic_account_active ON account_analytic_account(active);
CREATE UNIQUE INDEX IF NOT EXISTS idx_analytic_account_code ON account_analytic_account(code) WHERE code IS NOT NULL;

CREATE TRIGGER trg_account_analytic_account_updated_at
    BEFORE UPDATE ON account_analytic_account
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 4. Analytic Lines (account.analytic.line - G4, G5)
CREATE TABLE IF NOT EXISTS account_analytic_line (
    id                   BIGSERIAL PRIMARY KEY,
    name                 TEXT NOT NULL,                    -- G5: label / description
    date                 DATE NOT NULL,
    amount               DOUBLE PRECISION NOT NULL DEFAULT 0,
    unit_amount          DOUBLE PRECISION NOT NULL DEFAULT 0,
    product_uom_id       BIGINT,
    partner_id           BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    user_id              BIGINT NOT NULL,
    company_id           BIGINT NOT NULL,
    currency_code        TEXT NOT NULL DEFAULT 'USD',
    category             TEXT NOT NULL DEFAULT 'other',
    account_id           BIGINT NOT NULL REFERENCES account_analytic_account(id) ON DELETE RESTRICT, -- "Project plan" column (G4)
    analytic_distribution JSONB,                           -- G3: {account_ids: percentage}
    move_line_id         BIGINT,                           -- link to account_move_lines
    general_account_id   BIGINT,                           -- linked general ledger account
    source               TEXT NOT NULL DEFAULT 'manual',   -- manual|invoice|vendor_bill|sale_order|employee
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by           BIGINT,
    updated_by           BIGINT
);

CREATE INDEX IF NOT EXISTS idx_analytic_line_date ON account_analytic_line(date);
CREATE INDEX IF NOT EXISTS idx_analytic_line_account ON account_analytic_line(account_id);
CREATE INDEX IF NOT EXISTS idx_analytic_line_move ON account_analytic_line(move_line_id);
CREATE INDEX IF NOT EXISTS idx_analytic_line_company ON account_analytic_line(company_id);
CREATE INDEX IF NOT EXISTS idx_analytic_line_source ON account_analytic_line(source);
-- G3: GIN index to search analytic accounts referenced inside the JSONB distribution.
CREATE INDEX IF NOT EXISTS idx_analytic_line_dist_gin ON account_analytic_line
    USING gin(regexp_split_to_array(jsonb_path_query_array(
        analytic_distribution, '$.keyvalue()."key"')::text, '\D+'))
    WHERE analytic_distribution IS NOT NULL;

CREATE TRIGGER trg_account_analytic_line_updated_at
    BEFORE UPDATE ON account_analytic_line
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 5. Automatic Distribution Models (account.analytic.distribution.model - G7)
CREATE TABLE IF NOT EXISTS account_analytic_distribution_model (
    id                    BIGSERIAL PRIMARY KEY,
    sequence              INT DEFAULT 10,
    partner_id            BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    partner_category_id   BIGINT,                          -- FK added when partner categories land (M13)
    company_id            BIGINT,
    analytic_distribution JSONB NOT NULL,
    active                BOOLEAN DEFAULT TRUE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by            BIGINT,
    updated_by            BIGINT
);

CREATE INDEX IF NOT EXISTS idx_analytic_dist_model_partner ON account_analytic_distribution_model(partner_id);
CREATE INDEX IF NOT EXISTS idx_analytic_dist_model_seq ON account_analytic_distribution_model(sequence);
CREATE INDEX IF NOT EXISTS idx_analytic_dist_model_active ON account_analytic_distribution_model(active);

CREATE TRIGGER trg_account_analytic_distribution_model_updated_at
    BEFORE UPDATE ON account_analytic_distribution_model
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 6. Seed Data (analytic_data.xml equivalent - G9)
-- Default "Project" analytic plan (used by G8 project plan config).
INSERT INTO account_analytic_plan (id, name, description, sequence, color, default_applicability, active) VALUES
(1, 'Project', 'Default project plan (analytic.project_plan)', 10, 0, 'optional', true),
(2, 'Departments', 'Department cost center plan (demo)', 20, 1, 'optional', true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('account_analytic_plan_id_seq', (SELECT COALESCE(MAX(id), 1) FROM account_analytic_plan));

-- Default analytic accounts for each seeded plan.
INSERT INTO account_analytic_account (id, name, code, plan_id, root_plan_id, active) VALUES
(1, 'General', 'PRJ', 1, 1, true),
(2, 'Administration', 'ADM', 2, 2, true),
(3, 'Sales & Marketing', 'S&M', 2, 2, true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('account_analytic_account_id_seq', (SELECT COALESCE(MAX(id), 1) FROM account_analytic_account));

-- G8: analytic.project_plan -> id of the "Project" plan.
INSERT INTO ir_config_parameters (key, value, company_id) VALUES
('analytic.project_plan', '1', NULL)
ON CONFLICT (key) DO NOTHING;

-- ===========================================================================
-- Source: 000013_create_group_permissions.up.sql
-- ===========================================================================
-- Model-level ACL permissions.
CREATE TABLE IF NOT EXISTS res_group_permissions (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES res_groups(id) ON DELETE CASCADE,
    model VARCHAR(100) NOT NULL,
    can_read BOOLEAN NOT NULL DEFAULT false,
    can_create BOOLEAN NOT NULL DEFAULT false,
    can_update BOOLEAN NOT NULL DEFAULT false,
    can_delete BOOLEAN NOT NULL DEFAULT false,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_res_group_permissions_group_model UNIQUE (group_id, model)
);

CREATE INDEX IF NOT EXISTS idx_res_group_permissions_model
    ON res_group_permissions(model);
CREATE INDEX IF NOT EXISTS idx_res_group_permissions_group
    ON res_group_permissions(group_id);

CREATE TRIGGER trg_res_group_permissions_updated_at
    BEFORE UPDATE ON res_group_permissions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

INSERT INTO res_group_permissions (group_id, model, can_read)
SELECT 2, 'system.api', true
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = 2)
ON CONFLICT (group_id, model) DO NOTHING;

INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT group_id, model, can_read, can_create, can_update, can_delete
FROM (VALUES
    (1::BIGINT, 'res.company', true, false, false, false),
    (2::BIGINT, 'res.company', true, true, true, true),
    (1::BIGINT, 'product.template', true, true, true, false),
    (2::BIGINT, 'product.template', true, true, true, true),
    (1::BIGINT, 'product.product', true, true, true, false),
    (2::BIGINT, 'product.product', true, true, true, true),
    (1::BIGINT, 'product.category', true, true, true, false),
    (2::BIGINT, 'product.category', true, true, true, true),
    (1::BIGINT, 'uom.uom', true, false, false, false),
    (2::BIGINT, 'uom.uom', true, true, true, true),
    (1::BIGINT, 'product.pricelist', true, true, true, false),
    (2::BIGINT, 'product.pricelist', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;

INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT group_id, 'res.partner', can_read, can_create, can_update, can_delete
FROM (VALUES
    (1::BIGINT, true, true, true, false),
    (2::BIGINT, true, true, true, true)
) AS defaults(group_id, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;


-- ===========================================================================
-- Source: 000014_create_group_inheritance.up.sql
-- ===========================================================================
-- Group inheritance (implied groups) for effective RBAC membership.
CREATE TABLE IF NOT EXISTS res_groups_implied_rel (
    group_id BIGINT NOT NULL REFERENCES res_groups(id) ON DELETE CASCADE,
    implied_group_id BIGINT NOT NULL REFERENCES res_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, implied_group_id),
    CONSTRAINT chk_res_groups_implied_not_self CHECK (group_id <> implied_group_id)
);

CREATE INDEX IF NOT EXISTS idx_res_groups_implied_rel_implied
    ON res_groups_implied_rel(implied_group_id);


-- ===========================================================================
-- Source: 000015_create_record_rules.up.sql
-- ===========================================================================
-- Typed record-rule definitions. The domain column stores validated AST JSON,
-- never executable SQL or Python expressions.
CREATE TABLE IF NOT EXISTS res_record_rules (
    id BIGSERIAL PRIMARY KEY,
    model VARCHAR(100) NOT NULL,
    group_id BIGINT REFERENCES res_groups(id) ON DELETE CASCADE,
    domain JSONB NOT NULL,
    can_read BOOLEAN NOT NULL DEFAULT false,
    can_create BOOLEAN NOT NULL DEFAULT false,
    can_update BOOLEAN NOT NULL DEFAULT false,
    can_delete BOOLEAN NOT NULL DEFAULT false,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_res_record_rules_model ON res_record_rules(model);
CREATE INDEX IF NOT EXISTS idx_res_record_rules_group ON res_record_rules(group_id);
CREATE INDEX IF NOT EXISTS idx_res_record_rules_active ON res_record_rules(active);

CREATE TRIGGER trg_res_record_rules_updated_at
    BEFORE UPDATE ON res_record_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();


-- ===========================================================================
-- Source: 000016_seed_module_acl.up.sql
-- ===========================================================================
INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'sale.order', true, true, true, false), (2::BIGINT, 'sale.order', true, true, true, true),
    (1::BIGINT, 'purchase.order', true, true, true, false), (2::BIGINT, 'purchase.order', true, true, true, true),
    (1::BIGINT, 'account.account', true, false, false, false), (2::BIGINT, 'account.account', true, true, true, true),
    (1::BIGINT, 'account.journal', true, false, false, false), (2::BIGINT, 'account.journal', true, true, true, true),
    (1::BIGINT, 'account.tax', true, false, false, false), (2::BIGINT, 'account.tax', true, true, true, true),
    (1::BIGINT, 'account.payment_term', true, false, false, false), (2::BIGINT, 'account.payment_term', true, true, true, true),
    (1::BIGINT, 'account.move', true, true, true, false), (2::BIGINT, 'account.move', true, true, true, true),
    (1::BIGINT, 'account.report', true, false, false, false), (2::BIGINT, 'account.report', true, true, true, true),
    (1::BIGINT, 'stock.warehouse', true, false, false, false), (2::BIGINT, 'stock.warehouse', true, true, true, true),
    (1::BIGINT, 'stock.location', true, false, false, false), (2::BIGINT, 'stock.location', true, true, true, true),
    (1::BIGINT, 'stock.picking', true, true, true, false), (2::BIGINT, 'stock.picking', true, true, true, true),
    (1::BIGINT, 'stock.quant', true, true, true, false), (2::BIGINT, 'stock.quant', true, true, true, true),
    (1::BIGINT, 'stock.move', true, false, false, false), (2::BIGINT, 'stock.move', true, true, true, true),
    (1::BIGINT, 'account.payment', true, true, true, false), (2::BIGINT, 'account.payment', true, true, true, true),
    (1::BIGINT, 'hr.department', true, false, false, false), (2::BIGINT, 'hr.department', true, true, true, true),
    (1::BIGINT, 'hr.job', true, false, false, false), (2::BIGINT, 'hr.job', true, true, true, true),
    (1::BIGINT, 'hr.employee', true, false, false, false), (2::BIGINT, 'hr.employee', true, true, true, true),
    (1::BIGINT, 'hr.leave_allocation', true, true, true, false), (2::BIGINT, 'hr.leave_allocation', true, true, true, true),
    (1::BIGINT, 'hr.leave_request', true, true, true, false), (2::BIGINT, 'hr.leave_request', true, true, true, true),
    (1::BIGINT, 'crm.lead', true, true, true, false), (2::BIGINT, 'crm.lead', true, true, true, true),
    (1::BIGINT, 'crm.stage', true, false, false, false), (2::BIGINT, 'crm.stage', true, true, true, true),
    (1::BIGINT, 'crm.lost_reason', true, false, false, false), (2::BIGINT, 'crm.lost_reason', true, true, true, true),
    (1::BIGINT, 'crm.tag', true, true, true, false), (2::BIGINT, 'crm.tag', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;


-- ===========================================================================
-- Source: 000017_create_bank_statements_schema.up.sql
-- ===========================================================================
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

-- ===========================================================================
-- Source: 000018_create_stock_account_schema.up.sql
-- ===========================================================================
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


-- ===========================================================================
-- Source: 000019_create_project_schema.up.sql
-- ===========================================================================
-- Phase 17: Project management schema.
-- Project stages and task stages intentionally remain separate, matching Odoo 19.

CREATE TABLE IF NOT EXISTS project_project_stages (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    sequence INT NOT NULL DEFAULT 10,
    fold BOOLEAN NOT NULL DEFAULT false,
    color INT NOT NULL DEFAULT 0,
    company_id BIGINT REFERENCES res_companies(id) ON DELETE CASCADE,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    CONSTRAINT uq_project_project_stages_name_company UNIQUE (name, company_id)
);
CREATE INDEX IF NOT EXISTS idx_project_project_stages_company ON project_project_stages(company_id);
CREATE INDEX IF NOT EXISTS idx_project_project_stages_sequence ON project_project_stages(sequence, id);

CREATE TABLE IF NOT EXISTS project_projects (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    manager_id BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    stage_id BIGINT REFERENCES project_project_stages(id) ON DELETE RESTRICT,
    date_start DATE,
    date_end DATE,
    active BOOLEAN NOT NULL DEFAULT true,
    allow_milestones BOOLEAN NOT NULL DEFAULT false,
    allow_subtasks BOOLEAN NOT NULL DEFAULT true,
    allow_dependencies BOOLEAN NOT NULL DEFAULT false,
    analytic_account_id BIGINT REFERENCES account_analytic_account(id) ON DELETE SET NULL,
    tag_ids JSONB,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    CONSTRAINT chk_project_projects_dates CHECK (date_end IS NULL OR date_start IS NULL OR date_end >= date_start)
);
CREATE INDEX IF NOT EXISTS idx_project_projects_company ON project_projects(company_id);
CREATE INDEX IF NOT EXISTS idx_project_projects_manager ON project_projects(manager_id);
CREATE INDEX IF NOT EXISTS idx_project_projects_stage ON project_projects(stage_id);
CREATE INDEX IF NOT EXISTS idx_project_projects_active ON project_projects(active);

CREATE TABLE IF NOT EXISTS project_task_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    sequence INT NOT NULL DEFAULT 10,
    fold BOOLEAN NOT NULL DEFAULT false,
    color INT NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT true,
    company_id BIGINT REFERENCES res_companies(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    CONSTRAINT uq_project_task_types_name_company UNIQUE (name, company_id)
);
CREATE INDEX IF NOT EXISTS idx_project_task_types_company ON project_task_types(company_id);
CREATE INDEX IF NOT EXISTS idx_project_task_types_sequence ON project_task_types(sequence, id);

CREATE TABLE IF NOT EXISTS project_task_type_projects (
    task_type_id BIGINT NOT NULL REFERENCES project_task_types(id) ON DELETE CASCADE,
    project_id BIGINT NOT NULL REFERENCES project_projects(id) ON DELETE CASCADE,
    PRIMARY KEY (task_type_id, project_id)
);
CREATE INDEX IF NOT EXISTS idx_project_task_type_projects_project ON project_task_type_projects(project_id);

CREATE TABLE IF NOT EXISTS project_tasks (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    project_id BIGINT NOT NULL REFERENCES project_projects(id) ON DELETE CASCADE,
    stage_id BIGINT NOT NULL REFERENCES project_task_types(id) ON DELETE RESTRICT,
    parent_id BIGINT REFERENCES project_tasks(id) ON DELETE CASCADE,
    priority VARCHAR(1) NOT NULL DEFAULT '1',
    date_deadline TIMESTAMPTZ,
    date_assign TIMESTAMPTZ,
    state VARCHAR(32) NOT NULL DEFAULT 'in_progress',
    description TEXT,
    milestone_id BIGINT,
    sequence INT NOT NULL DEFAULT 10,
    allocated_hours DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (allocated_hours >= 0),
    tag_ids JSONB,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    CONSTRAINT chk_project_tasks_priority CHECK (priority IN ('0', '1', '2', '3')),
    CONSTRAINT chk_project_tasks_state CHECK (state IN ('in_progress', 'changes_requested', 'approved', 'waiting', 'done', 'cancelled')),
    CONSTRAINT chk_project_tasks_not_self_parent CHECK (parent_id IS NULL OR parent_id <> id)
);
CREATE INDEX IF NOT EXISTS idx_project_tasks_project ON project_tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_project_tasks_stage ON project_tasks(stage_id);
CREATE INDEX IF NOT EXISTS idx_project_tasks_parent ON project_tasks(parent_id);
CREATE INDEX IF NOT EXISTS idx_project_tasks_company ON project_tasks(company_id);
CREATE INDEX IF NOT EXISTS idx_project_tasks_state ON project_tasks(state);
CREATE INDEX IF NOT EXISTS idx_project_tasks_deadline ON project_tasks(date_deadline);

CREATE TABLE IF NOT EXISTS project_task_assignees (
    task_id BIGINT NOT NULL REFERENCES project_tasks(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES res_users(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_project_task_assignees_user ON project_task_assignees(user_id);

CREATE TABLE IF NOT EXISTS project_task_dependencies (
    task_id BIGINT NOT NULL REFERENCES project_tasks(id) ON DELETE CASCADE,
    depends_on_task_id BIGINT NOT NULL REFERENCES project_tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, depends_on_task_id),
    CONSTRAINT chk_project_task_dependencies_not_self CHECK (task_id <> depends_on_task_id)
);
CREATE INDEX IF NOT EXISTS idx_project_task_dependencies_dependency ON project_task_dependencies(depends_on_task_id);

CREATE TABLE IF NOT EXISTS project_milestones (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    project_id BIGINT NOT NULL REFERENCES project_projects(id) ON DELETE CASCADE,
    date_deadline DATE,
    is_reached BOOLEAN NOT NULL DEFAULT false,
    reached_date DATE,
    sequence INT NOT NULL DEFAULT 10,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_project_milestones_project ON project_milestones(project_id);
CREATE INDEX IF NOT EXISTS idx_project_milestones_deadline ON project_milestones(date_deadline);
ALTER TABLE project_tasks ADD CONSTRAINT fk_project_tasks_milestone FOREIGN KEY (milestone_id) REFERENCES project_milestones(id) ON DELETE SET NULL;

CREATE TRIGGER trg_project_project_stages_updated_at BEFORE UPDATE ON project_project_stages FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_project_projects_updated_at BEFORE UPDATE ON project_projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_project_task_types_updated_at BEFORE UPDATE ON project_task_types FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_project_tasks_updated_at BEFORE UPDATE ON project_tasks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_project_milestones_updated_at BEFORE UPDATE ON project_milestones FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'project.project', true, false, false, false), (2::BIGINT, 'project.project', true, true, true, true),
    (1::BIGINT, 'project.task', true, true, true, false), (2::BIGINT, 'project.task', true, true, true, true),
    (1::BIGINT, 'project.project.stage', true, false, false, false), (2::BIGINT, 'project.project.stage', true, true, true, true),
    (1::BIGINT, 'project.task.type', true, false, false, false), (2::BIGINT, 'project.task.type', true, true, true, true),
    (1::BIGINT, 'project.milestone', true, true, true, false), (2::BIGINT, 'project.milestone', true, true, true, true),
    (1::BIGINT, 'project.tags', true, true, true, false), (2::BIGINT, 'project.tags', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;


-- ===========================================================================
-- Source: 000020_create_activity_schema.up.sql
-- ===========================================================================
-- Phase 15: activities, notifications, messages, and email delivery.

CREATE TABLE IF NOT EXISTS mail_activity_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    summary VARCHAR(255),
    res_model VARCHAR(128),
    category VARCHAR(32) NOT NULL DEFAULT 'default',
    delay_count INT NOT NULL DEFAULT 0 CHECK (delay_count >= 0),
    delay_unit VARCHAR(16) NOT NULL DEFAULT 'days',
    icon VARCHAR(128),
    sequence INT NOT NULL DEFAULT 10,
    default_note TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    system_type BOOLEAN NOT NULL DEFAULT false,
    company_id BIGINT REFERENCES res_companies(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    CONSTRAINT chk_mail_activity_types_delay_unit CHECK (delay_unit IN ('days', 'weeks', 'months')),
    CONSTRAINT uq_mail_activity_types_name_company UNIQUE (name, company_id)
);
CREATE INDEX IF NOT EXISTS idx_mail_activity_types_company ON mail_activity_types(company_id);
CREATE INDEX IF NOT EXISTS idx_mail_activity_types_active ON mail_activity_types(active);

CREATE TABLE IF NOT EXISTS mail_activities (
    id BIGSERIAL PRIMARY KEY,
    activity_type_id BIGINT NOT NULL REFERENCES mail_activity_types(id) ON DELETE RESTRICT,
    summary VARCHAR(255) NOT NULL,
    note TEXT,
    date_deadline DATE NOT NULL,
    assigned_user_id BIGINT NOT NULL REFERENCES res_users(id) ON DELETE RESTRICT,
    res_model VARCHAR(128),
    res_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    date_done TIMESTAMPTZ,
    feedback TEXT,
    tag_ids JSONB,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    CONSTRAINT chk_mail_activities_reference_pair CHECK ((res_model IS NULL) = (res_id IS NULL)),
    CONSTRAINT chk_mail_activities_done_data CHECK (active OR date_done IS NOT NULL),
    CONSTRAINT chk_mail_activities_done_date CHECK (date_done IS NULL OR date_done >= created_at)
);
CREATE INDEX IF NOT EXISTS idx_mail_activities_user_deadline ON mail_activities(company_id, assigned_user_id, active, date_deadline, id);
CREATE INDEX IF NOT EXISTS idx_mail_activities_resource ON mail_activities(company_id, res_model, res_id, active, date_deadline, id);
CREATE INDEX IF NOT EXISTS idx_mail_activities_type_active ON mail_activities(activity_type_id, active);
CREATE INDEX IF NOT EXISTS idx_mail_activities_deadline ON mail_activities(company_id, active, date_deadline);

CREATE TABLE IF NOT EXISTS mail_messages (
    id BIGSERIAL PRIMARY KEY,
    subject VARCHAR(255),
    body TEXT NOT NULL,
    message_type VARCHAR(32) NOT NULL DEFAULT 'notification',
    res_model VARCHAR(128),
    res_id BIGINT,
    author_id BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    activity_id BIGINT REFERENCES mail_activities(id) ON DELETE SET NULL,
    tag_ids JSONB,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_mail_messages_reference_pair CHECK ((res_model IS NULL) = (res_id IS NULL))
);
CREATE INDEX IF NOT EXISTS idx_mail_messages_resource ON mail_messages(company_id, res_model, res_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_mail_messages_activity ON mail_messages(activity_id);

CREATE TABLE IF NOT EXISTS mail_notifications (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT REFERENCES mail_messages(id) ON DELETE CASCADE,
    activity_id BIGINT REFERENCES mail_activities(id) ON DELETE CASCADE,
    recipient_user_id BIGINT NOT NULL REFERENCES res_users(id) ON DELETE CASCADE,
    notification_type VARCHAR(16) NOT NULL DEFAULT 'inbox',
    status VARCHAR(16) NOT NULL DEFAULT 'unread',
    read_at TIMESTAMPTZ,
    email VARCHAR(320),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tag_ids JSONB,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE RESTRICT,
    CONSTRAINT chk_mail_notifications_type CHECK (notification_type IN ('inbox', 'email')),
    CONSTRAINT chk_mail_notifications_status CHECK (status IN ('unread', 'read', 'queued', 'sent', 'failed')),
    CONSTRAINT chk_mail_notifications_read_at CHECK ((status = 'read') = (read_at IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS idx_mail_notifications_recipient ON mail_notifications(company_id, recipient_user_id, status, created_at, id);
CREATE INDEX IF NOT EXISTS idx_mail_notifications_activity ON mail_notifications(activity_id);

CREATE TABLE IF NOT EXISTS mail_email_queue (
    id BIGSERIAL PRIMARY KEY,
    notification_id BIGINT REFERENCES mail_notifications(id) ON DELETE CASCADE,
    recipient_email VARCHAR(320) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'queued',
    attempts INT NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error TEXT,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tag_ids JSONB,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE RESTRICT,
    CONSTRAINT chk_mail_email_queue_status CHECK (status IN ('queued', 'processing', 'sent', 'failed', 'dead_letter'))
);
CREATE INDEX IF NOT EXISTS idx_mail_email_queue_delivery ON mail_email_queue(status, next_attempt_at, id);
CREATE INDEX IF NOT EXISTS idx_mail_email_queue_company ON mail_email_queue(company_id, status, next_attempt_at);

CREATE TRIGGER trg_mail_activity_types_updated_at BEFORE UPDATE ON mail_activity_types FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_mail_activities_updated_at BEFORE UPDATE ON mail_activities FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_mail_notifications_updated_at BEFORE UPDATE ON mail_notifications FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_mail_email_queue_updated_at BEFORE UPDATE ON mail_email_queue FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

INSERT INTO mail_activity_types (name, summary, category, icon, delay_count, delay_unit, system_type)
VALUES
    ('Email', 'Send an email', 'default', 'fa-envelope', 0, 'days', true),
    ('Call', 'Make a phone call', 'call', 'fa-phone', 0, 'days', true),
    ('Meeting', 'Schedule a meeting', 'meeting', 'fa-calendar', 0, 'days', true),
    ('To-Do', 'Complete a task', 'default', 'fa-check', 0, 'days', true),
    ('Document', 'Send or review a document', 'upload_file', 'fa-upload', 0, 'days', true),
    ('Exception', 'Handle an exception', 'exception', 'fa-warning', 0, 'days', true)
ON CONFLICT (name, company_id) DO NOTHING;

INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'mail.activity', true, true, true, false),
    (2::BIGINT, 'mail.activity', true, true, true, true),
    (1::BIGINT, 'mail.activity.type', true, false, false, false),
    (2::BIGINT, 'mail.activity.type', true, true, true, true),
    (1::BIGINT, 'mail.message', true, true, false, false),
    (2::BIGINT, 'mail.message', true, true, true, true),
    (1::BIGINT, 'mail.notification', true, true, true, false),
    (2::BIGINT, 'mail.notification', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;


-- ===========================================================================
-- Source: 000021_create_reorder_landed_cost_schema.up.sql
-- ===========================================================================
-- 000021_create_reorder_landed_cost_schema.up.sql
-- Phase 13: Auto Reorder (stock.orderpoint) + Landed Costs (stock.landed.cost)
-- Reference: Odoo 19.0 addons/stock/models/stock_orderpoint.py,
--            addons/stock_landed_costs/models/stock_landed_cost.py, purchase.py, account_move.py

-- 1. Reorder rules (stock.orderpoint)
CREATE TABLE IF NOT EXISTS stock_orderpoints (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,               -- "ROP/2026/00001"
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE CASCADE,
    warehouse_id BIGINT REFERENCES stock_warehouses(id) ON DELETE CASCADE,
    location_id BIGINT NOT NULL REFERENCES stock_locations(id) ON DELETE CASCADE,
    vendor_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,   -- preferred supplier (Odoo res.partner)
    min_qty NUMERIC(15, 4) NOT NULL DEFAULT 0.0,                  -- qty_forecast lower bound
    max_qty NUMERIC(15, 4) NOT NULL DEFAULT 0.0,                  -- qty_forecast upper bound
    qty_multiple NUMERIC(15, 4) NOT NULL DEFAULT 1.0,             -- qty ordering multiple
    lead_days INTEGER NOT NULL DEFAULT 0,
    source VARCHAR(16) NOT NULL DEFAULT 'buy',                    -- 'buy' (procurement route)
    trigger VARCHAR(16) NOT NULL DEFAULT 'manual',                -- 'auto' | 'manual'
    snoozed_until TIMESTAMPTZ,
    qty_on_hand NUMERIC(15, 4) NOT NULL DEFAULT 0.0,              -- computed
    qty_forecast NUMERIC(15, 4) NOT NULL DEFAULT 0.0,             -- computed (on_hand - out + in over lead horizon)
    qty_to_order NUMERIC(15, 4) NOT NULL DEFAULT 0.0,             -- computed (max(max_forecast - forecast, 0), rounded to multiple)
    qty_to_order_manual NUMERIC(15, 4) NOT NULL DEFAULT 0.0,      -- manual override (Prefer to order)
    deadline_date DATE,
    active BOOLEAN NOT NULL DEFAULT true,
    company_id BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Odoo stock_orderpoint._name: unique(product_id, location_id, company_id)
CREATE UNIQUE INDEX IF NOT EXISTS idx_stock_orderpoints_product_location_company
    ON stock_orderpoints(product_id, location_id, company_id) WHERE active = true;

CREATE INDEX IF NOT EXISTS idx_stock_orderpoints_warehouse ON stock_orderpoints(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_stock_orderpoints_trigger ON stock_orderpoints(trigger) WHERE active = true;

CREATE TRIGGER trg_stock_orderpoints_updated_at
    BEFORE UPDATE ON stock_orderpoints
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 2. Landed costs header (stock.landed.cost)
CREATE TABLE IF NOT EXISTS stock_landed_costs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,               -- "LC/2026/00001"
    date DATE NOT NULL,
    state VARCHAR(16) NOT NULL DEFAULT 'draft', -- 'draft' | 'done' | 'cancel'
    picking_ids BIGINT[] NOT NULL DEFAULT '{}', -- M2M stock.picking
    amount_total NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    description TEXT,
    account_move_id BIGINT REFERENCES account_moves(id) ON DELETE SET NULL, -- the created valuation entry
    journal_id BIGINT REFERENCES account_journals(id) ON DELETE RESTRICT,   -- STJ / expense journal
    vendor_bill_id BIGINT REFERENCES account_moves(id) ON DELETE SET NULL,  -- source vendor bill (stock_landed_costs.vendor_bill_id)
    company_id BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_stock_landed_costs_state ON stock_landed_costs(state);
CREATE INDEX IF NOT EXISTS idx_stock_landed_costs_pickings ON stock_landed_costs USING GIN (picking_ids);

CREATE TRIGGER trg_stock_landed_costs_updated_at
    BEFORE UPDATE ON stock_landed_costs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 3. Landed cost lines (stock.landed.cost.lines) — expense / cost lines
CREATE TABLE IF NOT EXISTS stock_landed_cost_lines (
    id BIGSERIAL PRIMARY KEY,
    landed_cost_id BIGINT NOT NULL REFERENCES stock_landed_costs(id) ON DELETE CASCADE,
    name VARCHAR(512) NOT NULL,
    product_id BIGINT REFERENCES product_templates(id) ON DELETE SET NULL, -- cost product (non-inventory)
    account_id BIGINT REFERENCES account_accounts(id) ON DELETE RESTRICT,  -- expense account (account_expense_line)
    price_unit NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    split_method VARCHAR(24) NOT NULL DEFAULT 'equal', -- 'equal' | 'by_quantity' | 'by_current_cost_price' | 'by_weight' | 'by_volume'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_landed_cost_lines_lc ON stock_landed_cost_lines(landed_cost_id);

CREATE TRIGGER trg_stock_landed_cost_lines_updated_at
    BEFORE UPDATE ON stock_landed_cost_lines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 4. Valuation adjustment lines (stock.valuation.adjustment.lines) — computed allocation per move
CREATE TABLE IF NOT EXISTS stock_valuation_adjustment_lines (
    id BIGSERIAL PRIMARY KEY,
    landed_cost_id BIGINT NOT NULL REFERENCES stock_landed_costs(id) ON DELETE CASCADE,
    cost_line_id BIGINT REFERENCES stock_landed_cost_lines(id) ON DELETE CASCADE,
    move_id BIGINT REFERENCES stock_moves(id) ON DELETE CASCADE,
    product_id BIGINT REFERENCES product_templates(id) ON DELETE CASCADE,
    quantity NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    weight NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    volume NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    former_cost NUMERIC(15, 4) NOT NULL DEFAULT 0.0,    -- additional_landed_cost
    additional_cost NUMERIC(15, 4) NOT NULL DEFAULT 0.0, -- per unit allocation for this move
    final_cost NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    move_remaining_qty NUMERIC(15, 4) NOT NULL DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_valuation_adj_landed_cost ON stock_valuation_adjustment_lines(landed_cost_id);
CREATE INDEX IF NOT EXISTS idx_stock_valuation_adj_move ON stock_valuation_adjustment_lines(move_id);

CREATE TRIGGER trg_stock_valuation_adjustment_lines_updated_at
    BEFORE UPDATE ON stock_valuation_adjustment_lines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 5. Product flag for landed-cost capability (stock_landed_costs.purchase.py: landed_cost_ok)
ALTER TABLE product_templates
    ADD COLUMN IF NOT EXISTS landed_cost_ok BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS split_method_landed_cost VARCHAR(24) NOT NULL DEFAULT 'equal';

-- 6. Company default journal for landed-cost entries (stock_landed_costs res.company: lc_journal_id)
ALTER TABLE res_companies
    ADD COLUMN IF NOT EXISTS lc_journal_id BIGINT REFERENCES account_journals(id) ON DELETE SET NULL;

-- 7. Orderpoint sequence
CREATE SEQUENCE IF NOT EXISTS stock_orderpoint_sequence START 1;

-- 8. Landed cost sequence
CREATE SEQUENCE IF NOT EXISTS stock_landed_cost_sequence START 1;

-- 9. ACL: reorder rules & landed costs (mirror stock.* permissions in 000016)
INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'stock.orderpoint', true, false, false, false), (2::BIGINT, 'stock.orderpoint', true, true, true, true),
    (1::BIGINT, 'stock.landed.cost', true, false, false, false), (2::BIGINT, 'stock.landed.cost', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;

-- ===========================================================================
-- Source: 000022_sale_purchase_stock_integration.up.sql
-- ===========================================================================
-- 000022_sale_purchase_stock_integration.up.sql

-- 1. Create Stock Procurement Groups
CREATE TABLE IF NOT EXISTS stock_procurement_groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    company_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_stock_procurement_groups_updated_at
    BEFORE UPDATE ON stock_procurement_groups
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 2. Add columns to sale_orders
ALTER TABLE sale_orders
ADD COLUMN IF NOT EXISTS delivery_status VARCHAR(20) NOT NULL DEFAULT 'nothing',
ADD COLUMN IF NOT EXISTS procurement_group_id BIGINT REFERENCES stock_procurement_groups(id) ON DELETE SET NULL;

-- 3. Add columns to sale_order_lines
ALTER TABLE sale_order_lines
ADD COLUMN IF NOT EXISTS route_id BIGINT;

-- 4. Add columns to purchase_orders
ALTER TABLE purchase_orders
ADD COLUMN IF NOT EXISTS receipt_status VARCHAR(20) NOT NULL DEFAULT 'nothing',
ADD COLUMN IF NOT EXISTS procurement_group_id BIGINT REFERENCES stock_procurement_groups(id) ON DELETE SET NULL;

-- 5. Add columns to stock_pickings
ALTER TABLE stock_pickings
ADD COLUMN IF NOT EXISTS procurement_group_id BIGINT REFERENCES stock_procurement_groups(id) ON DELETE SET NULL;

-- 6. Add columns to stock_moves
ALTER TABLE stock_moves
ADD COLUMN IF NOT EXISTS procurement_group_id BIGINT REFERENCES stock_procurement_groups(id) ON DELETE SET NULL;


-- ===========================================================================
-- Source: 000023_create_mrp_schema.up.sql
-- ===========================================================================
-- Phase 16: Manufacturing (MRP) Schema

-- 1. Workcenters
CREATE TABLE mrp_workcenters (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(32),
    active BOOLEAN DEFAULT TRUE,
    sequence INTEGER DEFAULT 10,
    company_id BIGINT NOT NULL,

    -- Capacity and Timing
    time_start DOUBLE PRECISION DEFAULT 0,  -- Setup time (minutes)
    time_stop DOUBLE PRECISION DEFAULT 0,   -- Cleanup time (minutes)
    time_efficiency DOUBLE PRECISION DEFAULT 100.0,
    capacity DOUBLE PRECISION DEFAULT 1.0,

    -- Costing
    cost_per_hour DOUBLE PRECISION DEFAULT 0,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT,
    updated_by BIGINT
);

-- 2. Bill of Materials (BoM)
CREATE TABLE mrp_boms (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64),
    product_id BIGINT NOT NULL REFERENCES product_variants(id),
    product_qty DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    uom_id BIGINT NOT NULL REFERENCES uom_uoms(id),
    type VARCHAR(32) NOT NULL DEFAULT 'normal', -- normal, phantom
    ready_to_produce VARCHAR(32) DEFAULT 'all_available', -- all_available, asap
    consumption VARCHAR(32) DEFAULT 'flexible', -- flexible, warning, strict
    active BOOLEAN DEFAULT TRUE,
    company_id BIGINT NOT NULL,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT,
    updated_by BIGINT
);

-- 3. Routing Operations
CREATE TABLE mrp_routing_operations (
    id BIGSERIAL PRIMARY KEY,
    bom_id BIGINT NOT NULL REFERENCES mrp_boms(id) ON DELETE CASCADE,
    workcenter_id BIGINT NOT NULL REFERENCES mrp_workcenters(id),
    name VARCHAR(255) NOT NULL,
    sequence INTEGER DEFAULT 10,
    time_mode VARCHAR(32) DEFAULT 'manual', -- manual, computed
    time_cycle_manual DOUBLE PRECISION DEFAULT 0 -- expected duration in minutes
);

-- 4. BoM Lines (Components)
CREATE TABLE mrp_bom_lines (
    id BIGSERIAL PRIMARY KEY,
    bom_id BIGINT NOT NULL REFERENCES mrp_boms(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES product_variants(id),
    quantity DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    uom_id BIGINT NOT NULL REFERENCES uom_uoms(id),
    operation_id BIGINT REFERENCES mrp_routing_operations(id) ON DELETE SET NULL,
    sequence INTEGER DEFAULT 10
);

-- Indices for performance
CREATE INDEX idx_mrp_boms_product ON mrp_boms(product_id);
CREATE INDEX idx_mrp_bom_lines_bom ON mrp_bom_lines(bom_id);
CREATE INDEX idx_mrp_routing_ops_bom ON mrp_routing_operations(bom_id);

-- 5. Production Orders (MO)
CREATE TABLE mrp_productions (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    priority INTEGER DEFAULT 0,
    backorder_sequence INTEGER DEFAULT 0,
    origin VARCHAR(255),

    product_id BIGINT NOT NULL REFERENCES product_variants(id),
    product_qty DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    uom_id BIGINT NOT NULL REFERENCES uom_uoms(id),
    qty_producing DOUBLE PRECISION DEFAULT 0,
    qty_produced DOUBLE PRECISION DEFAULT 0,

    bom_id BIGINT NOT NULL REFERENCES mrp_boms(id),
    picking_type_id BIGINT NOT NULL, -- references stock_picking_types
    location_src_id BIGINT NOT NULL, -- references stock_locations
    location_dest_id BIGINT NOT NULL, -- references stock_locations

    date_deadline TIMESTAMP WITH TIME ZONE,
    date_start TIMESTAMP WITH TIME ZONE NOT NULL,
    date_finished TIMESTAMP WITH TIME ZONE,

    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    reservation_state VARCHAR(32) DEFAULT 'confirmed',

    company_id BIGINT NOT NULL,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX idx_mrp_prod_product ON mrp_productions(product_id);
CREATE INDEX idx_mrp_prod_state ON mrp_productions(state);
CREATE INDEX idx_mrp_prod_name ON mrp_productions(name);

-- 6. Workorders
CREATE TABLE mrp_workorders (
    id BIGSERIAL PRIMARY KEY,
    production_id BIGINT NOT NULL REFERENCES mrp_productions(id) ON DELETE CASCADE,
    workcenter_id BIGINT NOT NULL REFERENCES mrp_workcenters(id),
    operation_id BIGINT NOT NULL REFERENCES mrp_routing_operations(id),
    name VARCHAR(255) NOT NULL,
    sequence INTEGER DEFAULT 10,
    state VARCHAR(32) NOT NULL DEFAULT 'ready',

    duration_expected DOUBLE PRECISION DEFAULT 0,
    duration DOUBLE PRECISION DEFAULT 0,

    date_start TIMESTAMP WITH TIME ZONE,
    date_finished TIMESTAMP WITH TIME ZONE,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX idx_mrp_wo_production ON mrp_workorders(production_id);
CREATE INDEX idx_mrp_wo_state ON mrp_workorders(state);

-- 7. Unbuild Orders
CREATE TABLE mrp_unbuilds (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    product_id BIGINT NOT NULL REFERENCES product_variants(id),
    bom_id BIGINT NOT NULL REFERENCES mrp_boms(id),
    mo_id BIGINT REFERENCES mrp_productions(id) ON DELETE SET NULL,
    quantity DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    uom_id BIGINT NOT NULL REFERENCES uom_uoms(id),
    location_id BIGINT NOT NULL, -- references stock_locations
    dest_location_id BIGINT NOT NULL,

    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    company_id BIGINT NOT NULL,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX idx_mrp_unbuild_product ON mrp_unbuilds(product_id);
CREATE INDEX idx_mrp_unbuild_mo ON mrp_unbuilds(mo_id);


-- ===========================================================================
-- Source: 000024_add_mrp_to_stock_move.up.sql
-- ===========================================================================
ALTER TABLE stock_moves ADD COLUMN production_id BIGINT REFERENCES mrp_productions(id) ON DELETE CASCADE;
ALTER TABLE stock_moves ADD COLUMN production_finished_id BIGINT REFERENCES mrp_productions(id) ON DELETE CASCADE;

CREATE INDEX idx_stock_move_production ON stock_moves(production_id);
CREATE INDEX idx_stock_move_production_fin ON stock_moves(production_finished_id);


-- ===========================================================================
-- Source: 000025_create_attendance_schema.up.sql
-- ===========================================================================
-- HR Attendance Schema

CREATE TABLE IF NOT EXISTS hr_attendance (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hr_employees(id) ON DELETE CASCADE,
    check_in TIMESTAMP WITH TIME ZONE NOT NULL,
    check_out TIMESTAMP WITH TIME ZONE,
    worked_hours DOUBLE PRECISION DEFAULT 0,
    expected_hours DOUBLE PRECISION DEFAULT 0,
    overtime_hours DOUBLE PRECISION DEFAULT 0,
    overtime_status VARCHAR(20) DEFAULT 'to_approve', -- to_approve, approved, refused

    -- In tracking
    in_latitude DOUBLE PRECISION,
    in_longitude DOUBLE PRECISION,
    in_ip_address VARCHAR(45),
    in_browser TEXT,
    in_mode VARCHAR(20), -- kiosk, systray, manual, technical

    -- Out tracking
    out_latitude DOUBLE PRECISION,
    out_longitude DOUBLE PRECISION,
    out_ip_address VARCHAR(45),
    out_browser TEXT,
    out_mode VARCHAR(20), -- kiosk, systray, manual, technical, auto_check_out
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_overtime_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    base_off VARCHAR(20) NOT NULL, -- quantity, timing
    timing_type VARCHAR(20),      -- work_days, non_work_days, leave, schedule
    timing_start DOUBLE PRECISION,
    multiplier DOUBLE PRECISION DEFAULT 1.0,
    active BOOLEAN DEFAULT TRUE,
    company_id BIGINT REFERENCES res_companies(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_overtime_lines (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hr_employees(id) ON DELETE CASCADE,
    attendance_id BIGINT REFERENCES hr_attendance(id) ON DELETE SET NULL,
    date DATE NOT NULL,
    duration DOUBLE PRECISION DEFAULT 0,
    manual_duration DOUBLE PRECISION DEFAULT 0,
    status VARCHAR(20) DEFAULT 'to_approve', -- to_approve, approved, refused
    time_start TIMESTAMP WITH TIME ZONE,
    time_stop TIMESTAMP WITH TIME ZONE,
    rule_ids BIGINT[],
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add Attendance Configuration to Companies
ALTER TABLE res_companies ADD COLUMN IF NOT EXISTS attendance_kiosk_mode VARCHAR(20) DEFAULT 'barcode_pin';
ALTER TABLE res_companies ADD COLUMN IF NOT EXISTS attendance_kiosk_delay INTEGER DEFAULT 10;
ALTER TABLE res_companies ADD COLUMN IF NOT EXISTS overtime_company_threshold INTEGER DEFAULT 0;
ALTER TABLE res_companies ADD COLUMN IF NOT EXISTS auto_check_out_tolerance DOUBLE PRECISION DEFAULT 0;

-- Add Overtime Threshold to Employees
ALTER TABLE hr_employees ADD COLUMN IF NOT EXISTS overtime_employee_threshold INTEGER DEFAULT 0;

CREATE INDEX idx_hr_attendance_employee ON hr_attendance(employee_id);
CREATE INDEX idx_hr_attendance_check_in ON hr_attendance(check_in);
CREATE INDEX idx_hr_overtime_lines_employee ON hr_overtime_lines(employee_id);
CREATE INDEX idx_hr_overtime_lines_date ON hr_overtime_lines(date);


-- ===========================================================================
-- Source: 000026_create_maintenance_fleet_schema.up.sql
-- ===========================================================================
-- Phase 24: Maintenance & Fleet Schema

-- ═══════════════════════════════════════════════════════════════════
-- 1. Maintenance Schema
-- ═══════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS maintenance_equipment_categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    color INTEGER,
    active BOOLEAN DEFAULT true,
    company_id BIGINT REFERENCES res_companies(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS maintenance_stages (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    sequence INTEGER DEFAULT 0,
    fold BOOLEAN DEFAULT false,
    done BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS maintenance_equipment (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category_id BIGINT REFERENCES maintenance_equipment_categories(id),
    technician_user_id BIGINT REFERENCES res_users(id),
    owner_user_id BIGINT REFERENCES res_users(id),
    employee_id BIGINT REFERENCES hr_employees(id),
    department_id BIGINT REFERENCES hr_departments(id),
    assign_to VARCHAR(20) DEFAULT 'employee', -- 'employee', 'department', 'other'
    location_id BIGINT REFERENCES stock_locations(id),
    serial_no VARCHAR(255),
    model VARCHAR(255),
    warranty_date DATE,
    effective_date DATE,
    next_action_date DATE,
    period INTEGER DEFAULT 0, -- Maintenance frequency in days
    active BOOLEAN DEFAULT true,
    tag_ids JSONB,
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS maintenance_requests (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    equipment_id BIGINT REFERENCES maintenance_equipment(id),
    request_date DATE NOT NULL DEFAULT CURRENT_DATE,
    close_date DATE,
    schedule_date TIMESTAMPTZ,
    maintenance_type VARCHAR(20) DEFAULT 'corrective', -- 'corrective', 'preventive'
    priority VARCHAR(1) DEFAULT '0', -- '0', '1', '2', '3'
    stage_id BIGINT REFERENCES maintenance_stages(id),
    technician_user_id BIGINT REFERENCES res_users(id),
    owner_user_id BIGINT REFERENCES res_users(id),
    employee_id BIGINT REFERENCES hr_employees(id),
    department_id BIGINT REFERENCES hr_departments(id),
    duration DOUBLE PRECISION DEFAULT 0.0,
    description TEXT,
    tag_ids JSONB,
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ═══════════════════════════════════════════════════════════════════
-- 2. Fleet Schema
-- ═══════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS fleet_vehicle_brands (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    image_128 BYTEA,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fleet_vehicle_model_categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fleet_vehicle_models (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    brand_id BIGINT NOT NULL REFERENCES fleet_vehicle_brands(id),
    category_id BIGINT REFERENCES fleet_vehicle_model_categories(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fleet_vehicles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255),
    license_plate VARCHAR(32) NOT NULL UNIQUE,
    model_id BIGINT NOT NULL REFERENCES fleet_vehicle_models(id),
    driver_id BIGINT REFERENCES res_partners(id),
    future_driver_id BIGINT REFERENCES res_partners(id),
    vin_sn VARCHAR(64),
    acquisition_date DATE,
    first_contract_date DATE,
    odometer DOUBLE PRECISION DEFAULT 0.0,
    odometer_unit VARCHAR(10) DEFAULT 'kilometers',
    fuel_type VARCHAR(20), -- 'gasoline', 'diesel', 'lpg', 'electric', 'hybrid'
    horsepower INTEGER,
    horsepower_tax DOUBLE PRECISION,
    seats INTEGER,
    doors INTEGER,
    color VARCHAR(32),
    location VARCHAR(128),
    state VARCHAR(20) DEFAULT 'active',
    active BOOLEAN DEFAULT true,
    tag_ids JSONB,
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fleet_vehicle_assignation_logs (
    id BIGSERIAL PRIMARY KEY,
    vehicle_id BIGINT NOT NULL REFERENCES fleet_vehicles(id) ON DELETE CASCADE,
    driver_id BIGINT NOT NULL REFERENCES res_partners(id),
    date_start DATE,
    date_end DATE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fleet_vehicle_odometers (
    id BIGSERIAL PRIMARY KEY,
    vehicle_id BIGINT NOT NULL REFERENCES fleet_vehicles(id) ON DELETE CASCADE,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    value DOUBLE PRECISION NOT NULL,
    unit VARCHAR(10) DEFAULT 'kilometers',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fleet_vehicle_log_services (
    id BIGSERIAL PRIMARY KEY,
    vehicle_id BIGINT NOT NULL REFERENCES fleet_vehicles(id) ON DELETE CASCADE,
    description VARCHAR(255),
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    amount NUMERIC(20, 4) DEFAULT 0.0,
    vendor_id BIGINT REFERENCES res_partners(id),
    odometer DOUBLE PRECISION,
    notes TEXT,
    tag_ids JSONB,
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fleet_vehicle_log_contracts (
    id BIGSERIAL PRIMARY KEY,
    vehicle_id BIGINT NOT NULL REFERENCES fleet_vehicles(id) ON DELETE CASCADE,
    start_date DATE NOT NULL,
    expiration_date DATE,
    cost_generated NUMERIC(20, 4) DEFAULT 0.0,
    cost_frequency VARCHAR(20) DEFAULT 'monthly', -- 'no', 'daily', 'weekly', 'monthly', 'yearly'
    ins_ref VARCHAR(64),
    insurer_id BIGINT REFERENCES res_partners(id),
    state VARCHAR(20) DEFAULT 'open', -- 'open', 'expired', 'closed'
    notes TEXT,
    tag_ids JSONB,
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Triggers for updated_at
CREATE TRIGGER trg_maintenance_equipment_updated_at BEFORE UPDATE ON maintenance_equipment FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_maintenance_requests_updated_at BEFORE UPDATE ON maintenance_requests FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_fleet_vehicles_updated_at BEFORE UPDATE ON fleet_vehicles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_fleet_vehicle_log_services_updated_at BEFORE UPDATE ON fleet_vehicle_log_services FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_fleet_vehicle_log_contracts_updated_at BEFORE UPDATE ON fleet_vehicle_log_contracts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Indexes
CREATE INDEX idx_maintenance_equipment_company ON maintenance_equipment(company_id);
CREATE INDEX idx_maintenance_requests_equipment ON maintenance_requests(equipment_id);
CREATE INDEX idx_fleet_vehicles_license ON fleet_vehicles(license_plate);
CREATE INDEX idx_fleet_vehicle_odometers_vehicle ON fleet_vehicle_odometers(vehicle_id);
CREATE INDEX idx_fleet_vehicle_services_vehicle ON fleet_vehicle_log_services(vehicle_id);


-- ===========================================================================
-- Source: 000027_create_expense_schema.up.sql
-- ===========================================================================
-- Add expense_manager_id to hr_employees
ALTER TABLE hr_employees ADD COLUMN IF NOT EXISTS expense_manager_id BIGINT REFERENCES res_users(id);

-- Expense Table (hr.expense in Odoo)
CREATE TABLE IF NOT EXISTS hr_expenses (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    employee_id BIGINT NOT NULL REFERENCES hr_employees(id) ON DELETE CASCADE,
    manager_id BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    department_id BIGINT REFERENCES hr_departments(id) ON DELETE SET NULL,
    product_id BIGINT REFERENCES product_variants(id) ON DELETE SET NULL,
    unit_amount NUMERIC(19, 4) NOT NULL DEFAULT 0,
    quantity NUMERIC(19, 4) NOT NULL DEFAULT 1,
    total_amount NUMERIC(19, 4) NOT NULL DEFAULT 0,
    untaxed_amount NUMERIC(19, 4) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(19, 4) NOT NULL DEFAULT 0,
    currency_id BIGINT NOT NULL REFERENCES res_currencies(id) ON DELETE RESTRICT,
    payment_mode VARCHAR(50) NOT NULL DEFAULT 'own_account', -- 'own_account', 'company_account'
    account_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL,
    analytic_account_id BIGINT REFERENCES account_analytic_account(id) ON DELETE SET NULL,
    account_move_id BIGINT REFERENCES account_moves(id) ON DELETE SET NULL,
    vendor_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    description TEXT,
    state VARCHAR(50) NOT NULL DEFAULT 'draft', -- 'draft', 'submitted', 'approved', 'posted', 'paid', 'refused'
    approval_date TIMESTAMPTZ,
    refuse_reason TEXT,
    split_origin_id BIGINT REFERENCES hr_expenses(id) ON DELETE SET NULL,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL
);

-- Indexing for performance
CREATE INDEX IF NOT EXISTS idx_hr_expenses_employee_id ON hr_expenses(employee_id);
CREATE INDEX IF NOT EXISTS idx_hr_expenses_manager_id ON hr_expenses(manager_id);
CREATE INDEX IF NOT EXISTS idx_hr_expenses_state ON hr_expenses(state);
CREATE INDEX IF NOT EXISTS idx_hr_expenses_company_id ON hr_expenses(company_id);
CREATE INDEX IF NOT EXISTS idx_hr_expenses_date ON hr_expenses(date);

-- Expense Taxes (Many-to-Many)
CREATE TABLE IF NOT EXISTS hr_expense_taxes (
    expense_id BIGINT NOT NULL REFERENCES hr_expenses(id) ON DELETE CASCADE,
    tax_id BIGINT NOT NULL REFERENCES account_taxes(id) ON DELETE CASCADE,
    PRIMARY KEY (expense_id, tax_id)
);

CREATE TRIGGER trg_hr_expenses_updated_at
    BEFORE UPDATE ON hr_expenses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();


-- ===========================================================================
-- Source: 000028_add_user_email_notification_preference.up.sql
-- ===========================================================================
ALTER TABLE res_users
    ADD COLUMN IF NOT EXISTS email_notifications_enabled BOOLEAN NOT NULL DEFAULT true;

-- ===========================================================================
-- Source: 000029_create_purchase_requisition_schema.up.sql
-- ===========================================================================
-- Purchase requisition schema: blanket orders and templates in Odoo-style workflow

CREATE TABLE IF NOT EXISTS purchase_requisitions (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    requisition_type VARCHAR(32) NOT NULL DEFAULT 'blanket_order',
    vendor_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    user_id BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    date_start TIMESTAMPTZ,
    date_end TIMESTAMPTZ,
    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    currency_id BIGINT REFERENCES res_currencies(id) ON DELETE RESTRICT,
    company_id BIGINT REFERENCES res_companies(id) ON DELETE CASCADE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_vendor_id ON purchase_requisitions(vendor_id);
CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_user_id ON purchase_requisitions(user_id);
CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_state ON purchase_requisitions(state);
CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_company_id ON purchase_requisitions(company_id);
CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_date_start ON purchase_requisitions(date_start);

CREATE TRIGGER trg_purchase_requisitions_updated_at
    BEFORE UPDATE ON purchase_requisitions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS purchase_requisition_lines (
    id BIGSERIAL PRIMARY KEY,
    requisition_id BIGINT NOT NULL REFERENCES purchase_requisitions(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    product_qty NUMERIC(15, 4) NOT NULL DEFAULT 1.0000,
    product_uom BIGINT REFERENCES uom_uoms(id) ON DELETE SET NULL,
    price_unit NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    schedule_date TIMESTAMPTZ,
    supplier_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_requisition_lines_requisition_id ON purchase_requisition_lines(requisition_id);
CREATE INDEX IF NOT EXISTS idx_purchase_requisition_lines_product_id ON purchase_requisition_lines(product_id);

CREATE TRIGGER trg_purchase_requisition_lines_updated_at
    BEFORE UPDATE ON purchase_requisition_lines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();


-- ===========================================================================
-- Source: 000030_link_purchase_orders_to_requisitions.up.sql
-- ===========================================================================
ALTER TABLE purchase_orders
    ADD COLUMN IF NOT EXISTS requisition_id BIGINT REFERENCES purchase_requisitions(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS requisition_type VARCHAR(32);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_requisition_id
    ON purchase_orders(requisition_id);


-- ===========================================================================
-- Source: 000031_seed_purchase_requisition_acl.up.sql
-- ===========================================================================
INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'purchase.requisition', true, true, true, false),
    (2::BIGINT, 'purchase.requisition', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;

-- ===========================================================================
-- Source: 000032_align_mrp_odoo19_literals.up.sql
-- ===========================================================================
-- Align legacy MRP and landed-cost literals with Odoo 19.
UPDATE stock_landed_cost_lines
SET split_method = 'by_current_cost_price'
WHERE split_method = 'by_current_cost';

UPDATE mrp_workorders
SET state = 'ready'
WHERE state = 'pending';

UPDATE mrp_productions
SET reservation_state = 'waiting'
WHERE reservation_state = 'partial';


-- ===========================================================================
-- Source: 000033_add_reserved_quantity_to_stock_moves.up.sql
-- ===========================================================================
ALTER TABLE stock_moves
ADD COLUMN IF NOT EXISTS reserved_quantity NUMERIC(15, 4) NOT NULL DEFAULT 0.0000;

-- ===========================================================================
-- Source: 000034_create_stock_move_lines.up.sql
-- ===========================================================================
CREATE TABLE IF NOT EXISTS stock_move_lines (
    id BIGSERIAL PRIMARY KEY,
    move_id BIGINT NOT NULL REFERENCES stock_moves(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    product_uom BIGINT REFERENCES uom_uoms(id) ON DELETE SET NULL,
    lot_id BIGINT,
    package_id BIGINT,
    owner_id BIGINT,
    location_id BIGINT NOT NULL REFERENCES stock_locations(id) ON DELETE RESTRICT,
    location_dest_id BIGINT NOT NULL REFERENCES stock_locations(id) ON DELETE RESTRICT,
    reserved_quantity NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    quantity_done NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT stock_move_lines_nonnegative CHECK (reserved_quantity >= 0 AND quantity_done >= 0)
);

CREATE INDEX IF NOT EXISTS idx_stock_move_lines_move ON stock_move_lines(move_id);
CREATE INDEX IF NOT EXISTS idx_stock_move_lines_product ON stock_move_lines(product_id);

-- ===========================================================================
-- Source: 000035_create_stock_lots.up.sql
-- ===========================================================================
CREATE TABLE IF NOT EXISTS stock_lots (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    name VARCHAR(128) NOT NULL,
    tracking_mode VARCHAR(16) NOT NULL DEFAULT 'lot',
    company_id BIGINT,
    expiration_at TIMESTAMPTZ,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT stock_lots_tracking_mode CHECK (tracking_mode IN ('lot', 'serial')),
    CONSTRAINT stock_lots_product_name_unique UNIQUE (product_id, name)
);

CREATE INDEX IF NOT EXISTS idx_stock_lots_product ON stock_lots(product_id);

-- ===========================================================================
-- Source: 000036_add_product_tracking.up.sql
-- ===========================================================================
ALTER TABLE product_templates
ADD COLUMN IF NOT EXISTS tracking VARCHAR(16) NOT NULL DEFAULT 'none';

ALTER TABLE product_templates
ADD CONSTRAINT product_templates_tracking_check
CHECK (tracking IN ('none', 'lot', 'serial'));

-- ===========================================================================
-- Source: 000037_add_tracking_mode_to_stock_move_lines.up.sql
-- ===========================================================================
ALTER TABLE stock_move_lines
ADD COLUMN IF NOT EXISTS tracking_mode VARCHAR(16) NOT NULL DEFAULT 'none';

ALTER TABLE stock_move_lines
ADD CONSTRAINT stock_move_lines_tracking_check
CHECK (tracking_mode IN ('none', 'lot', 'serial'));

-- ===========================================================================
-- Source: 000038_align_maintenance_fleet_odoo19.up.sql
-- ===========================================================================
-- Phase 24 (round 2): Align Maintenance & Fleet with Odoo 19.0 behavior.
-- Adds teams, vehicle states as records, service types, recurring maintenance,
-- and kanban stage support. Ship with migration 000039 (ACL).

-- ═══════════════════════════════════════════════════════════════════
-- 1. Maintenance teams (res_team-like) and stage kanban support
-- Teams are now merged into res_groups (group_type = 'maintenance')


-- Maintenance Schema


-- Equipment: Odoo maintenance.equipment additions
ALTER TABLE maintenance_equipment
    ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES res_groups(id),
    ADD COLUMN IF NOT EXISTS partner_id BIGINT REFERENCES res_partners(id),
    ADD COLUMN IF NOT EXISTS partner_ref VARCHAR(64),
    ADD COLUMN IF NOT EXISTS cost NUMERIC(20, 4) DEFAULT 0.0,
    ADD COLUMN IF NOT EXISTS notes TEXT,
    ADD COLUMN IF NOT EXISTS assign_date DATE,
    ADD COLUMN IF NOT EXISTS scrap_date DATE;

-- Requests: Odoo repeat_* preventive maintenance, kanban stage and scheduling
ALTER TABLE maintenance_requests
    ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES res_groups(id),
    ADD COLUMN IF NOT EXISTS kanban_state VARCHAR(20) NOT NULL DEFAULT 'normal',
    ADD COLUMN IF NOT EXISTS schedule_end TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS recurring_maintenance BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS repeat_interval INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS repeat_unit VARCHAR(16) NOT NULL DEFAULT 'week',
    ADD COLUMN IF NOT EXISTS repeat_type VARCHAR(16) NOT NULL DEFAULT 'forever',
    ADD COLUMN IF NOT EXISTS repeat_until DATE,
    ADD COLUMN IF NOT EXISTS archived BOOLEAN NOT NULL DEFAULT false;

-- ═══════════════════════════════════════════════════════════════════
-- 2. Fleet: vehicle states as records, tags M2M, service types
-- ═══════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS fleet_vehicle_states (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    sequence INTEGER NOT NULL DEFAULT 10,
    fold BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO fleet_vehicle_states (name, sequence, fold)
SELECT 'New Request', 4, false
WHERE NOT EXISTS (SELECT 1 FROM fleet_vehicle_states WHERE name = 'New Request');
INSERT INTO fleet_vehicle_states (name, sequence, fold)
SELECT 'To Order', 5, false
WHERE NOT EXISTS (SELECT 1 FROM fleet_vehicle_states WHERE name = 'To Order');
INSERT INTO fleet_vehicle_states (name, sequence, fold)
SELECT 'Ordered', 6, false
WHERE NOT EXISTS (SELECT 1 FROM fleet_vehicle_states WHERE name = 'Ordered');
INSERT INTO fleet_vehicle_states (name, sequence, fold)
SELECT 'Registered', 7, false
WHERE NOT EXISTS (SELECT 1 FROM fleet_vehicle_states WHERE name = 'Registered');
INSERT INTO fleet_vehicle_states (name, sequence, fold)
SELECT 'Downgraded', 8, false
WHERE NOT EXISTS (SELECT 1 FROM fleet_vehicle_states WHERE name = 'Downgraded');
INSERT INTO fleet_vehicle_states (name, sequence, fold)
SELECT 'Reserve', 9, false
WHERE NOT EXISTS (SELECT 1 FROM fleet_vehicle_states WHERE name = 'Reserve');
INSERT INTO fleet_vehicle_states (name, sequence, fold)
SELECT 'Waiting List', 10, false
WHERE NOT EXISTS (SELECT 1 FROM fleet_vehicle_states WHERE name = 'Waiting List');

ALTER TABLE fleet_vehicles
    ADD COLUMN IF NOT EXISTS state_id BIGINT REFERENCES fleet_vehicle_states(id),
    ADD COLUMN IF NOT EXISTS manager_id BIGINT REFERENCES res_users(id);

CREATE TABLE IF NOT EXISTS fleet_service_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(16) NOT NULL DEFAULT 'service', -- 'service' | 'contract'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO fleet_service_types (name, category)
SELECT 'Repair and maintenance', 'service'
WHERE NOT EXISTS (SELECT 1 FROM fleet_service_types WHERE name = 'Repair and maintenance');
INSERT INTO fleet_service_types (name, category)
SELECT 'Omnium', 'contract'
WHERE NOT EXISTS (SELECT 1 FROM fleet_service_types WHERE name = 'Omnium');
INSERT INTO fleet_service_types (name, category)
SELECT 'Leasing', 'contract'
WHERE NOT EXISTS (SELECT 1 FROM fleet_service_types WHERE name = 'Leasing');

ALTER TABLE fleet_vehicle_log_services
    ADD COLUMN IF NOT EXISTS service_type_id BIGINT REFERENCES fleet_service_types(id),
    ADD COLUMN IF NOT EXISTS inv_ref VARCHAR(64),
    ADD COLUMN IF NOT EXISTS state VARCHAR(16) NOT NULL DEFAULT 'new';

ALTER TABLE fleet_vehicle_log_contracts
    ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES res_users(id),
    ADD COLUMN IF NOT EXISTS date DATE,
    ADD COLUMN IF NOT EXISTS name VARCHAR(255);

-- New Request / In Progress / Repaired / Scrap default stages (Odoo data)
INSERT INTO maintenance_stages (name, sequence, fold, done)
SELECT 'New Request', 1, false, false
WHERE NOT EXISTS (SELECT 1 FROM maintenance_stages WHERE name = 'New Request');
INSERT INTO maintenance_stages (name, sequence, fold, done)
SELECT 'In Progress', 2, false, false
WHERE NOT EXISTS (SELECT 1 FROM maintenance_stages WHERE name = 'In Progress');
INSERT INTO maintenance_stages (name, sequence, fold, done)
SELECT 'Repaired', 3, true, true
WHERE NOT EXISTS (SELECT 1 FROM maintenance_stages WHERE name = 'Repaired');
INSERT INTO maintenance_stages (name, sequence, fold, done)
SELECT 'Scrap', 4, true, true
WHERE NOT EXISTS (SELECT 1 FROM maintenance_stages WHERE name = 'Scrap');

-- Indexes
CREATE INDEX IF NOT EXISTS idx_maintenance_equipment_team ON maintenance_equipment(team_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_requests_team ON maintenance_requests(team_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_requests_kanban ON maintenance_requests(kanban_state, archived);
CREATE INDEX IF NOT EXISTS idx_maintenance_requests_recurring ON maintenance_requests(recurring_maintenance, archived);
CREATE INDEX IF NOT EXISTS idx_fleet_vehicles_state ON fleet_vehicles(state_id);
CREATE INDEX IF NOT EXISTS idx_fleet_vehicles_manager ON fleet_vehicles(manager_id);
CREATE INDEX IF NOT EXISTS idx_fleet_services_type ON fleet_vehicle_log_services(service_type_id);
CREATE INDEX IF NOT EXISTS idx_fleet_services_state ON fleet_vehicle_log_services(state);

-- ===========================================================================
-- Source: 000039_seed_maintenance_fleet_acl.up.sql
-- ===========================================================================
-- Phase 24: Seed ACL permissions for Maintenance & Fleet models.
-- Group 1 (internal user): CRUD except delete. Group 2 (manager/admin): full CRUD.

INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'maintenance.equipment', true, true, true, false),
    (2::BIGINT, 'maintenance.equipment', true, true, true, true),
    (1::BIGINT, 'maintenance.equipment.category', true, true, true, false),
    (2::BIGINT, 'maintenance.equipment.category', true, true, true, true),
    (1::BIGINT, 'maintenance.stage', true, false, false, false),
    (2::BIGINT, 'maintenance.stage', true, true, true, true),
    (1::BIGINT, 'maintenance.request', true, true, true, false),
    (2::BIGINT, 'maintenance.request', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle.model', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle.model', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle.model.brand', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle.model.brand', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle.model.category', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle.model.category', true, true, true, true),
    (1::BIGINT, 'fleet.service.type', true, true, true, false),
    (2::BIGINT, 'fleet.service.type', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle.odometer', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle.odometer', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle.log.services', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle.log.services', true, true, true, true),
    (1::BIGINT, 'fleet.vehicle.log.contract', true, true, true, false),
    (2::BIGINT, 'fleet.vehicle.log.contract', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;

-- ===========================================================================
-- Source: 000040_create_loyalty_schema.up.sql
-- ===========================================================================
-- 000040_create_loyalty_schema.up.sql
-- Phase 23: Loyalty & Rewards (loyalty.* / sale_loyalty in Odoo 19.0)

-- 1. Loyalty Programs (loyalty.program)
CREATE TABLE IF NOT EXISTS loyalty_programs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    sequence INT NOT NULL DEFAULT 10,
    company_id BIGINT,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    program_type VARCHAR(20) NOT NULL DEFAULT 'promotion',
    date_from TIMESTAMPTZ,
    date_to TIMESTAMPTZ,
    limit_usage BOOLEAN NOT NULL DEFAULT false,
    max_usage INT NOT NULL DEFAULT 0,
    applies_on VARCHAR(10) NOT NULL DEFAULT 'current',
    trigger VARCHAR(10) NOT NULL DEFAULT 'auto',
    portal_visible BOOLEAN NOT NULL DEFAULT false,
    portal_point_name VARCHAR(64) NOT NULL DEFAULT 'Points',
    is_nominative BOOLEAN NOT NULL DEFAULT false,
    is_payment_program BOOLEAN NOT NULL DEFAULT false,
    sale_ok BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_loyalty_programs_active ON loyalty_programs(active);
CREATE INDEX IF NOT EXISTS idx_loyalty_programs_type ON loyalty_programs(program_type);
CREATE INDEX IF NOT EXISTS idx_loyalty_programs_company ON loyalty_programs(company_id);

CREATE TRIGGER trg_loyalty_programs_updated_at
    BEFORE UPDATE ON loyalty_programs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Program Pricelists junction (many2many)
CREATE TABLE IF NOT EXISTS loyalty_program_pricelists (
    program_id BIGINT NOT NULL REFERENCES loyalty_programs(id) ON DELETE CASCADE,
    pricelist_id BIGINT NOT NULL REFERENCES product_pricelists(id) ON DELETE CASCADE,
    PRIMARY KEY (program_id, pricelist_id)
);

-- 2. Loyalty Rules (loyalty.rule)
CREATE TABLE IF NOT EXISTS loyalty_rules (
    id BIGSERIAL PRIMARY KEY,
    active BOOLEAN NOT NULL DEFAULT true,
    program_id BIGINT NOT NULL REFERENCES loyalty_programs(id) ON DELETE CASCADE,
    company_id BIGINT,
    product_ids BIGINT[] DEFAULT '{}',
    product_category_id BIGINT,
    product_tag_id BIGINT,
    product_domain TEXT,
    reward_point_amount NUMERIC(15,4) NOT NULL DEFAULT 1.0000,
    reward_point_split BOOLEAN NOT NULL DEFAULT false,
    reward_point_mode VARCHAR(8) NOT NULL DEFAULT 'order',
    minimum_qty INT NOT NULL DEFAULT 1,
    minimum_amount NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    minimum_amount_tax_mode VARCHAR(4) NOT NULL DEFAULT 'incl',
    mode VARCHAR(10) NOT NULL DEFAULT 'auto',
    code VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_loyalty_rules_program ON loyalty_rules(program_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_loyalty_rules_code ON loyalty_rules(code) WHERE code IS NOT NULL;

CREATE TRIGGER trg_loyalty_rules_updated_at
    BEFORE UPDATE ON loyalty_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 3. Loyalty Rewards (loyalty.reward)
CREATE TABLE IF NOT EXISTS loyalty_rewards (
    id BIGSERIAL PRIMARY KEY,
    active BOOLEAN NOT NULL DEFAULT true,
    program_id BIGINT NOT NULL REFERENCES loyalty_programs(id) ON DELETE CASCADE,
    description VARCHAR(255),
    reward_type VARCHAR(10) NOT NULL DEFAULT 'discount',
    discount NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    discount_mode VARCHAR(10) NOT NULL DEFAULT 'percent',
    discount_applicability VARCHAR(10) NOT NULL DEFAULT 'order',
    discount_product_ids BIGINT[] DEFAULT '{}',
    discount_product_category_id BIGINT,
    discount_product_tag_id BIGINT,
    discount_max_amount NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    discount_line_product_id BIGINT REFERENCES product_templates(id) ON DELETE RESTRICT,
    reward_product_id BIGINT REFERENCES product_templates(id) ON DELETE SET NULL,
    reward_product_qty INT NOT NULL DEFAULT 1,
    reward_product_uom_id BIGINT,
    required_points NUMERIC(15,4) NOT NULL DEFAULT 1.0000,
    clear_wallet BOOLEAN NOT NULL DEFAULT false,
    product_domain TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_loyalty_rewards_program ON loyalty_rewards(program_id);

CREATE TRIGGER trg_loyalty_rewards_updated_at
    BEFORE UPDATE ON loyalty_rewards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 4. Loyalty Cards (loyalty.card)
CREATE TABLE IF NOT EXISTS loyalty_cards (
    id BIGSERIAL PRIMARY KEY,
    program_id BIGINT NOT NULL REFERENCES loyalty_programs(id) ON DELETE RESTRICT,
    company_id BIGINT,
    partner_id BIGINT,
    points NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    code VARCHAR(64) NOT NULL UNIQUE,
    expiration_date TIMESTAMPTZ,
    use_count INT NOT NULL DEFAULT 0,
    order_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_loyalty_cards_program ON loyalty_cards(program_id);
CREATE INDEX IF NOT EXISTS idx_loyalty_cards_partner ON loyalty_cards(partner_id);
CREATE INDEX IF NOT EXISTS idx_loyalty_cards_active ON loyalty_cards(active);

CREATE TRIGGER trg_loyalty_cards_updated_at
    BEFORE UPDATE ON loyalty_cards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 5. Loyalty History (loyalty.history)
CREATE TABLE IF NOT EXISTS loyalty_card_history (
    id BIGSERIAL PRIMARY KEY,
    card_id BIGINT NOT NULL REFERENCES loyalty_cards(id) ON DELETE CASCADE,
    company_id BIGINT,
    description TEXT NOT NULL,
    issued NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    used NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    order_model VARCHAR(32),
    order_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_loyalty_card_history_card ON loyalty_card_history(card_id);

-- 6. Loyalty Mails (loyalty.mail — config only)
CREATE TABLE IF NOT EXISTS loyalty_mails (
    id BIGSERIAL PRIMARY KEY,
    active BOOLEAN NOT NULL DEFAULT true,
    program_id BIGINT NOT NULL REFERENCES loyalty_programs(id) ON DELETE CASCADE,
    trigger VARCHAR(16) NOT NULL DEFAULT 'create',
    points NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_loyalty_mails_program ON loyalty_mails(program_id);

CREATE TRIGGER trg_loyalty_mails_updated_at
    BEFORE UPDATE ON loyalty_mails
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 7. Sale Order Coupon Points (sale.order.coupon.points)
CREATE TABLE IF NOT EXISTS sale_order_coupon_points (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES sale_orders(id) ON DELETE CASCADE,
    coupon_id BIGINT NOT NULL REFERENCES loyalty_cards(id) ON DELETE CASCADE,
    points NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (order_id, coupon_id)
);

CREATE INDEX IF NOT EXISTS idx_sale_order_coupon_points_coupon ON sale_order_coupon_points(coupon_id);

-- 8. Sale order integration columns
ALTER TABLE sale_orders
    ADD COLUMN IF NOT EXISTS applied_coupon_ids BIGINT[] DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS code_enabled_rule_ids BIGINT[] DEFAULT '{}';

ALTER TABLE sale_order_lines
    ADD COLUMN IF NOT EXISTS reward_id BIGINT,
    ADD COLUMN IF NOT EXISTS coupon_id BIGINT,
    ADD COLUMN IF NOT EXISTS reward_identifier_code VARCHAR(64),
    ADD COLUMN IF NOT EXISTS points_cost NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    ADD COLUMN IF NOT EXISTS is_reward_line BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE sale_order_lines
    ADD CONSTRAINT fk_sale_order_lines_reward FOREIGN KEY (reward_id) REFERENCES loyalty_rewards(id) ON DELETE RESTRICT,
    ADD CONSTRAINT fk_sale_order_lines_coupon FOREIGN KEY (coupon_id) REFERENCES loyalty_cards(id) ON DELETE RESTRICT;

-- ===========================================================================
-- Source: 000041_seed_loyalty_acl.up.sql
-- ===========================================================================
-- 000041_seed_loyalty_acl.up.sql
-- Phase 23: Seed ACL permissions for Loyalty & Rewards models.
-- Group 1 (internal user): CRUD except delete. Group 2 (manager/admin): full CRUD.

INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'loyalty.program', true, true, true, false),
    (2::BIGINT, 'loyalty.program', true, true, true, true),
    (1::BIGINT, 'loyalty.rule', true, true, true, false),
    (2::BIGINT, 'loyalty.rule', true, true, true, true),
    (1::BIGINT, 'loyalty.reward', true, true, true, false),
    (2::BIGINT, 'loyalty.reward', true, true, true, true),
    (1::BIGINT, 'loyalty.card', true, true, true, false),
    (2::BIGINT, 'loyalty.card', true, true, true, true),
    (1::BIGINT, 'loyalty.card.history', true, true, true, false),
    (2::BIGINT, 'loyalty.card.history', true, true, true, true),
    (1::BIGINT, 'loyalty.mail', true, true, true, false),
    (2::BIGINT, 'loyalty.mail', true, true, true, true),
    (1::BIGINT, 'sale.order.coupon.points', true, true, true, false),
    (2::BIGINT, 'sale.order.coupon.points', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;

-- ===========================================================================
-- Source: 000042_align_expenses_odoo19.up.sql
-- ===========================================================================
-- Align the expense schema with the Odoo 19 hr.expense contract.
ALTER TABLE hr_expenses
    ADD COLUMN IF NOT EXISTS attachment_checksums TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE hr_expenses
    DROP CONSTRAINT IF EXISTS hr_expenses_payment_mode_check,
    DROP CONSTRAINT IF EXISTS hr_expenses_state_check,
    DROP CONSTRAINT IF EXISTS hr_expenses_amounts_check,
    DROP CONSTRAINT IF EXISTS hr_expenses_quantity_check;

ALTER TABLE hr_expenses
    ADD CONSTRAINT hr_expenses_payment_mode_check
        CHECK (payment_mode IN ('own_account', 'company_account')),
    ADD CONSTRAINT hr_expenses_state_check
        CHECK (state IN ('draft', 'submitted', 'approved', 'posted', 'in_payment', 'paid', 'refused')),
    ADD CONSTRAINT hr_expenses_amounts_check
        CHECK (unit_amount >= 0 AND total_amount >= 0 AND untaxed_amount >= 0 AND tax_amount >= 0),
    ADD CONSTRAINT hr_expenses_quantity_check
        CHECK (quantity > 0);

CREATE INDEX IF NOT EXISTS idx_hr_expenses_duplicate
    ON hr_expenses (employee_id, date, total_amount)
    WHERE state <> 'refused';

CREATE INDEX IF NOT EXISTS idx_hr_expenses_manager_state
    ON hr_expenses (manager_id, state);

CREATE INDEX IF NOT EXISTS idx_hr_expenses_split_origin
    ON hr_expenses (split_origin_id);

CREATE INDEX IF NOT EXISTS idx_ir_attachments_expense_checksum
    ON ir_attachments (res_model, res_id, checksum)
    WHERE res_model = 'hr.expense';

-- ===========================================================================
-- Source: 000043_seed_expense_acl.up.sql
-- ===========================================================================
-- Phase 19: Expense permissions, matching Odoo employee/approver/admin roles.
INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'hr.expense', true, true, true, false),
    (2::BIGINT, 'hr.expense', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;

-- ===========================================================================
-- Source: 000044_align_purchase_requisition_odoo19.up.sql
-- ===========================================================================
-- Align the existing requisition schema with the Odoo 19 domain contract.

ALTER TABLE purchase_requisitions
    ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS reference VARCHAR(255),
    ADD COLUMN IF NOT EXISTS order_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE purchase_requisition_lines
    ADD COLUMN IF NOT EXISTS qty_ordered NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    ADD COLUMN IF NOT EXISTS product_description_variants VARCHAR(255);

ALTER TABLE purchase_requisitions
    DROP CONSTRAINT IF EXISTS purchase_requisitions_type_check,
    DROP CONSTRAINT IF EXISTS purchase_requisitions_state_check,
    DROP CONSTRAINT IF EXISTS purchase_requisitions_dates_check,
    DROP CONSTRAINT IF EXISTS purchase_requisitions_order_count_check;

ALTER TABLE purchase_requisition_lines
    DROP CONSTRAINT IF EXISTS purchase_requisition_lines_quantity_check,
    DROP CONSTRAINT IF EXISTS purchase_requisition_lines_price_check,
    DROP CONSTRAINT IF EXISTS purchase_requisition_lines_ordered_quantity_check;

ALTER TABLE purchase_requisitions
    ADD CONSTRAINT purchase_requisitions_type_check
        CHECK (requisition_type IN ('blanket_order', 'purchase_template')),
    ADD CONSTRAINT purchase_requisitions_state_check
        CHECK (state IN ('draft', 'confirmed', 'done', 'cancel')),
    ADD CONSTRAINT purchase_requisitions_dates_check
        CHECK (date_end IS NULL OR date_start IS NULL OR date_end >= date_start),
    ADD CONSTRAINT purchase_requisitions_order_count_check
        CHECK (order_count >= 0);

ALTER TABLE purchase_requisition_lines
    ADD CONSTRAINT purchase_requisition_lines_quantity_check
        CHECK (product_qty > 0),
    ADD CONSTRAINT purchase_requisition_lines_price_check
        CHECK (price_unit >= 0),
    ADD CONSTRAINT purchase_requisition_lines_ordered_quantity_check
        CHECK (qty_ordered >= 0);

CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_type
    ON purchase_requisitions(requisition_type);
CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_active
    ON purchase_requisitions(active);
CREATE INDEX IF NOT EXISTS idx_purchase_requisition_lines_supplier
    ON purchase_requisition_lines(supplier_id);

INSERT INTO ir_sequences (name, code, prefix, suffix, padding, start_number, current_number, sequence_type, company_id)
VALUES
    ('Blanket Order', 'purchase.requisition.blanket.order', 'BO', '', 5, 1, 0, 'normal', NULL),
    ('Purchase Template', 'purchase.requisition.purchase.template', 'PT', '', 5, 1, 0, 'normal', NULL)
ON CONFLICT (code) DO NOTHING;

-- ===========================================================================
-- Source: 000045_create_purchase_supplier_info.up.sql
-- ===========================================================================
CREATE TABLE IF NOT EXISTS purchase_supplier_infos (
    id BIGSERIAL PRIMARY KEY,
    requisition_id BIGINT NOT NULL REFERENCES purchase_requisitions(id) ON DELETE CASCADE,
    requisition_line_id BIGINT NOT NULL UNIQUE REFERENCES purchase_requisition_lines(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    vendor_id BIGINT NOT NULL REFERENCES res_partners(id) ON DELETE RESTRICT,
    product_uom BIGINT REFERENCES uom_uoms(id) ON DELETE SET NULL,
    price NUMERIC(15, 4) NOT NULL CHECK (price > 0),
    currency_id BIGINT REFERENCES res_currencies(id) ON DELETE RESTRICT,
    company_id BIGINT REFERENCES res_companies(id) ON DELETE RESTRICT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_supplier_infos_requisition
    ON purchase_supplier_infos(requisition_id);
CREATE INDEX IF NOT EXISTS idx_purchase_supplier_infos_product_vendor
    ON purchase_supplier_infos(product_id, vendor_id);

CREATE TRIGGER trg_purchase_supplier_infos_updated_at
    BEFORE UPDATE ON purchase_supplier_infos
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ===========================================================================
-- Source: 000046_create_purchase_order_groups.up.sql
-- ===========================================================================
CREATE TABLE IF NOT EXISTS purchase_order_groups (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT REFERENCES res_companies(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS purchase_order_group_members (
    group_id BIGINT NOT NULL REFERENCES purchase_order_groups(id) ON DELETE CASCADE,
    order_id BIGINT NOT NULL UNIQUE REFERENCES purchase_orders(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, order_id)
);

CREATE INDEX IF NOT EXISTS idx_purchase_order_group_members_group
    ON purchase_order_group_members(group_id);

CREATE TRIGGER trg_purchase_order_groups_updated_at
    BEFORE UPDATE ON purchase_order_groups
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ===========================================================================
-- Source: 000047_create_delivery_schema.up.sql
-- ===========================================================================
-- Delivery Carrier Table
CREATE TABLE IF NOT EXISTS delivery_carrier (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    sequence INTEGER DEFAULT 10,
    delivery_type VARCHAR(50) NOT NULL, -- 'fixed', 'base_on_rule'
    integration_level VARCHAR(50) DEFAULT 'rate',
    invoice_policy VARCHAR(50) DEFAULT 'estimated',
    product_id BIGINT NOT NULL REFERENCES product_templates(id),
    fixed_price DECIMAL(19,4) DEFAULT 0,
    margin DECIMAL(19,4) DEFAULT 0,
    fixed_margin DECIMAL(19,4) DEFAULT 0,
    free_over BOOLEAN DEFAULT FALSE,
    amount DECIMAL(19,4) DEFAULT 0,
    max_weight DECIMAL(19,4),
    max_volume DECIMAL(19,4),
    company_id BIGINT REFERENCES res_companies(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Delivery Price Rules Table
CREATE TABLE IF NOT EXISTS delivery_price_rule (
    id BIGSERIAL PRIMARY KEY,
    carrier_id BIGINT NOT NULL REFERENCES delivery_carrier(id) ON DELETE CASCADE,
    sequence INTEGER DEFAULT 10,
    variable VARCHAR(50) NOT NULL, -- 'weight', 'volume', 'wv', 'price', 'quantity'
    operator VARCHAR(10) NOT NULL, -- '==', '<=', '<', '>=', '>'
    max_value DECIMAL(19,4) NOT NULL,
    list_base_price DECIMAL(19,4) DEFAULT 0,
    list_price DECIMAL(19,4) DEFAULT 0,
    variable_factor VARCHAR(50) DEFAULT 'weight'
);

-- Delivery Zip Prefix Table
CREATE TABLE IF NOT EXISTS delivery_zip_prefix (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE
);

-- Carrier - Country Relation
CREATE TABLE IF NOT EXISTS delivery_carrier_country_rel (
    carrier_id BIGINT REFERENCES delivery_carrier(id) ON DELETE CASCADE,
    country_id BIGINT REFERENCES res_country(id) ON DELETE CASCADE,
    PRIMARY KEY (carrier_id, country_id)
);

-- Carrier - State Relation
CREATE TABLE IF NOT EXISTS delivery_carrier_state_rel (
    carrier_id BIGINT REFERENCES delivery_carrier(id) ON DELETE CASCADE,
    state_id BIGINT REFERENCES res_country_state(id) ON DELETE CASCADE,
    PRIMARY KEY (carrier_id, state_id)
);

-- Carrier - Zip Prefix Relation
CREATE TABLE IF NOT EXISTS delivery_carrier_zip_prefix_rel (
    carrier_id BIGINT REFERENCES delivery_carrier(id) ON DELETE CASCADE,
    zip_prefix_id BIGINT REFERENCES delivery_zip_prefix(id) ON DELETE CASCADE,
    PRIMARY KEY (carrier_id, zip_prefix_id)
);

-- Add delivery fields to sale_order
ALTER TABLE sale_orders ADD COLUMN IF NOT EXISTS carrier_id BIGINT REFERENCES delivery_carrier(id);
ALTER TABLE sale_orders ADD COLUMN IF NOT EXISTS shipping_weight DECIMAL(19,4) DEFAULT 0;
ALTER TABLE sale_orders ADD COLUMN IF NOT EXISTS delivery_message TEXT;
ALTER TABLE sale_orders ADD COLUMN IF NOT EXISTS recompute_delivery_price BOOLEAN DEFAULT FALSE;

-- Add delivery flag to sale_order_line
ALTER TABLE sale_order_lines ADD COLUMN IF NOT EXISTS is_delivery BOOLEAN DEFAULT FALSE;

-- Add delivery fields to stock_picking
ALTER TABLE stock_pickings ADD COLUMN IF NOT EXISTS carrier_id BIGINT REFERENCES delivery_carrier(id);
ALTER TABLE stock_pickings ADD COLUMN IF NOT EXISTS carrier_tracking_ref VARCHAR(255);
ALTER TABLE stock_pickings ADD COLUMN IF NOT EXISTS weight DECIMAL(19,4) DEFAULT 0;
ALTER TABLE stock_pickings ADD COLUMN IF NOT EXISTS shipping_weight DECIMAL(19,4) DEFAULT 0;
ALTER TABLE stock_pickings ADD COLUMN IF NOT EXISTS number_of_packages INTEGER DEFAULT 0;


-- ===========================================================================
-- Source: 000047_seed_purchase_requisition_alternative_acl.up.sql
-- ===========================================================================
INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'purchase.order.alternative', true, true, true, false),
    (2::BIGINT, 'purchase.order.alternative', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;

-- ===========================================================================
-- Source: 000048_add_chatter_features.up.sql
-- ===========================================================================
CREATE TABLE IF NOT EXISTS mail_message_subtypes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    res_model VARCHAR(128),
    description TEXT,
    internal BOOLEAN NOT NULL DEFAULT false,
    default_subtype BOOLEAN NOT NULL DEFAULT true,
    sequence INT NOT NULL DEFAULT 10,
    CONSTRAINT uq_mail_message_subtypes_name_model UNIQUE (name, res_model)
);

ALTER TABLE mail_messages ADD COLUMN IF NOT EXISTS subtype_id BIGINT REFERENCES mail_message_subtypes(id) ON DELETE SET NULL;
ALTER TABLE mail_messages ADD COLUMN IF NOT EXISTS parent_id BIGINT REFERENCES mail_messages(id) ON DELETE CASCADE;

CREATE TABLE IF NOT EXISTS mail_followers (
    id BIGSERIAL PRIMARY KEY,
    res_model VARCHAR(128) NOT NULL,
    res_id BIGINT NOT NULL,
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES res_users(id) ON DELETE CASCADE,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE CASCADE,
    CONSTRAINT uq_mail_followers_res_partner UNIQUE (res_model, res_id, partner_id),
    CONSTRAINT uq_mail_followers_res_user UNIQUE (res_model, res_id, user_id),
    CONSTRAINT chk_mail_followers_target CHECK (partner_id IS NOT NULL OR user_id IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS mail_tracking_values (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES mail_messages(id) ON DELETE CASCADE,
    field_name VARCHAR(64) NOT NULL,
    field_desc VARCHAR(128) NOT NULL,
    old_value_text TEXT,
    new_value_text TEXT
);

CREATE INDEX IF NOT EXISTS idx_mail_followers_resource ON mail_followers(res_model, res_id);
CREATE INDEX IF NOT EXISTS idx_mail_tracking_message ON mail_tracking_values(message_id);

INSERT INTO mail_message_subtypes (name, description, default_subtype, sequence)
VALUES
    ('discussions', 'Discussions', true, 1),
    ('mt_comment', 'Comment', true, 1),
    ('mt_note', 'Note', false, 10),
    ('activities', 'Activities', false, 20)
ON CONFLICT (name, res_model) DO NOTHING;


