CREATE TABLE IF NOT EXISTS recruitment_stages (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    sequence INT NOT NULL DEFAULT 10,
    folded BOOLEAN NOT NULL DEFAULT FALSE,
    company_id BIGINT NOT NULL REFERENCES res_companies(id)
);

CREATE TABLE IF NOT EXISTS recruitment_applicants (
    id BIGSERIAL PRIMARY KEY,
    partner_name VARCHAR(128) NOT NULL,
    email VARCHAR(128) NOT NULL,
    phone VARCHAR(32),
    job_id BIGINT NOT NULL REFERENCES hr_jobs(id),
    department_id BIGINT REFERENCES hr_departments(id),
    stage_id BIGINT NOT NULL REFERENCES recruitment_stages(id),
    recruiter_user_id BIGINT REFERENCES res_users(id),
    priority INT NOT NULL DEFAULT 0,
    salary_expected NUMERIC(15,2) NOT NULL DEFAULT 0,
    salary_proposed NUMERIC(15,2) NOT NULL DEFAULT 0,
    availability DATE,
    refusal_reason TEXT,
    resume_url TEXT,
    employee_id BIGINT REFERENCES hr_employees(id),
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_recruitment_applicants_job ON recruitment_applicants(job_id);
CREATE INDEX IF NOT EXISTS idx_recruitment_applicants_stage ON recruitment_applicants(stage_id);

CREATE TABLE IF NOT EXISTS recruitment_interviews (
    id BIGSERIAL PRIMARY KEY,
    applicant_id BIGINT NOT NULL REFERENCES recruitment_applicants(id) ON DELETE CASCADE,
    interviewer_id BIGINT NOT NULL REFERENCES hr_employees(id),
    event_id BIGINT REFERENCES calendar_events(id),
    interview_date TIMESTAMPTZ NOT NULL,
    score INT NOT NULL DEFAULT 0 CHECK (score BETWEEN 0 AND 10),
    feedback TEXT,
    recommendation VARCHAR(32) NOT NULL DEFAULT 'consider'
);

CREATE TABLE IF NOT EXISTS project_timesheets (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES project_projects(id),
    task_id BIGINT REFERENCES project_tasks(id),
    employee_id BIGINT NOT NULL REFERENCES hr_employees(id),
    user_id BIGINT NOT NULL REFERENCES res_users(id),
    date DATE NOT NULL,
    unit_amount NUMERIC(6,2) NOT NULL CHECK (unit_amount > 0 AND unit_amount <= 24),
    name TEXT NOT NULL,
    hourly_cost NUMERIC(15,4) NOT NULL DEFAULT 0,
    amount_total_cost NUMERIC(15,4) NOT NULL DEFAULT 0,
    analytic_account_id BIGINT REFERENCES account_analytic_account(id),
    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    billable BOOLEAN NOT NULL DEFAULT TRUE,
    invoiced_timesheet BOOLEAN NOT NULL DEFAULT FALSE,
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_timesheets_employee_date ON project_timesheets(employee_id, date);
CREATE INDEX IF NOT EXISTS idx_timesheets_project ON project_timesheets(project_id);
CREATE INDEX IF NOT EXISTS idx_timesheets_task ON project_timesheets(task_id);

CREATE TABLE IF NOT EXISTS project_task_timers (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES project_tasks(id),
    employee_id BIGINT NOT NULL REFERENCES hr_employees(id),
    start_time TIMESTAMPTZ NOT NULL,
    is_running BOOLEAN NOT NULL DEFAULT TRUE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_running_task_timer_employee ON project_task_timers(employee_id) WHERE is_running = TRUE;

CREATE TABLE IF NOT EXISTS resource_calendars (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    hours_per_day NUMERIC(4,2) NOT NULL DEFAULT 8.0,
    full_time_required_hours NUMERIC(4,2) NOT NULL DEFAULT 40.0,
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS hr_work_entries (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    employee_id BIGINT NOT NULL REFERENCES hr_employees(id),
    work_entry_type VARCHAR(32) NOT NULL DEFAULT 'attendance',
    date_start TIMESTAMPTZ NOT NULL,
    date_stop TIMESTAMPTZ NOT NULL,
    duration_hours NUMERIC(6,2) NOT NULL,
    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    CHECK (date_stop > date_start)
);
CREATE INDEX IF NOT EXISTS idx_work_entries_emp_dates ON hr_work_entries(employee_id, date_start, date_stop);