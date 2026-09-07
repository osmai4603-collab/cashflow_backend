-- 000012_create_analytic_schema.down.sql
-- Rollback the analytic accounting schema

DELETE FROM ir_config_parameters WHERE key = 'analytic.project_plan';

DROP TABLE IF EXISTS account_analytic_distribution_model CASCADE;
DROP TABLE IF EXISTS account_analytic_line CASCADE;
DROP TABLE IF EXISTS account_analytic_account CASCADE;
DROP TABLE IF EXISTS account_analytic_applicability CASCADE;
DROP TABLE IF EXISTS account_analytic_plan CASCADE;