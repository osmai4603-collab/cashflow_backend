-- 000050_localize_all_entities.down.sql

ALTER TABLE account_taxes ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE account_journals ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE account_payment_terms ALTER COLUMN note TYPE TEXT USING note->>'en_US';
ALTER TABLE account_payment_terms ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';

ALTER TABLE account_analytic_line ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE account_analytic_account ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE account_analytic_plan ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';

ALTER TABLE loyalty_rewards ALTER COLUMN description TYPE TEXT USING description->>'en_US';
ALTER TABLE loyalty_programs ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';

ALTER TABLE fleet_service_types ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE fleet_vehicle_states ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE fleet_vehicle_tags ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE fleet_vehicle_models ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE fleet_vehicle_model_categories ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE fleet_vehicle_brands ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';

ALTER TABLE maintenance_teams ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE maintenance_requests ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE maintenance_equipment ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE maintenance_stages ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE maintenance_equipment_categories ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';

ALTER TABLE project_milestones ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE project_task_tags ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE project_tasks ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE project_task_types ALTER COLUMN name TYPE VARCHAR(128) USING name->>'en_US';
ALTER TABLE project_projects ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE project_project_stages ALTER COLUMN name TYPE VARCHAR(128) USING name->>'en_US';

ALTER TABLE product_attribute_values ALTER COLUMN name TYPE VARCHAR(100) USING name->>'en_US';
ALTER TABLE product_attributes ALTER COLUMN name TYPE VARCHAR(100) USING name->>'en_US';

ALTER TABLE uom_uoms ALTER COLUMN name TYPE VARCHAR(100) USING name->>'en_US';

ALTER TABLE crm_tags ALTER COLUMN name TYPE VARCHAR(64) USING name->>'en_US';
ALTER TABLE crm_lost_reasons ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE crm_stages ALTER COLUMN name TYPE VARCHAR(128) USING name->>'en_US';

ALTER TABLE hr_leave_requests ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE hr_leave_allocations ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE hr_jobs ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
ALTER TABLE hr_departments ALTER COLUMN complete_name TYPE VARCHAR(500) USING complete_name->>'en_US';
ALTER TABLE hr_departments ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';
