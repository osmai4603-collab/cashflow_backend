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
