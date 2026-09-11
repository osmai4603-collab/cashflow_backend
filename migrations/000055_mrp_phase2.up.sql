CREATE TABLE mrp_workorder_time_logs (
    id BIGSERIAL PRIMARY KEY,
    workorder_id BIGINT NOT NULL REFERENCES mrp_workorders(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL DEFAULT 0,
    date_start TIMESTAMPTZ NOT NULL,
    date_end TIMESTAMPTZ,
    duration DOUBLE PRECISION NOT NULL DEFAULT 0,
    loss_id BIGINT,
    CONSTRAINT mrp_time_log_dates CHECK (date_end IS NULL OR date_end >= date_start)
);

CREATE TABLE mrp_workcenter_calendars (
    id BIGSERIAL PRIMARY KEY,
    workcenter_id BIGINT NOT NULL REFERENCES mrp_workcenters(id) ON DELETE CASCADE,
    day_of_week SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    hour_from NUMERIC(4,2) NOT NULL,
    hour_to NUMERIC(4,2) NOT NULL,
    attendance_type VARCHAR(32) NOT NULL DEFAULT 'working',
    company_id BIGINT NOT NULL,
    UNIQUE(workcenter_id, day_of_week, hour_from)
);

CREATE TABLE mrp_capacity_slots (
    id BIGSERIAL PRIMARY KEY,
    workcenter_id BIGINT NOT NULL REFERENCES mrp_workcenters(id) ON DELETE CASCADE,
    date_start TIMESTAMPTZ NOT NULL,
    date_end TIMESTAMPTZ NOT NULL,
    available_hours NUMERIC(10,2) NOT NULL DEFAULT 0,
    allocated_hours NUMERIC(10,2) NOT NULL DEFAULT 0,
    company_id BIGINT NOT NULL,
    CHECK (date_end > date_start)
);
CREATE INDEX idx_mrp_capacity_slots_wc_date ON mrp_capacity_slots(workcenter_id, date_start, date_end);

CREATE TABLE mrp_productivity_losses (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    loss_type VARCHAR(32) NOT NULL,
    company_id BIGINT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE mrp_workcenter_productivity (
    id BIGSERIAL PRIMARY KEY,
    workcenter_id BIGINT NOT NULL REFERENCES mrp_workcenters(id) ON DELETE CASCADE,
    workorder_id BIGINT REFERENCES mrp_workorders(id) ON DELETE SET NULL,
    loss_id BIGINT NOT NULL REFERENCES mrp_productivity_losses(id),
    loss_type VARCHAR(32) NOT NULL,
    date_start TIMESTAMPTZ NOT NULL,
    date_end TIMESTAMPTZ,
    duration DOUBLE PRECISION NOT NULL DEFAULT 0,
    description TEXT,
    company_id BIGINT NOT NULL
);

CREATE TABLE mrp_subcontracting_bom (
    id BIGSERIAL PRIMARY KEY,
    bom_id BIGINT NOT NULL REFERENCES mrp_boms(id) ON DELETE CASCADE,
    subcontractor_id BIGINT NOT NULL,
    lead_time_days INT NOT NULL DEFAULT 0,
    cost_per_unit NUMERIC(15,4) NOT NULL DEFAULT 0,
    company_id BIGINT NOT NULL,
    UNIQUE(bom_id, subcontractor_id)
);

CREATE TABLE mrp_subcontracting_orders (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    production_id BIGINT NOT NULL REFERENCES mrp_productions(id) ON DELETE CASCADE,
    subcontractor_id BIGINT NOT NULL,
    purchase_order_id BIGINT,
    picking_out_id BIGINT,
    picking_in_id BIGINT,
    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    company_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE mrp_quality_points (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    product_id BIGINT,
    operation_id BIGINT REFERENCES mrp_routing_operations(id) ON DELETE SET NULL,
    workcenter_id BIGINT REFERENCES mrp_workcenters(id) ON DELETE SET NULL,
    check_type VARCHAR(32) NOT NULL,
    norm_min DOUBLE PRECISION,
    norm_max DOUBLE PRECISION,
    instructions TEXT,
    company_id BIGINT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE mrp_quality_checks (
    id BIGSERIAL PRIMARY KEY,
    point_id BIGINT NOT NULL REFERENCES mrp_quality_points(id) ON DELETE RESTRICT,
    workorder_id BIGINT REFERENCES mrp_workorders(id) ON DELETE SET NULL,
    production_id BIGINT NOT NULL REFERENCES mrp_productions(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL,
    result TEXT NOT NULL DEFAULT '',
    measure_value DOUBLE PRECISION,
    note TEXT,
    state VARCHAR(16) NOT NULL DEFAULT 'none',
    company_id BIGINT NOT NULL
);