-- HR Attendance Schema

CREATE TABLE IF NOT EXISTS hr_attendance (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hr_employees(id) ON DELETE CASCADE,
    check_in TIMESTAMP WITH TIME ZONE NOT NULL,
    check_out TIMESTAMP WITH TIME ZONE,
    worked_hours DOUBLE PRECISION DEFAULT 0,
    expected_hours DOUBLE PRECISION DEFAULT 0,
    overtime_hours DOUBLE PRECISION DEFAULT 0,
    overtime_status VARCHAR(20) DEFAULT 'to_approve', -- to_approve, approved, refused

    -- In tracking
    in_latitude DOUBLE PRECISION,
    in_longitude DOUBLE PRECISION,
    in_ip_address VARCHAR(45),
    in_browser TEXT,
    in_mode VARCHAR(20), -- kiosk, systray, manual, technical

    -- Out tracking
    out_latitude DOUBLE PRECISION,
    out_longitude DOUBLE PRECISION,
    out_ip_address VARCHAR(45),
    out_browser TEXT,
    out_mode VARCHAR(20), -- kiosk, systray, manual, technical, auto_check_out

    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_overtime_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    base_off VARCHAR(20) NOT NULL, -- quantity, timing
    timing_type VARCHAR(20),      -- work_days, non_work_days, leave, schedule
    timing_start DOUBLE PRECISION,
    multiplier DOUBLE PRECISION DEFAULT 1.0,
    active BOOLEAN DEFAULT TRUE,
    company_id BIGINT REFERENCES res_companies(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hr_overtime_lines (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hr_employees(id) ON DELETE CASCADE,
    attendance_id BIGINT REFERENCES hr_attendance(id) ON DELETE SET NULL,
    date DATE NOT NULL,
    duration DOUBLE PRECISION DEFAULT 0,
    manual_duration DOUBLE PRECISION DEFAULT 0,
    status VARCHAR(20) DEFAULT 'to_approve', -- to_approve, approved, refused
    time_start TIMESTAMP WITH TIME ZONE,
    time_stop TIMESTAMP WITH TIME ZONE,
    rule_ids BIGINT[],
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add Attendance Configuration to Companies
ALTER TABLE res_companies ADD COLUMN IF NOT EXISTS attendance_kiosk_mode VARCHAR(20) DEFAULT 'barcode_pin';
ALTER TABLE res_companies ADD COLUMN IF NOT EXISTS attendance_kiosk_delay INTEGER DEFAULT 10;
ALTER TABLE res_companies ADD COLUMN IF NOT EXISTS overtime_company_threshold INTEGER DEFAULT 0;
ALTER TABLE res_companies ADD COLUMN IF NOT EXISTS auto_check_out_tolerance DOUBLE PRECISION DEFAULT 0;

-- Add Overtime Threshold to Employees
ALTER TABLE hr_employees ADD COLUMN IF NOT EXISTS overtime_employee_threshold INTEGER DEFAULT 0;

CREATE INDEX idx_hr_attendance_employee ON hr_attendance(employee_id);
CREATE INDEX idx_hr_attendance_check_in ON hr_attendance(check_in);
CREATE INDEX idx_hr_overtime_lines_employee ON hr_overtime_lines(employee_id);
CREATE INDEX idx_hr_overtime_lines_date ON hr_overtime_lines(date);
