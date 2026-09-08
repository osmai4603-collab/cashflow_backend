-- Phase 24 (round 2): Revert Odoo 19.0 alignment for Maintenance & Fleet.

ALTER TABLE fleet_vehicle_log_contracts
    DROP COLUMN IF EXISTS name,
    DROP COLUMN IF EXISTS date,
    DROP COLUMN IF EXISTS user_id;

ALTER TABLE fleet_vehicle_log_services
    DROP COLUMN IF EXISTS state,
    DROP COLUMN IF EXISTS inv_ref,
    DROP COLUMN IF EXISTS service_type_id;

DELETE FROM fleet_service_types WHERE name IN ('Repair and maintenance', 'Omnium', 'Leasing');
DROP TABLE IF EXISTS fleet_service_types;

DROP TABLE IF EXISTS fleet_vehicle_tag_rel;

ALTER TABLE fleet_vehicles
    DROP COLUMN IF EXISTS manager_id,
    DROP COLUMN IF EXISTS state_id;

DELETE FROM fleet_vehicle_states WHERE name IN
    ('New Request', 'To Order', 'Ordered', 'Registered', 'Downgraded', 'Reserve', 'Waiting List');
DROP TABLE IF EXISTS fleet_vehicle_states;

ALTER TABLE maintenance_requests
    DROP COLUMN IF EXISTS archived,
    DROP COLUMN IF EXISTS repeat_until,
    DROP COLUMN IF EXISTS repeat_type,
    DROP COLUMN IF EXISTS repeat_unit,
    DROP COLUMN IF EXISTS repeat_interval,
    DROP COLUMN IF EXISTS recurring_maintenance,
    DROP COLUMN IF EXISTS schedule_end,
    DROP COLUMN IF EXISTS kanban_state,
    DROP COLUMN IF EXISTS team_id;

DELETE FROM maintenance_stages WHERE name IN ('New Request', 'In Progress', 'Repaired', 'Scrap');

ALTER TABLE maintenance_equipment
    DROP COLUMN IF EXISTS scrap_date,
    DROP COLUMN IF EXISTS assign_date,
    DROP COLUMN IF EXISTS notes,
    DROP COLUMN IF EXISTS cost,
    DROP COLUMN IF EXISTS partner_ref,
    DROP COLUMN IF EXISTS partner_id,
    DROP COLUMN IF EXISTS team_id;

DROP TABLE IF EXISTS maintenance_team_members;
DROP TABLE IF EXISTS maintenance_teams;