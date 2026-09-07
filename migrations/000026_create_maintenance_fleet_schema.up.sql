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

CREATE TABLE IF NOT EXISTS fleet_vehicle_tags (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    color INTEGER,
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
