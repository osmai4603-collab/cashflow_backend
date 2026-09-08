package expensestorage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cashflow_backend/internal/domain/expense"
	platformerrors "cashflow_backend/internal/platform/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const expenseSelect = `
	SELECT e.id, e.name, e.date, e.employee_id, e.manager_id, e.department_id,
		e.product_id, e.unit_amount, e.quantity, e.total_amount, e.untaxed_amount,
		e.tax_amount, e.currency_id, e.payment_mode, e.account_id,
		e.analytic_account_id, e.account_move_id, e.vendor_id, e.description,
		e.state, e.approval_date, e.refuse_reason,
		COALESCE((SELECT array_agg(tax_id ORDER BY tax_id) FROM hr_expense_taxes WHERE expense_id = e.id), '{}'),
		COALESCE((SELECT array_agg(id ORDER BY id) FROM ir_attachments WHERE res_model = 'hr.expense' AND res_id = e.id), '{}'),
		e.attachment_checksums, e.split_origin_id, e.company_id,
		e.created_at, e.updated_at, e.created_by, e.updated_by
	FROM hr_expenses e`

// PostgresRepo implements expense.Repository on PostgreSQL.
type PostgresRepo struct{ pool *pgxpool.Pool }

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo { return &PostgresRepo{pool: pool} }

func scanExpense(row pgx.Row) (*expense.Expense, error) {
	value := &expense.Expense{}
	err := row.Scan(
		&value.ID, &value.Name, &value.Date, &value.EmployeeID, &value.ManagerID, &value.DepartmentID,
		&value.ProductID, &value.UnitAmount, &value.Quantity, &value.TotalAmount, &value.UntaxedAmount,
		&value.TaxAmount, &value.CurrencyID, &value.PaymentMode, &value.AccountID,
		&value.AnalyticAccountID, &value.AccountMoveID, &value.VendorID, &value.Description,
		&value.State, &value.ApprovalDate, &value.RefuseReason, &value.TaxIDs, &value.AttachmentIDs,
		&value.AttachmentChecksums, &value.SplitOriginID, &value.CompanyID,
		&value.CreatedAt, &value.UpdatedAt, &value.Audit.CreatedBy, &value.Audit.UpdatedBy,
	)
	return value, err
}

func (r *PostgresRepo) Create(ctx context.Context, value *expense.Expense) error {
	if err := value.Validate(); err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return platformerrors.Internal("failed to start expense transaction", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO hr_expenses (
			name, date, employee_id, manager_id, department_id, product_id,
			unit_amount, quantity, total_amount, untaxed_amount, tax_amount,
			currency_id, payment_mode, account_id, analytic_account_id, vendor_id,
			description, state, split_origin_id, company_id, attachment_checksums,
			created_at, updated_at, created_by, updated_by
		) VALUES ($1, COALESCE($2, CURRENT_DATE), $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, NOW(), NOW(), $22, $23)
		RETURNING id, created_at, updated_at`
	if err := tx.QueryRow(ctx, query,
		value.Name, value.Date, value.EmployeeID, value.ManagerID, value.DepartmentID, value.ProductID,
		value.UnitAmount, value.Quantity, value.TotalAmount, value.UntaxedAmount, value.TaxAmount,
		value.CurrencyID, value.PaymentMode, value.AccountID, value.AnalyticAccountID, value.VendorID,
		value.Description, value.State, value.SplitOriginID, value.CompanyID, value.AttachmentChecksums,
		value.Audit.CreatedBy, value.Audit.UpdatedBy,
	).Scan(&value.ID, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return platformerrors.Internal("failed to create expense", err)
	}
	if err := replaceTaxes(ctx, tx, value.ID, value.TaxIDs); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return platformerrors.Internal("failed to commit expense", err)
	}
	return nil
}

func (r *PostgresRepo) Update(ctx context.Context, value *expense.Expense) error {
	if err := value.Validate(); err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return platformerrors.Internal("failed to start expense transaction", err)
	}
	defer tx.Rollback(ctx)
	query := `
		UPDATE hr_expenses SET name=$1, date=$2, employee_id=$3, manager_id=$4,
			department_id=$5, product_id=$6, unit_amount=$7, quantity=$8,
			total_amount=$9, untaxed_amount=$10, tax_amount=$11, currency_id=$12,
			payment_mode=$13, account_id=$14, analytic_account_id=$15, vendor_id=$16,
			description=$17, state=$18, approval_date=$19, refuse_reason=$20,
			split_origin_id=$21, company_id=$22, attachment_checksums=$23,
			updated_at=NOW(), updated_by=$24
		WHERE id=$25
		RETURNING created_at, updated_at`
	if err := tx.QueryRow(ctx, query,
		value.Name, value.Date, value.EmployeeID, value.ManagerID, value.DepartmentID, value.ProductID,
		value.UnitAmount, value.Quantity, value.TotalAmount, value.UntaxedAmount, value.TaxAmount,
		value.CurrencyID, value.PaymentMode, value.AccountID, value.AnalyticAccountID, value.VendorID,
		value.Description, value.State, value.ApprovalDate, value.RefuseReason, value.SplitOriginID,
		value.CompanyID, value.AttachmentChecksums, value.Audit.UpdatedBy, value.ID,
	).Scan(&value.CreatedAt, &value.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("expense with ID %d not found", value.ID))
		}
		return platformerrors.Internal("failed to update expense", err)
	}
	if err := replaceTaxes(ctx, tx, value.ID, value.TaxIDs); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return platformerrors.Internal("failed to commit expense", err)
	}
	return nil
}

