-- 000010_create_hr_schema.down.sql
-- Rollback HR schema

DROP TRIGGER IF EXISTS trg_hr_leave_requests_updated_at ON hr_leave_requests;
DROP TABLE IF EXISTS hr_leave_requests CASCADE;

DROP TRIGGER IF EXISTS trg_hr_leave_allocations_updated_at ON hr_leave_allocations;
DROP TABLE IF EXISTS hr_leave_allocations CASCADE;

ALTER TABLE IF EXISTS hr_departments DROP CONSTRAINT IF EXISTS fk_hr_departments_manager;

DROP TRIGGER IF EXISTS trg_hr_employees_updated_at ON hr_employees;
DROP TABLE IF EXISTS hr_employees CASCADE;

DROP TRIGGER IF EXISTS trg_hr_jobs_updated_at ON hr_jobs;
DROP TABLE IF EXISTS hr_jobs CASCADE;

DROP TRIGGER IF EXISTS trg_hr_departments_updated_at ON hr_departments;
DROP TABLE IF EXISTS hr_departments CASCADE;
