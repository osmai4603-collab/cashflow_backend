-- Down migration for Maintenance & Fleet Schema

DROP TABLE IF EXISTS fleet_vehicle_log_contracts CASCADE;
DROP TABLE IF EXISTS fleet_vehicle_log_services CASCADE;
DROP TABLE IF EXISTS fleet_vehicle_odometers CASCADE;
DROP TABLE IF EXISTS fleet_vehicle_assignation_logs CASCADE;
DROP TABLE IF EXISTS fleet_vehicles CASCADE;
DROP TABLE IF EXISTS fleet_vehicle_tags CASCADE;
DROP TABLE IF EXISTS fleet_vehicle_models CASCADE;
DROP TABLE IF EXISTS fleet_vehicle_model_categories CASCADE;
DROP TABLE IF EXISTS fleet_vehicle_brands CASCADE;

DROP TABLE IF EXISTS maintenance_requests CASCADE;
DROP TABLE IF EXISTS maintenance_equipment CASCADE;
DROP TABLE IF EXISTS maintenance_stages CASCADE;
DROP TABLE IF EXISTS maintenance_equipment_categories CASCADE;