func (r *PostgresRepo) Delete(ctx context.Context, id int64) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM hr_expenses WHERE id = $1", id)
	if err != nil {
		return platformerrors.Internal("failed to delete expense", err)
	}
	if result.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("expense with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id int64) (*expense.Expense, error) {
	value, err := scanExpense(r.pool.QueryRow(ctx, expenseSelect+" WHERE e.id = $1", id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("expense with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get expense", err)
	}
	return value, nil
}

func (r *PostgresRepo) List(ctx context.Context, filter expense.Filter) ([]*expense.Expense, error) {
	query := expenseSelect
	args := make([]any, 0, 10)
	conditions := make([]string, 0, 8)
	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}
	if filter.EmployeeID != nil {
		add("e.employee_id = $%d", *filter.EmployeeID)
	}
	if filter.ManagerID != nil {
		add("e.manager_id = $%d", *filter.ManagerID)
	}
	if filter.State != nil {
		add("e.state = $%d", *filter.State)
	}
	if filter.DepartmentID != nil {
		add("e.department_id = $%d", *filter.DepartmentID)
	}
	if filter.CompanyID != nil {
		add("e.company_id = $%d", *filter.CompanyID)
	}
	if filter.DateFrom != nil {
		add("e.date >= $%d::date", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		add("e.date <= $%d::date", *filter.DateTo)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY e.date DESC, e.id DESC"
	limitPosition := len(args) + 1
	offsetPosition := len(args) + 2
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", limitPosition, offsetPosition)
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	args = append(args, limit, maxInt(filter.Offset))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, platformerrors.Internal("failed to list expenses", err)
	}
	defer rows.Close()
	result := make([]*expense.Expense, 0)
	for rows.Next() {
		value, scanErr := scanExpense(rows)
		if scanErr != nil {
			return nil, platformerrors.Internal("failed to scan expense", scanErr)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, platformerrors.Internal("failed to iterate expenses", err)
	}
	return result, nil
}

func (r *PostgresRepo) GetDuplicateExpenses(ctx context.Context, value *expense.Expense) ([]*expense.Expense, error) {
	query := expenseSelect + `
	WHERE e.id <> $1 AND e.state <> 'refused' AND (
		(e.employee_id = $2 AND e.date = $3 AND e.total_amount = $4)
		OR (cardinality($5::text[]) > 0 AND EXISTS (
			SELECT 1 FROM ir_attachments a
			WHERE a.res_model = 'hr.expense' AND a.res_id = e.id AND a.checksum = ANY($5::text[])
		))
	) ORDER BY e.date DESC, e.id DESC`
	rows, err := r.pool.Query(ctx, query, value.ID, value.EmployeeID, value.Date, value.TotalAmount, value.AttachmentChecksums)
	if err != nil {
		return nil, platformerrors.Internal("failed to find duplicate expenses", err)
	}
	defer rows.Close()
	result := make([]*expense.Expense, 0)
	for rows.Next() {
		candidate, scanErr := scanExpense(rows)
		if scanErr != nil {
			return nil, platformerrors.Internal("failed to scan duplicate expense", scanErr)
		}
		result = append(result, candidate)
	}
	return result, rows.Err()
}

func replaceTaxes(ctx context.Context, tx pgx.Tx, expenseID int64, taxIDs []int64) error {
	if _, err := tx.Exec(ctx, "DELETE FROM hr_expense_taxes WHERE expense_id = $1", expenseID); err != nil {
		return platformerrors.Internal("failed to replace expense taxes", err)
	}
	for _, taxID := range taxIDs {
		if _, err := tx.Exec(ctx, "INSERT INTO hr_expense_taxes (expense_id, tax_id) VALUES ($1, $2)", expenseID, taxID); err != nil {
			return platformerrors.Internal("failed to link expense tax", err)
		}
	}
	return nil
}

func maxInt(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

var _ expense.Repository = (*PostgresRepo)(nil)
var _ expense.Repository = (*MemoryRepo)(nil)
