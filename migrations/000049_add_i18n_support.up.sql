-- 000049_add_i18n_support.up.sql

-- 1. Create res_lang table for language configurations
CREATE TABLE IF NOT EXISTS res_lang (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(10) NOT NULL UNIQUE, -- e.g. 'en_US', 'ar_SA'
    iso_code VARCHAR(5), -- e.g. 'en', 'ar'
    direction VARCHAR(3) NOT NULL DEFAULT 'ltr', -- 'ltr' or 'rtl'
    date_format VARCHAR(50) NOT NULL DEFAULT '%Y-%m-%d',
    time_format VARCHAR(50) NOT NULL DEFAULT '%H:%M:%S',
    week_start INT NOT NULL DEFAULT 1,
    decimal_point VARCHAR(5) NOT NULL DEFAULT '.',
    thousands_sep VARCHAR(5) NOT NULL DEFAULT ',',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Seed default languages
INSERT INTO res_lang (name, code, iso_code, direction, date_format, week_start, active) VALUES
('English (US)', 'en_US', 'en', 'ltr', '%m/%d/%Y', 7, true),
('Arabic (Saudi Arabia)', 'ar_SA', 'ar', 'rtl', '%d/%m/%Y', 7, true)
ON CONFLICT (code) DO NOTHING;

-- 3. Convert Product Template Name to JSONB
ALTER TABLE product_templates ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);

-- 4. Convert Product Category Name to JSONB
ALTER TABLE product_categories ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);

-- 5. Convert Account Name to JSONB
ALTER TABLE accounts ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);

-- 6. Convert Warehouse and Location names to JSONB
ALTER TABLE warehouses ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE stock_locations ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE stock_locations ALTER COLUMN complete_name TYPE JSONB USING jsonb_build_object('en_US', complete_name);

-- 7. Convert Sale Order Line name (description) to JSONB
ALTER TABLE sale_order_lines ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);

-- 8. Create ir_translation table (optional but good for overrides/legacy)
CREATE TABLE IF NOT EXISTS ir_translation (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL, -- e.g. 'product.template,name'
    res_id BIGINT,
    lang VARCHAR(10) NOT NULL,
    type VARCHAR(20) NOT NULL, -- 'model', 'code', etc.
    src TEXT,
    value TEXT,
    module VARCHAR(100),
    state VARCHAR(20) DEFAULT 'translated',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ir_translation_name_res_id ON ir_translation(name, res_id);
CREATE INDEX IF NOT EXISTS idx_ir_translation_lang ON ir_translation(lang);
