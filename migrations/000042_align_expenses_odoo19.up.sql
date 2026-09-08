-- Align the expense schema with the Odoo 19 hr.expense contract.
ALTER TABLE hr_expenses
    ADD COLUMN IF NOT EXISTS attachment_checksums TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE hr_expenses
    DROP CONSTRAINT IF EXISTS hr_expenses_payment_mode_check,
    DROP CONSTRAINT IF EXISTS hr_expenses_state_check,
    DROP CONSTRAINT IF EXISTS hr_expenses_amounts_check,
    DROP CONSTRAINT IF EXISTS hr_expenses_quantity_check;

ALTER TABLE hr_expenses
    ADD CONSTRAINT hr_expenses_payment_mode_check
        CHECK (payment_mode IN ('own_account', 'company_account')),
    ADD CONSTRAINT hr_expenses_state_check
        CHECK (state IN ('draft', 'submitted', 'approved', 'posted', 'in_payment', 'paid', 'refused')),
    ADD CONSTRAINT hr_expenses_amounts_check
        CHECK (unit_amount >= 0 AND total_amount >= 0 AND untaxed_amount >= 0 AND tax_amount >= 0),
    ADD CONSTRAINT hr_expenses_quantity_check
        CHECK (quantity > 0);

CREATE INDEX IF NOT EXISTS idx_hr_expenses_duplicate
    ON hr_expenses (employee_id, date, total_amount)
    WHERE state <> 'refused';

CREATE INDEX IF NOT EXISTS idx_hr_expenses_manager_state
    ON hr_expenses (manager_id, state);

CREATE INDEX IF NOT EXISTS idx_hr_expenses_split_origin
    ON hr_expenses (split_origin_id);

CREATE INDEX IF NOT EXISTS idx_ir_attachments_expense_checksum
    ON ir_attachments (res_model, res_id, checksum)
    WHERE res_model = 'hr.expense';