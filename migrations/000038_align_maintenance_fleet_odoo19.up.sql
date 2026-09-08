-- Phase 24 (round 2): Align Maintenance & Fleet with Odoo 19.0 behavior.
-- Adds teams, vehicle states as records, service types, recurring maintenance,
-- and kanban stage support. Ship with migration 000039 (ACL).

-- ═══════════════════════════════════════════════════════════════════
-- 1. Maintenance teams (res_team-like) and stage kanban support
-- ═══════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS maintenance_teams (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    color INTEGER,
    active BOOLEAN DEFAULT true,
    company_id BIGINT REFERENCES res_companies(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS maintenance_team_members (
    team_id BIGINT NOT NULL REFERENCES maintenance_teams(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES res_users(id) ON DELETE CASCADE,
    PRIMARY KEY (team_id, user_id)
);

-- Equipment: Odoo maintenance.equipment additions
ALTER TABLE maintenance_equipment
    ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES maintenance_teams(id),
    ADD COLUMN IF NOT EXISTS partner_id BIGINT REFERENCES res_partners(id),
    ADD COLUMN IF NOT EXISTS partner_ref VARCHAR(64),
    ADD COLUMN IF NOT EXISTS cost NUMERIC(20, 4) DEFAULT 0.0,
    ADD COLUMN IF NOT EXISTS notes TEXT,
    ADD COLUMN IF NOT EXISTS assign_date DATE,
    ADD COLUMN IF NOT EXISTS scrap_date DATE;

-- Requests: Odoo repeat_* preventive maintenance, kanban stage and scheduling
ALTER TABLE maintenance_requests
    ADD COLUMN IF NOT EXISTS team_id BIGINT REFERENCES maintenance_teams(id),
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

CREATE TABLE IF NOT EXISTS fleet_vehicle_tag_rel (
    vehicle_id BIGINT NOT NULL REFERENCES fleet_vehicles(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES fleet_vehicle_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (vehicle_id, tag_id)
);

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