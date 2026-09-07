-- Add expense_manager_id to hr_employees
ALTER TABLE hr_employees ADD COLUMN IF NOT EXISTS expense_manager_id BIGINT REFERENCES res_users(id);

-- Expense Table (hr.expense in Odoo)
CREATE TABLE IF NOT EXISTS hr_expenses (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    employee_id BIGINT NOT NULL REFERENCES hr_employees(id) ON DELETE CASCADE,
    manager_id BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    department_id BIGINT REFERENCES hr_departments(id) ON DELETE SET NULL,
    product_id BIGINT REFERENCES product_variants(id) ON DELETE SET NULL,
    unit_amount NUMERIC(19, 4) NOT NULL DEFAULT 0,
    quantity NUMERIC(19, 4) NOT NULL DEFAULT 1,
    total_amount NUMERIC(19, 4) NOT NULL DEFAULT 0,
    untaxed_amount NUMERIC(19, 4) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(19, 4) NOT NULL DEFAULT 0,
    currency_id BIGINT NOT NULL REFERENCES res_currencies(id) ON DELETE RESTRICT,
    payment_mode VARCHAR(50) NOT NULL DEFAULT 'own_account', -- 'own_account', 'company_account'
    account_id BIGINT REFERENCES account_accounts(id) ON DELETE SET NULL,
    analytic_account_id BIGINT REFERENCES account_analytic_account(id) ON DELETE SET NULL,
    account_move_id BIGINT REFERENCES account_moves(id) ON DELETE SET NULL,
    vendor_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    description TEXT,
    state VARCHAR(50) NOT NULL DEFAULT 'draft', -- 'draft', 'submitted', 'approved', 'posted', 'paid', 'refused'
    approval_date TIMESTAMPTZ,
    refuse_reason TEXT,
    split_origin_id BIGINT REFERENCES hr_expenses(id) ON DELETE SET NULL,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL
);

-- Indexing for performance
CREATE INDEX IF NOT EXISTS idx_hr_expenses_employee_id ON hr_expenses(employee_id);
CREATE INDEX IF NOT EXISTS idx_hr_expenses_manager_id ON hr_expenses(manager_id);
CREATE INDEX IF NOT EXISTS idx_hr_expenses_state ON hr_expenses(state);
CREATE INDEX IF NOT EXISTS idx_hr_expenses_company_id ON hr_expenses(company_id);
CREATE INDEX IF NOT EXISTS idx_hr_expenses_date ON hr_expenses(date);

-- Expense Taxes (Many-to-Many)
CREATE TABLE IF NOT EXISTS hr_expense_taxes (
    expense_id BIGINT NOT NULL REFERENCES hr_expenses(id) ON DELETE CASCADE,
    tax_id BIGINT NOT NULL REFERENCES account_taxes(id) ON DELETE CASCADE,
    PRIMARY KEY (expense_id, tax_id)
);

CREATE TRIGGER trg_hr_expenses_updated_at
    BEFORE UPDATE ON hr_expenses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
