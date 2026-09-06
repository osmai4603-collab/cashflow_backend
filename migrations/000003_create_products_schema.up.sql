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
