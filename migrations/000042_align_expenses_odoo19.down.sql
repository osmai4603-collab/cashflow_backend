DROP INDEX IF EXISTS idx_ir_attachments_expense_checksum;
DROP INDEX IF EXISTS idx_hr_expenses_split_origin;
DROP INDEX IF EXISTS idx_hr_expenses_manager_state;
DROP INDEX IF EXISTS idx_hr_expenses_duplicate;

ALTER TABLE hr_expenses
    DROP CONSTRAINT IF EXISTS hr_expenses_payment_mode_check,
    DROP CONSTRAINT IF EXISTS hr_expenses_state_check,
    DROP CONSTRAINT IF EXISTS hr_expenses_amounts_check,
    DROP CONSTRAINT IF EXISTS hr_expenses_quantity_check;

ALTER TABLE hr_expenses
    DROP COLUMN IF EXISTS attachment_checksums;