-- 000011_create_core_infrastructure_schema.down.sql
-- Rollback Core ERP Infrastructure

-- Drop FKs on existing tables (reverse order)
ALTER TABLE IF EXISTS hr_leave_requests DROP CONSTRAINT IF EXISTS fk_hr_leave_requests_company;
ALTER TABLE IF EXISTS hr_leave_allocations DROP CONSTRAINT IF EXISTS fk_hr_leave_allocations_company;
ALTER TABLE IF EXISTS hr_employees DROP CONSTRAINT IF EXISTS fk_hr_employees_company;
ALTER TABLE IF EXISTS hr_jobs DROP CONSTRAINT IF EXISTS fk_hr_jobs_company;
ALTER TABLE IF EXISTS hr_departments DROP CONSTRAINT IF EXISTS fk_hr_departments_company;
ALTER TABLE IF EXISTS account_payments DROP CONSTRAINT IF EXISTS fk_account_payments_company;
ALTER TABLE IF EXISTS crm_leads DROP CONSTRAINT IF EXISTS fk_crm_leads_company;
ALTER TABLE IF EXISTS crm_stages DROP CONSTRAINT IF EXISTS fk_crm_stages_company;
ALTER TABLE IF EXISTS stock_quants DROP CONSTRAINT IF EXISTS fk_stock_quants_company;
ALTER TABLE IF EXISTS stock_pickings DROP CONSTRAINT IF EXISTS fk_stock_pickings_company;
ALTER TABLE IF EXISTS stock_warehouses DROP CONSTRAINT IF EXISTS fk_stock_warehouses_company;
ALTER TABLE IF EXISTS stock_locations DROP CONSTRAINT IF EXISTS fk_stock_locations_company;
ALTER TABLE IF EXISTS purchase_orders DROP CONSTRAINT IF EXISTS fk_purchase_orders_company;
ALTER TABLE IF EXISTS sale_orders DROP CONSTRAINT IF EXISTS fk_sale_orders_company;
ALTER TABLE IF EXISTS account_accounts DROP CONSTRAINT IF EXISTS fk_account_accounts_company;
ALTER TABLE IF EXISTS product_templates DROP CONSTRAINT IF EXISTS fk_product_templates_company;

-- Drop config parameters
DROP TABLE IF EXISTS ir_config_parameters CASCADE;

-- Drop attachments
DROP TRIGGER IF EXISTS trg_ir_attachments_updated_at ON ir_attachments;
DROP TABLE IF EXISTS ir_attachments CASCADE;

-- Drop sequences
DROP TRIGGER IF EXISTS trg_ir_sequences_updated_at ON ir_sequences;
DROP TABLE IF EXISTS ir_sequences CASCADE;

-- Drop groups M2M and groups
DROP TABLE IF EXISTS res_groups_users_rel CASCADE;
DROP TRIGGER IF EXISTS trg_res_groups_updated_at ON res_groups;
DROP TABLE IF EXISTS res_groups CASCADE;

-- Drop users
DROP TRIGGER IF EXISTS trg_res_users_updated_at ON res_users;
ALTER TABLE res_users DROP CONSTRAINT IF EXISTS fk_res_users_company;
ALTER TABLE res_users DROP CONSTRAINT IF EXISTS fk_res_users_partner;
DROP TABLE IF EXISTS res_users CASCADE;

-- Drop companies
ALTER TABLE res_companies DROP CONSTRAINT IF EXISTS fk_res_companies_partner;
DROP TRIGGER IF EXISTS trg_res_companies_updated_at ON res_companies;
ALTER TABLE res_partners DROP CONSTRAINT IF EXISTS fk_res_partners_company;
DROP TABLE IF EXISTS res_companies CASCADE;

-- Drop currency rates
DROP TRIGGER IF EXISTS trg_res_currency_rates_updated_at ON res_currency_rates;
DROP TABLE IF EXISTS res_currency_rates CASCADE;

-- Drop currencies
DROP TRIGGER IF EXISTS trg_res_currencies_updated_at ON res_currencies;
DROP TABLE IF EXISTS res_currencies CASCADE;
