-- 000050_localize_all_entities.up.sql

-- PostgreSQL requires text defaults to be removed before changing the column
-- type to JSONB. The entity migrations use defaults on several name columns.
DO $$
DECLARE
	target RECORD;
BEGIN
	FOR target IN
		SELECT table_name, column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND column_name IN ('name', 'complete_name', 'note', 'description')
		  AND data_type IN ('character varying', 'text')
	LOOP
		EXECUTE format('ALTER TABLE %I ALTER COLUMN %I DROP DEFAULT', target.table_name, target.column_name);
	END LOOP;
END $$;

-- HR
ALTER TABLE hr_departments ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE hr_departments ALTER COLUMN complete_name TYPE JSONB USING jsonb_build_object('en_US', complete_name);
ALTER TABLE hr_jobs ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE hr_leave_allocations ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE hr_leave_requests ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);

-- CRM
ALTER TABLE crm_stages ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE crm_lost_reasons ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE crm_tags ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);

-- UoM
ALTER TABLE uom_uoms ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);

-- Product Attributes
ALTER TABLE product_attributes ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE product_attribute_values ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);

-- Project
ALTER TABLE project_project_stages ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE project_projects ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE project_task_types ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE project_tasks ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE project_task_tags ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE project_milestones ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);

-- Maintenance & Fleet
ALTER TABLE maintenance_equipment_categories ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE maintenance_stages ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE maintenance_equipment ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE maintenance_requests ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE maintenance_teams ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);

ALTER TABLE fleet_vehicle_brands ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE fleet_vehicle_model_categories ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE fleet_vehicle_models ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE fleet_vehicle_tags ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE fleet_vehicle_states ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE fleet_service_types ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);

-- Loyalty
ALTER TABLE loyalty_programs ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE loyalty_rewards ALTER COLUMN description TYPE JSONB USING jsonb_build_object('en_US', description);

-- Analytic
ALTER TABLE account_analytic_plan ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE account_analytic_account ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE account_analytic_line ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);

-- Payment Terms & Taxes & Journals
ALTER TABLE account_payment_terms ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE account_payment_terms ALTER COLUMN note TYPE JSONB USING jsonb_build_object('en_US', note);
ALTER TABLE account_journals ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
ALTER TABLE account_taxes ALTER COLUMN name TYPE JSONB USING jsonb_build_object('en_US', name);
