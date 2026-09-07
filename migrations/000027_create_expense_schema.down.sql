DROP TABLE IF EXISTS hr_expense_taxes;
DROP TABLE IF EXISTS hr_expenses;
ALTER TABLE hr_employees DROP COLUMN IF EXISTS expense_manager_id;
