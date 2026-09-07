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
    uom_id BIGINT NOT NULL REFERENCES unit_of_measures(id),
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
    uom_id BIGINT NOT NULL REFERENCES unit_of_measures(id),
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
    uom_id BIGINT NOT NULL REFERENCES unit_of_measures(id),
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
    uom_id BIGINT NOT NULL REFERENCES unit_of_measures(id),
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
