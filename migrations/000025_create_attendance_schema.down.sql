DROP TABLE IF EXISTS hr_overtime_lines;
DROP TABLE IF EXISTS hr_overtime_rules;
DROP TABLE IF EXISTS hr_attendance;

ALTER TABLE res_companies DROP COLUMN IF EXISTS attendance_kiosk_mode;
ALTER TABLE res_companies DROP COLUMN IF EXISTS attendance_kiosk_delay;
ALTER TABLE res_companies DROP COLUMN IF EXISTS overtime_company_threshold;
ALTER TABLE res_companies DROP COLUMN IF EXISTS auto_check_out_tolerance;

ALTER TABLE hr_employees DROP COLUMN IF EXISTS overtime_employee_threshold;
