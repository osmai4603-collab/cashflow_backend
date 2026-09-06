-- 000010_create_hr_schema.up.sql
-- Human Resources (HR) Module: Departments, Jobs, Employees, Leave Allocations, and Leave Requests

-- 1. Departments (hr.department in Odoo)
CREATE TABLE IF NOT EXISTS hr_departments (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    complete_name VARCHAR(500),
    parent_id BIGINT REFERENCES hr_departments(id) ON DELETE SET NULL,
    manager_id BIGINT, -- Foreign key to hr_employees(id) added below
    company_id BIGINT,
    color INTEGER DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_hr_departments_name ON hr_departments(name);
CREATE INDEX IF NOT EXISTS idx_hr_departments_parent_id ON hr_departments(parent_id);
CREATE INDEX IF NOT EXISTS idx_hr_departments_active ON hr_departments(active);

CREATE TRIGGER trg_hr_departments_updated_at
    BEFORE UPDATE ON hr_departments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 2. Jobs / Job Positions (hr.job in Odoo)
CREATE TABLE IF NOT EXISTS hr_jobs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    department_id BIGINT REFERENCES hr_departments(id) ON DELETE SET NULL,
    description TEXT,
    expected_employees INTEGER NOT NULL DEFAULT 1 CHECK (expected_employees >= 0),
    no_of_employee INTEGER NOT NULL DEFAULT 0 CHECK (no_of_employee >= 0),
    company_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_hr_jobs_name ON hr_jobs(name);
CREATE INDEX IF NOT EXISTS idx_hr_jobs_dept ON hr_jobs(department_id);
CREATE INDEX IF NOT EXISTS idx_hr_jobs_active ON hr_jobs(active);

CREATE TRIGGER trg_hr_jobs_updated_at
    BEFORE UPDATE ON hr_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 3. Employees (hr.employee in Odoo)
CREATE TABLE IF NOT EXISTS hr_employees (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    department_id BIGINT REFERENCES hr_departments(id) ON DELETE SET NULL,
    job_id BIGINT REFERENCES hr_jobs(id) ON DELETE SET NULL,
    job_title VARCHAR(255),
    manager_id BIGINT REFERENCES hr_employees(id) ON DELETE SET NULL,
    work_email VARCHAR(255),
    work_phone VARCHAR(50),
    work_location VARCHAR(255),
    hire_date DATE,
    gender VARCHAR(20) DEFAULT 'other',
    marital_status VARCHAR(20) DEFAULT 'single',
    identification_id VARCHAR(100),
    bank_account_no VARCHAR(100),
    company_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_hr_employees_name ON hr_employees(name);
CREATE INDEX IF NOT EXISTS idx_hr_employees_partner_id ON hr_employees(partner_id);
CREATE INDEX IF NOT EXISTS idx_hr_employees_dept ON hr_employees(department_id);
CREATE INDEX IF NOT EXISTS idx_hr_employees_job ON hr_employees(job_id);
CREATE INDEX IF NOT EXISTS idx_hr_employees_manager ON hr_employees(manager_id);
CREATE INDEX IF NOT EXISTS idx_hr_employees_email ON hr_employees(work_email);
CREATE INDEX IF NOT EXISTS idx_hr_employees_active ON hr_employees(active);

CREATE TRIGGER trg_hr_employees_updated_at
    BEFORE UPDATE ON hr_employees
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add circular foreign key for Department Manager
ALTER TABLE hr_departments
    ADD CONSTRAINT fk_hr_departments_manager
    FOREIGN KEY (manager_id)
    REFERENCES hr_employees(id)
    ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_hr_departments_manager_id ON hr_departments(manager_id);

-- 4. Leave Allocations (hr.leave.allocation in Odoo)
CREATE TABLE IF NOT EXISTS hr_leave_allocations (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL DEFAULT '/',
    employee_id BIGINT NOT NULL REFERENCES hr_employees(id) ON DELETE CASCADE,
    leave_type VARCHAR(50) NOT NULL DEFAULT 'annual', -- 'annual', 'sick', 'unpaid', 'emergency', etc.
    allocated_days NUMERIC(5, 2) NOT NULL CHECK (allocated_days >= 0),
    year INTEGER NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'approved', -- 'draft', 'approved', 'cancelled'
    notes TEXT,
    company_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_hr_allocations_emp ON hr_leave_allocations(employee_id);
CREATE INDEX IF NOT EXISTS idx_hr_allocations_type_year ON hr_leave_allocations(leave_type, year);
CREATE INDEX IF NOT EXISTS idx_hr_allocations_state ON hr_leave_allocations(state);

CREATE TRIGGER trg_hr_leave_allocations_updated_at
    BEFORE UPDATE ON hr_leave_allocations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 5. Leave Requests (hr.leave in Odoo)
CREATE TABLE IF NOT EXISTS hr_leave_requests (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL DEFAULT '/',
    employee_id BIGINT NOT NULL REFERENCES hr_employees(id) ON DELETE CASCADE,
    leave_type VARCHAR(50) NOT NULL DEFAULT 'annual',
    date_from DATE NOT NULL,
    date_to DATE NOT NULL,
    days NUMERIC(5, 2) NOT NULL CHECK (days > 0),
    state VARCHAR(20) NOT NULL DEFAULT 'draft', -- 'draft', 'confirm', 'validate', 'refuse', 'cancelled'
    description TEXT,
    approver_id BIGINT REFERENCES hr_employees(id) ON DELETE SET NULL,
    refusal_reason TEXT,
    company_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT,
    CONSTRAINT chk_hr_leave_dates CHECK (date_to >= date_from)
);

CREATE INDEX IF NOT EXISTS idx_hr_leave_requests_emp ON hr_leave_requests(employee_id);
CREATE INDEX IF NOT EXISTS idx_hr_leave_requests_dates ON hr_leave_requests(date_from, date_to);
CREATE INDEX IF NOT EXISTS idx_hr_leave_requests_state ON hr_leave_requests(state);
CREATE INDEX IF NOT EXISTS idx_hr_leave_requests_type ON hr_leave_requests(leave_type);

CREATE TRIGGER trg_hr_leave_requests_updated_at
    BEFORE UPDATE ON hr_leave_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
