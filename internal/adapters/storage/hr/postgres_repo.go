package hrstorage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/hr"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedDepartmentFilterFields = map[string]string{
	"name":      "name",
	"parent_id": "parent_id",
	"active":    "active",
}

var allowedJobFilterFields = map[string]string{
	"name":          "name",
	"department_id": "department_id",
	"active":        "active",
}

var allowedEmployeeFilterFields = map[string]string{
	"name":          "name",
	"department_id": "department_id",
	"job_id":        "job_id",
	"manager_id":    "manager_id",
	"active":        "active",
	"work_email":    "work_email",
}

var allowedAttendanceFilterFields = map[string]string{
	"employee_id": "employee_id",
	"company_id":  "company_id",
	"check_in":    "check_in",
	"check_out":   "check_out",
}

var allowedOvertimeLineFilterFields = map[string]string{
	"employee_id": "employee_id",
	"company_id":  "company_id",
	"date":        "date",
	"status":      "status",
}

var allowedOvertimeRuleFilterFields = map[string]string{
	"name":       "name",
	"company_id": "company_id",
	"active":     "active",
}

var allowedAllocationFilterFields = map[string]string{
	"employee_id": "employee_id",
	"leave_type":  "leave_type",
	"year":        "year",
	"state":       "state",
}

var allowedLeaveRequestFilterFields = map[string]string{
	"employee_id": "employee_id",
	"leave_type":  "leave_type",
	"state":       "state",
}

// PostgresRepo implements hr.Repository on PostgreSQL.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresRepo creates a new PostgresRepo.
func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

// ─────────────────────────────────────────────────────────────────────────────
// Departments
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateDepartment(ctx context.Context, dept *hr.Department) error {
	query := `
		INSERT INTO hr_departments (
			name, complete_name, parent_id, manager_id, company_id, color, active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		dept.Name, dept.CompleteName, dept.ParentID, dept.ManagerID, dept.CompanyID, dept.Color, dept.Active,
	).Scan(&dept.ID, &dept.CreatedAt, &dept.UpdatedAt)
}

func (r *PostgresRepo) GetDepartmentByID(ctx context.Context, id int64) (*hr.Department, error) {
	query := `
		SELECT
			id, name, complete_name, parent_id, manager_id, company_id, color, active,
			created_at, updated_at, created_by, updated_by
		FROM hr_departments
		WHERE id = $1
	`
	var d hr.Department
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.Name, &d.CompleteName, &d.ParentID, &d.ManagerID, &d.CompanyID, &d.Color, &d.Active,
		&d.CreatedAt, &d.UpdatedAt, &d.CreatedBy, &d.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("department with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get department", err)
	}
	return &d, nil
}

func (r *PostgresRepo) UpdateDepartment(ctx context.Context, dept *hr.Department) error {
	query := `
		UPDATE hr_departments
		SET name = $1, complete_name = $2, parent_id = $3, manager_id = $4,
		    company_id = $5, color = $6, active = $7, updated_at = NOW()
		WHERE id = $8
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		dept.Name, dept.CompleteName, dept.ParentID, dept.ManagerID, dept.CompanyID, dept.Color, dept.Active, dept.ID,
	).Scan(&dept.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("department with ID %d not found", dept.ID))
		}
		return platformerrors.Internal("failed to update department", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteDepartment(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM hr_departments WHERE id = $1", id)
	if err != nil {
		return platformerrors.Internal("failed to delete department", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("department with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListDepartments(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.Department], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedDepartmentFilterFields, 1)
	if err != nil {
		return pagination.PageResult[hr.Department]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM hr_departments %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[hr.Department]{}, platformerrors.Internal("failed to count departments", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedDepartmentFilterFields[page.SortBy]; ok {
			sortBy = col
		}
	}

	limit := page.LimitClamped()
	offset := page.Offset()

	dataQuery := fmt.Sprintf(`
		SELECT
			id, name, complete_name, parent_id, manager_id, company_id, color, active,
			created_at, updated_at, created_by, updated_by
		FROM hr_departments
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return pagination.PageResult[hr.Department]{}, platformerrors.Internal("failed to list departments", err)
	}
	defer rows.Close()

	var list []hr.Department
	for rows.Next() {
		var d hr.Department
		if err := rows.Scan(
			&d.ID, &d.Name, &d.CompleteName, &d.ParentID, &d.ManagerID, &d.CompanyID, &d.Color, &d.Active,
			&d.CreatedAt, &d.UpdatedAt, &d.CreatedBy, &d.UpdatedBy,
		); err != nil {
			return pagination.PageResult[hr.Department]{}, platformerrors.Internal("failed to scan department row", err)
		}
		list = append(list, d)
	}

	return pagination.NewPageResult(list, totalItems, page), nil
}

func (r *PostgresRepo) GetSubDepartments(ctx context.Context, parentID *int64) ([]hr.Department, error) {
	var query string
	var args []any
	if parentID == nil {
		query = `SELECT id, name, complete_name, parent_id, manager_id, company_id, color, active, created_at, updated_at, created_by, updated_by FROM hr_departments WHERE parent_id IS NULL ORDER BY id ASC`
	} else {
		query = `SELECT id, name, complete_name, parent_id, manager_id, company_id, color, active, created_at, updated_at, created_by, updated_by FROM hr_departments WHERE parent_id = $1 ORDER BY id ASC`
		args = append(args, *parentID)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, platformerrors.Internal("failed to query sub departments", err)
	}
	defer rows.Close()

	var list []hr.Department
	for rows.Next() {
		var d hr.Department
		if err := rows.Scan(&d.ID, &d.Name, &d.CompleteName, &d.ParentID, &d.ManagerID, &d.CompanyID, &d.Color, &d.Active, &d.CreatedAt, &d.UpdatedAt, &d.CreatedBy, &d.UpdatedBy); err != nil {
			return nil, platformerrors.Internal("failed to scan sub department", err)
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *PostgresRepo) CountEmployeesByDepartment(ctx context.Context, deptID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM hr_employees WHERE department_id = $1 AND active = true", deptID).Scan(&count)
	if err != nil {
		return 0, platformerrors.Internal("failed to count department employees", err)
	}
	return count, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Jobs
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateJob(ctx context.Context, job *hr.Job) error {
	query := `
		INSERT INTO hr_jobs (
			name, department_id, description, expected_employees, no_of_employee, company_id, active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		job.Name, job.DepartmentID, job.Description, job.ExpectedEmployees, job.NoOfEmployee, job.CompanyID, job.Active,
	).Scan(&job.ID, &job.CreatedAt, &job.UpdatedAt)
}

func (r *PostgresRepo) GetJobByID(ctx context.Context, id int64) (*hr.Job, error) {
	query := `
		SELECT
			id, name, department_id, description, expected_employees, no_of_employee, company_id, active,
			created_at, updated_at, created_by, updated_by
		FROM hr_jobs
		WHERE id = $1
	`
	var j hr.Job
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&j.ID, &j.Name, &j.DepartmentID, &j.Description, &j.ExpectedEmployees, &j.NoOfEmployee, &j.CompanyID, &j.Active,
		&j.CreatedAt, &j.UpdatedAt, &j.CreatedBy, &j.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("job with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get job", err)
	}
	return &j, nil
}

func (r *PostgresRepo) UpdateJob(ctx context.Context, job *hr.Job) error {
	query := `
		UPDATE hr_jobs
		SET name = $1, department_id = $2, description = $3, expected_employees = $4,
		    no_of_employee = $5, company_id = $6, active = $7, updated_at = NOW()
		WHERE id = $8
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		job.Name, job.DepartmentID, job.Description, job.ExpectedEmployees, job.NoOfEmployee, job.CompanyID, job.Active, job.ID,
	).Scan(&job.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("job with ID %d not found", job.ID))
		}
		return platformerrors.Internal("failed to update job", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteJob(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM hr_jobs WHERE id = $1", id)
	if err != nil {
		return platformerrors.Internal("failed to delete job", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("job with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListJobs(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.Job], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedJobFilterFields, 1)
	if err != nil {
		return pagination.PageResult[hr.Job]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM hr_jobs %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[hr.Job]{}, platformerrors.Internal("failed to count jobs", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedJobFilterFields[page.SortBy]; ok {
			sortBy = col
		}
	}

	limit := page.LimitClamped()
	offset := page.Offset()

	dataQuery := fmt.Sprintf(`
		SELECT
			id, name, department_id, description, expected_employees, no_of_employee, company_id, active,
			created_at, updated_at, created_by, updated_by
		FROM hr_jobs
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return pagination.PageResult[hr.Job]{}, platformerrors.Internal("failed to list jobs", err)
	}
	defer rows.Close()

	var list []hr.Job
	for rows.Next() {
		var j hr.Job
		if err := rows.Scan(
			&j.ID, &j.Name, &j.DepartmentID, &j.Description, &j.ExpectedEmployees, &j.NoOfEmployee, &j.CompanyID, &j.Active,
			&j.CreatedAt, &j.UpdatedAt, &j.CreatedBy, &j.UpdatedBy,
		); err != nil {
			return pagination.PageResult[hr.Job]{}, platformerrors.Internal("failed to scan job row", err)
		}
		list = append(list, j)
	}

	return pagination.NewPageResult(list, totalItems, page), nil
}

func (r *PostgresRepo) CountEmployeesByJob(ctx context.Context, jobID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM hr_employees WHERE job_id = $1 AND active = true", jobID).Scan(&count)
	if err != nil {
		return 0, platformerrors.Internal("failed to count job employees", err)
	}
	return count, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Employees
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateEmployee(ctx context.Context, emp *hr.Employee) error {
	query := `
		INSERT INTO hr_employees (
			name, partner_id, department_id, job_id, job_title, manager_id,
			work_email, work_phone, work_location, hire_date, gender, marital_status,
			identification_id, bank_account_no, company_id, active,
			overtime_employee_threshold, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		emp.Name, emp.PartnerID, emp.DepartmentID, emp.JobID, emp.JobTitle, emp.ManagerID,
		emp.WorkEmail, emp.WorkPhone, emp.WorkLocation, emp.HireDate, emp.Gender, emp.MaritalStatus,
		emp.IdentificationID, emp.BankAccountNo, emp.CompanyID, emp.Active,
		emp.OvertimeEmployeeThreshold,
	).Scan(&emp.ID, &emp.CreatedAt, &emp.UpdatedAt)
}

func (r *PostgresRepo) GetEmployeeByID(ctx context.Context, id int64) (*hr.Employee, error) {
	query := `
		SELECT
			id, name, partner_id, department_id, job_id, job_title, manager_id,
			work_email, work_phone, work_location, hire_date, gender, marital_status,
			identification_id, bank_account_no, company_id, active,
			overtime_employee_threshold,
			created_at, updated_at, created_by, updated_by
		FROM hr_employees
		WHERE id = $1
	`
	var e hr.Employee
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.Name, &e.PartnerID, &e.DepartmentID, &e.JobID, &e.JobTitle, &e.ManagerID,
		&e.WorkEmail, &e.WorkPhone, &e.WorkLocation, &e.HireDate, &e.Gender, &e.MaritalStatus,
		&e.IdentificationID, &e.BankAccountNo, &e.CompanyID, &e.Active,
		&e.OvertimeEmployeeThreshold,
		&e.CreatedAt, &e.UpdatedAt, &e.CreatedBy, &e.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("employee with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get employee", err)
	}
	return &e, nil
}

func (r *PostgresRepo) GetEmployeeByPartnerID(ctx context.Context, partnerID int64) (*hr.Employee, error) {
	query := `
		SELECT
			id, name, partner_id, department_id, job_id, job_title, manager_id,
			work_email, work_phone, work_location, hire_date, gender, marital_status,
			identification_id, bank_account_no, company_id, active,
			overtime_employee_threshold,
			created_at, updated_at, created_by, updated_by
		FROM hr_employees
		WHERE partner_id = $1
		LIMIT 1
	`
	var e hr.Employee
	err := r.pool.QueryRow(ctx, query, partnerID).Scan(
		&e.ID, &e.Name, &e.PartnerID, &e.DepartmentID, &e.JobID, &e.JobTitle, &e.ManagerID,
		&e.WorkEmail, &e.WorkPhone, &e.WorkLocation, &e.HireDate, &e.Gender, &e.MaritalStatus,
		&e.IdentificationID, &e.BankAccountNo, &e.CompanyID, &e.Active,
		&e.OvertimeEmployeeThreshold,
		&e.CreatedAt, &e.UpdatedAt, &e.CreatedBy, &e.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("employee with partner ID %d not found", partnerID))
		}
		return nil, platformerrors.Internal("failed to get employee by partner", err)
	}
	return &e, nil
}

func (r *PostgresRepo) UpdateEmployee(ctx context.Context, emp *hr.Employee) error {
	query := `
		UPDATE hr_employees
		SET name = $1, partner_id = $2, department_id = $3, job_id = $4, job_title = $5,
		    manager_id = $6, work_email = $7, work_phone = $8, work_location = $9,
		    hire_date = $10, gender = $11, marital_status = $12, identification_id = $13,
		    bank_account_no = $14, company_id = $15, active = $16,
		    overtime_employee_threshold = $17, updated_at = NOW()
		WHERE id = $18
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		emp.Name, emp.PartnerID, emp.DepartmentID, emp.JobID, emp.JobTitle,
		emp.ManagerID, emp.WorkEmail, emp.WorkPhone, emp.WorkLocation,
		emp.HireDate, emp.Gender, emp.MaritalStatus, emp.IdentificationID,
		emp.BankAccountNo, emp.CompanyID, emp.Active, emp.OvertimeEmployeeThreshold, emp.ID,
	).Scan(&emp.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("employee with ID %d not found", emp.ID))
		}
		return platformerrors.Internal("failed to update employee", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteEmployee(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM hr_employees WHERE id = $1", id)
	if err != nil {
		return platformerrors.Internal("failed to delete employee", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("employee with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListEmployees(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.Employee], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedEmployeeFilterFields, 1)
	if err != nil {
		return pagination.PageResult[hr.Employee]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM hr_employees %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[hr.Employee]{}, platformerrors.Internal("failed to count employees", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedEmployeeFilterFields[page.SortBy]; ok {
			sortBy = col
		}
	}

	limit := page.LimitClamped()
	offset := page.Offset()

	dataQuery := fmt.Sprintf(`
		SELECT
			id, name, partner_id, department_id, job_id, job_title, manager_id,
			work_email, work_phone, work_location, hire_date, gender, marital_status,
			identification_id, bank_account_no, company_id, active,
			overtime_employee_threshold,
			created_at, updated_at, created_by, updated_by
		FROM hr_employees
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return pagination.PageResult[hr.Employee]{}, platformerrors.Internal("failed to list employees", err)
	}
	defer rows.Close()

	var list []hr.Employee
	for rows.Next() {
		var e hr.Employee
		if err := rows.Scan(
			&e.ID, &e.Name, &e.PartnerID, &e.DepartmentID, &e.JobID, &e.JobTitle, &e.ManagerID,
			&e.WorkEmail, &e.WorkPhone, &e.WorkLocation, &e.HireDate, &e.Gender, &e.MaritalStatus,
			&e.IdentificationID, &e.BankAccountNo, &e.CompanyID, &e.Active,
			&e.OvertimeEmployeeThreshold,
			&e.CreatedAt, &e.UpdatedAt, &e.CreatedBy, &e.UpdatedBy,
		); err != nil {
			return pagination.PageResult[hr.Employee]{}, platformerrors.Internal("failed to scan employee row", err)
		}
		list = append(list, e)
	}

	return pagination.NewPageResult(list, totalItems, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Leave Allocations
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateAllocation(ctx context.Context, alloc *hr.LeaveAllocation) error {
	query := `
		INSERT INTO hr_leave_allocations (
			name, employee_id, leave_type, allocated_days, year, state, notes, company_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		alloc.Name, alloc.EmployeeID, alloc.LeaveType, alloc.AllocatedDays, alloc.Year, string(alloc.State),
		alloc.Notes, alloc.CompanyID,
	).Scan(&alloc.ID, &alloc.CreatedAt, &alloc.UpdatedAt)
}

func (r *PostgresRepo) GetAllocationByID(ctx context.Context, id int64) (*hr.LeaveAllocation, error) {
	query := `
		SELECT
			id, name, employee_id, leave_type, allocated_days, year, state, notes, company_id,
			created_at, updated_at, created_by, updated_by
		FROM hr_leave_allocations
		WHERE id = $1
	`
	var a hr.LeaveAllocation
	var state string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.Name, &a.EmployeeID, &a.LeaveType, &a.AllocatedDays, &a.Year, &state, &a.Notes, &a.CompanyID,
		&a.CreatedAt, &a.UpdatedAt, &a.CreatedBy, &a.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("allocation with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get allocation", err)
	}
	a.State = hr.AllocationState(state)
	return &a, nil
}

func (r *PostgresRepo) UpdateAllocation(ctx context.Context, alloc *hr.LeaveAllocation) error {
	query := `
		UPDATE hr_leave_allocations
		SET name = $1, employee_id = $2, leave_type = $3, allocated_days = $4,
		    year = $5, state = $6, notes = $7, company_id = $8, updated_at = NOW()
		WHERE id = $9
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		alloc.Name, alloc.EmployeeID, alloc.LeaveType, alloc.AllocatedDays,
		alloc.Year, string(alloc.State), alloc.Notes, alloc.CompanyID, alloc.ID,
	).Scan(&alloc.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("allocation with ID %d not found", alloc.ID))
		}
		return platformerrors.Internal("failed to update allocation", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteAllocation(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM hr_leave_allocations WHERE id = $1", id)
	if err != nil {
		return platformerrors.Internal("failed to delete allocation", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("allocation with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListAllocations(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.LeaveAllocation], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedAllocationFilterFields, 1)
	if err != nil {
		return pagination.PageResult[hr.LeaveAllocation]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM hr_leave_allocations %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[hr.LeaveAllocation]{}, platformerrors.Internal("failed to count allocations", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedAllocationFilterFields[page.SortBy]; ok {
			sortBy = col
		}
	}

	limit := page.LimitClamped()
	offset := page.Offset()

	dataQuery := fmt.Sprintf(`
		SELECT
			id, name, employee_id, leave_type, allocated_days, year, state, notes, company_id,
			created_at, updated_at, created_by, updated_by
		FROM hr_leave_allocations
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return pagination.PageResult[hr.LeaveAllocation]{}, platformerrors.Internal("failed to list allocations", err)
	}
	defer rows.Close()

	var list []hr.LeaveAllocation
	for rows.Next() {
		var a hr.LeaveAllocation
		var state string
		if err := rows.Scan(
			&a.ID, &a.Name, &a.EmployeeID, &a.LeaveType, &a.AllocatedDays, &a.Year, &state, &a.Notes, &a.CompanyID,
			&a.CreatedAt, &a.UpdatedAt, &a.CreatedBy, &a.UpdatedBy,
		); err != nil {
			return pagination.PageResult[hr.LeaveAllocation]{}, platformerrors.Internal("failed to scan allocation row", err)
		}
		a.State = hr.AllocationState(state)
		list = append(list, a)
	}

	return pagination.NewPageResult(list, totalItems, page), nil
}

func (r *PostgresRepo) GetAllocationsByEmployee(ctx context.Context, employeeID int64, year int) ([]hr.LeaveAllocation, error) {
	var query string
	var args []any
	if year == 0 {
		query = `SELECT id, name, employee_id, leave_type, allocated_days, year, state, notes, company_id, created_at, updated_at, created_by, updated_by FROM hr_leave_allocations WHERE employee_id = $1 AND state = 'approved' ORDER BY id ASC`
		args = append(args, employeeID)
	} else {
		query = `SELECT id, name, employee_id, leave_type, allocated_days, year, state, notes, company_id, created_at, updated_at, created_by, updated_by FROM hr_leave_allocations WHERE employee_id = $1 AND year = $2 AND state = 'approved' ORDER BY id ASC`
		args = append(args, employeeID, year)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, platformerrors.Internal("failed to query employee allocations", err)
	}
	defer rows.Close()

	var list []hr.LeaveAllocation
	for rows.Next() {
		var a hr.LeaveAllocation
		var state string
		if err := rows.Scan(&a.ID, &a.Name, &a.EmployeeID, &a.LeaveType, &a.AllocatedDays, &a.Year, &state, &a.Notes, &a.CompanyID, &a.CreatedAt, &a.UpdatedAt, &a.CreatedBy, &a.UpdatedBy); err != nil {
			return nil, platformerrors.Internal("failed to scan allocation", err)
		}
		a.State = hr.AllocationState(state)
		list = append(list, a)
	}
	return list, nil
}

func (r *PostgresRepo) GetTotalAllocatedDays(ctx context.Context, employeeID int64, leaveType string, year int) (float64, error) {
	var query string
	var args []any
	if year == 0 {
		query = `SELECT COALESCE(SUM(allocated_days), 0) FROM hr_leave_allocations WHERE employee_id = $1 AND leave_type = $2 AND state = 'approved'`
		args = append(args, employeeID, leaveType)
	} else {
		query = `SELECT COALESCE(SUM(allocated_days), 0) FROM hr_leave_allocations WHERE employee_id = $1 AND leave_type = $2 AND year = $3 AND state = 'approved'`
		args = append(args, employeeID, leaveType, year)
	}

	var total float64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return 0, platformerrors.Internal("failed to query total allocated days", err)
	}
	return total, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Leave Requests
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateLeaveRequest(ctx context.Context, req *hr.LeaveRequest) error {
	query := `
		INSERT INTO hr_leave_requests (
			name, employee_id, leave_type, date_from, date_to, days, state, description, approver_id, refusal_reason, company_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		req.Name, req.EmployeeID, req.LeaveType, req.DateFrom, req.DateTo, req.Days, string(req.State),
		req.Description, req.ApproverID, req.RefusalReason, req.CompanyID,
	).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)
}

func (r *PostgresRepo) GetLeaveRequestByID(ctx context.Context, id int64) (*hr.LeaveRequest, error) {
	query := `
		SELECT
			id, name, employee_id, leave_type, date_from, date_to, days, state,
			description, approver_id, refusal_reason, company_id,
			created_at, updated_at, created_by, updated_by
		FROM hr_leave_requests
		WHERE id = $1
	`
	var req hr.LeaveRequest
	var state string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&req.ID, &req.Name, &req.EmployeeID, &req.LeaveType, &req.DateFrom, &req.DateTo, &req.Days, &state,
		&req.Description, &req.ApproverID, &req.RefusalReason, &req.CompanyID,
		&req.CreatedAt, &req.UpdatedAt, &req.CreatedBy, &req.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("leave request with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get leave request", err)
	}
	req.State = hr.LeaveState(state)
	return &req, nil
}

func (r *PostgresRepo) UpdateLeaveRequest(ctx context.Context, req *hr.LeaveRequest) error {
	query := `
		UPDATE hr_leave_requests
		SET name = $1, employee_id = $2, leave_type = $3, date_from = $4, date_to = $5,
		    days = $6, state = $7, description = $8, approver_id = $9, refusal_reason = $10,
		    company_id = $11, updated_at = NOW()
		WHERE id = $12
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		req.Name, req.EmployeeID, req.LeaveType, req.DateFrom, req.DateTo,
		req.Days, string(req.State), req.Description, req.ApproverID, req.RefusalReason,
		req.CompanyID, req.ID,
	).Scan(&req.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("leave request with ID %d not found", req.ID))
		}
		return platformerrors.Internal("failed to update leave request", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteLeaveRequest(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM hr_leave_requests WHERE id = $1", id)
	if err != nil {
		return platformerrors.Internal("failed to delete leave request", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("leave request with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListLeaveRequests(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.LeaveRequest], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedLeaveRequestFilterFields, 1)
	if err != nil {
		return pagination.PageResult[hr.LeaveRequest]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM hr_leave_requests %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[hr.LeaveRequest]{}, platformerrors.Internal("failed to count leave requests", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedLeaveRequestFilterFields[page.SortBy]; ok {
			sortBy = col
		}
	}

	limit := page.LimitClamped()
	offset := page.Offset()

	dataQuery := fmt.Sprintf(`
		SELECT
			id, name, employee_id, leave_type, date_from, date_to, days, state,
			description, approver_id, refusal_reason, company_id,
			created_at, updated_at, created_by, updated_by
		FROM hr_leave_requests
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return pagination.PageResult[hr.LeaveRequest]{}, platformerrors.Internal("failed to list leave requests", err)
	}
	defer rows.Close()

	var list []hr.LeaveRequest
	for rows.Next() {
		var req hr.LeaveRequest
		var state string
		if err := rows.Scan(
			&req.ID, &req.Name, &req.EmployeeID, &req.LeaveType, &req.DateFrom, &req.DateTo, &req.Days, &state,
			&req.Description, &req.ApproverID, &req.RefusalReason, &req.CompanyID,
			&req.CreatedAt, &req.UpdatedAt, &req.CreatedBy, &req.UpdatedBy,
		); err != nil {
			return pagination.PageResult[hr.LeaveRequest]{}, platformerrors.Internal("failed to scan leave request row", err)
		}
		req.State = hr.LeaveState(state)
		list = append(list, req)
	}

	return pagination.NewPageResult(list, totalItems, page), nil
}

func (r *PostgresRepo) GetApprovedLeaveDays(ctx context.Context, employeeID int64, leaveType string, year int) (float64, error) {
	var query string
	var args []any
	if year == 0 {
		query = `SELECT COALESCE(SUM(days), 0) FROM hr_leave_requests WHERE employee_id = $1 AND leave_type = $2 AND state = 'validate'`
		args = append(args, employeeID, leaveType)
	} else {
		query = `SELECT COALESCE(SUM(days), 0) FROM hr_leave_requests WHERE employee_id = $1 AND leave_type = $2 AND EXTRACT(YEAR FROM date_from) = $3 AND state = 'validate'`
		args = append(args, employeeID, leaveType, year)
	}

	var total float64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return 0, platformerrors.Internal("failed to query approved leave days", err)
	}
	return total, nil
}

func (r *PostgresRepo) GetPendingLeaveDays(ctx context.Context, employeeID int64, leaveType string, year int) (float64, error) {
	var query string
	var args []any
	if year == 0 {
		query = `SELECT COALESCE(SUM(days), 0) FROM hr_leave_requests WHERE employee_id = $1 AND leave_type = $2 AND state = 'confirm'`
		args = append(args, employeeID, leaveType)
	} else {
		query = `SELECT COALESCE(SUM(days), 0) FROM hr_leave_requests WHERE employee_id = $1 AND leave_type = $2 AND EXTRACT(YEAR FROM date_from) = $3 AND state = 'confirm'`
		args = append(args, employeeID, leaveType, year)
	}

	var total float64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return 0, platformerrors.Internal("failed to query pending leave days", err)
	}
	return total, nil
}

func (r *PostgresRepo) HasOverlappingLeave(ctx context.Context, employeeID int64, from, to time.Time, excludeID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM hr_leave_requests
			WHERE employee_id = $1
			  AND id != $2
			  AND state IN ('confirm', 'validate')
			  AND date_to >= $3
			  AND date_from <= $4
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, employeeID, excludeID, from, to).Scan(&exists)
	if err != nil {
		return false, platformerrors.Internal("failed to check overlapping leave", err)
	}
	return exists, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Attendance
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateAttendance(ctx context.Context, att *hr.Attendance) error {
	query := `
		INSERT INTO hr_attendance (
			employee_id, check_in, check_out, worked_hours, expected_hours,
			overtime_hours, overtime_status, in_latitude, in_longitude,
			in_ip_address, in_browser, in_mode, out_latitude, out_longitude,
			out_ip_address, out_browser, out_mode, company_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		att.EmployeeID, att.CheckIn, att.CheckOut, att.WorkedHours, att.ExpectedHours,
		att.OvertimeHours, att.OvertimeStatus, att.InLatitude, att.InLongitude,
		att.InIPAddress, att.InBrowser, att.InMode, att.OutLatitude, att.OutLongitude,
		att.OutIPAddress, att.OutBrowser, att.OutMode, att.CompanyID,
	).Scan(&att.ID, &att.CreatedAt, &att.UpdatedAt)
}

func (r *PostgresRepo) GetAttendanceByID(ctx context.Context, id int64) (*hr.Attendance, error) {
	query := `
		SELECT
			id, employee_id, check_in, check_out, worked_hours, expected_hours,
			overtime_hours, overtime_status, in_latitude, in_longitude,
			in_ip_address, in_browser, in_mode, out_latitude, out_longitude,
			out_ip_address, out_browser, out_mode, company_id, created_at, updated_at
		FROM hr_attendance
		WHERE id = $1
	`
	var a hr.Attendance
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.EmployeeID, &a.CheckIn, &a.CheckOut, &a.WorkedHours, &a.ExpectedHours,
		&a.OvertimeHours, &a.OvertimeStatus, &a.InLatitude, &a.InLongitude,
		&a.InIPAddress, &a.InBrowser, &a.InMode, &a.OutLatitude, &a.OutLongitude,
		&a.OutIPAddress, &a.OutBrowser, &a.OutMode, &a.CompanyID, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("attendance with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get attendance", err)
	}
	return &a, nil
}

func (r *PostgresRepo) UpdateAttendance(ctx context.Context, att *hr.Attendance) error {
	query := `
		UPDATE hr_attendance
		SET employee_id = $1, check_in = $2, check_out = $3, worked_hours = $4,
		    expected_hours = $5, overtime_hours = $6, overtime_status = $7,
		    in_latitude = $8, in_longitude = $9, in_ip_address = $10,
		    in_browser = $11, in_mode = $12, out_latitude = $13,
		    out_longitude = $14, out_ip_address = $15, out_browser = $16,
		    out_mode = $17, company_id = $18, updated_at = NOW()
		WHERE id = $19
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		att.EmployeeID, att.CheckIn, att.CheckOut, att.WorkedHours, att.ExpectedHours,
		att.OvertimeHours, att.OvertimeStatus, att.InLatitude, att.InLongitude,
		att.InIPAddress, att.InBrowser, att.InMode, att.OutLatitude, att.OutLongitude,
		att.OutIPAddress, att.OutBrowser, att.OutMode, att.CompanyID, att.ID,
	).Scan(&att.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("attendance with ID %d not found", att.ID))
		}
		return platformerrors.Internal("failed to update attendance", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteAttendance(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM hr_attendance WHERE id = $1", id)
	if err != nil {
		return platformerrors.Internal("failed to delete attendance", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("attendance with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListAttendance(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.Attendance], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedAttendanceFilterFields, 1)
	if err != nil {
		return pagination.PageResult[hr.Attendance]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM hr_attendance %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[hr.Attendance]{}, platformerrors.Internal("failed to count attendance", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedAttendanceFilterFields[page.SortBy]; ok {
			sortBy = col
		}
	}

	limit := page.LimitClamped()
	offset := page.Offset()

	dataQuery := fmt.Sprintf(`
		SELECT
			id, employee_id, check_in, check_out, worked_hours, expected_hours,
			overtime_hours, overtime_status, in_latitude, in_longitude,
			in_ip_address, in_browser, in_mode, out_latitude, out_longitude,
			out_ip_address, out_browser, out_mode, company_id, created_at, updated_at
		FROM hr_attendance
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return pagination.PageResult[hr.Attendance]{}, platformerrors.Internal("failed to list attendance", err)
	}
	defer rows.Close()

	var list []hr.Attendance
	for rows.Next() {
		var a hr.Attendance
		if err := rows.Scan(
			&a.ID, &a.EmployeeID, &a.CheckIn, &a.CheckOut, &a.WorkedHours, &a.ExpectedHours,
			&a.OvertimeHours, &a.OvertimeStatus, &a.InLatitude, &a.InLongitude,
			&a.InIPAddress, &a.InBrowser, &a.InMode, &a.OutLatitude, &a.OutLongitude,
			&a.OutIPAddress, &a.OutBrowser, &a.OutMode, &a.CompanyID, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return pagination.PageResult[hr.Attendance]{}, platformerrors.Internal("failed to scan attendance row", err)
		}
		list = append(list, a)
	}

	return pagination.NewPageResult(list, totalItems, page), nil
}

func (r *PostgresRepo) GetLastAttendance(ctx context.Context, employeeID int64) (*hr.Attendance, error) {
	query := `
		SELECT
			id, employee_id, check_in, check_out, worked_hours, expected_hours,
			overtime_hours, overtime_status, in_latitude, in_longitude,
			in_ip_address, in_browser, in_mode, out_latitude, out_longitude,
			out_ip_address, out_browser, out_mode, company_id, created_at, updated_at
		FROM hr_attendance
		WHERE employee_id = $1
		ORDER BY check_in DESC
		LIMIT 1
	`
	var a hr.Attendance
	err := r.pool.QueryRow(ctx, query, employeeID).Scan(
		&a.ID, &a.EmployeeID, &a.CheckIn, &a.CheckOut, &a.WorkedHours, &a.ExpectedHours,
		&a.OvertimeHours, &a.OvertimeStatus, &a.InLatitude, &a.InLongitude,
		&a.InIPAddress, &a.InBrowser, &a.InMode, &a.OutLatitude, &a.OutLongitude,
		&a.OutIPAddress, &a.OutBrowser, &a.OutMode, &a.CompanyID, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, platformerrors.Internal("failed to get last attendance", err)
	}
	return &a, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Overtime
// ─────────────────────────────────────────────────────────────────────────────

func (r *PostgresRepo) CreateOvertimeLine(ctx context.Context, line *hr.OvertimeLine) error {
	query := `
		INSERT INTO hr_overtime_lines (
			employee_id, attendance_id, date, duration, manual_duration,
			status, time_start, time_stop, rule_ids, company_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query,
		line.EmployeeID, line.AttendanceID, line.Date, line.Duration, line.ManualDuration,
		line.Status, line.TimeStart, line.TimeStop, line.RuleIDs, line.CompanyID,
	).Scan(&line.ID, &line.CreatedAt, &line.UpdatedAt)
}

func (r *PostgresRepo) GetOvertimeLineByID(ctx context.Context, id int64) (*hr.OvertimeLine, error) {
	query := `
		SELECT
			id, employee_id, attendance_id, date, duration, manual_duration,
			status, time_start, time_stop, rule_ids, company_id, created_at, updated_at
		FROM hr_overtime_lines
		WHERE id = $1
	`
	var l hr.OvertimeLine
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&l.ID, &l.EmployeeID, &l.AttendanceID, &l.Date, &l.Duration, &l.ManualDuration,
		&l.Status, &l.TimeStart, &l.TimeStop, &l.RuleIDs, &l.CompanyID, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("overtime line with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get overtime line", err)
	}
	return &l, nil
}

func (r *PostgresRepo) UpdateOvertimeLine(ctx context.Context, line *hr.OvertimeLine) error {
	query := `
		UPDATE hr_overtime_lines
		SET employee_id = $1, attendance_id = $2, date = $3, duration = $4,
		    manual_duration = $5, status = $6, time_start = $7, time_stop = $8,
		    rule_ids = $9, company_id = $10, updated_at = NOW()
		WHERE id = $11
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		line.EmployeeID, line.AttendanceID, line.Date, line.Duration, line.ManualDuration,
		line.Status, line.TimeStart, line.TimeStop, line.RuleIDs, line.CompanyID, line.ID,
	).Scan(&line.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return platformerrors.NotFound(fmt.Sprintf("overtime line with ID %d not found", line.ID))
		}
		return platformerrors.Internal("failed to update overtime line", err)
	}
	return nil
}

func (r *PostgresRepo) DeleteOvertimeLine(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM hr_overtime_lines WHERE id = $1", id)
	if err != nil {
		return platformerrors.Internal("failed to delete overtime line", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("overtime line with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListOvertimeLines(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.OvertimeLine], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedOvertimeLineFilterFields, 1)
	if err != nil {
		return pagination.PageResult[hr.OvertimeLine]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM hr_overtime_lines %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[hr.OvertimeLine]{}, platformerrors.Internal("failed to count overtime lines", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedOvertimeLineFilterFields[page.SortBy]; ok {
			sortBy = col
		}
	}

	limit := page.LimitClamped()
	offset := page.Offset()

	dataQuery := fmt.Sprintf(`
		SELECT
			id, employee_id, attendance_id, date, duration, manual_duration,
			status, time_start, time_stop, rule_ids, company_id, created_at, updated_at
		FROM hr_overtime_lines
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return pagination.PageResult[hr.OvertimeLine]{}, platformerrors.Internal("failed to list overtime lines", err)
	}
	defer rows.Close()

	var list []hr.OvertimeLine
	for rows.Next() {
		var l hr.OvertimeLine
		if err := rows.Scan(
			&l.ID, &l.EmployeeID, &l.AttendanceID, &l.Date, &l.Duration, &l.ManualDuration,
			&l.Status, &l.TimeStart, &l.TimeStop, &l.RuleIDs, &l.CompanyID, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return pagination.PageResult[hr.OvertimeLine]{}, platformerrors.Internal("failed to scan overtime line row", err)
		}
		list = append(list, l)
	}

	return pagination.NewPageResult(list, totalItems, page), nil
}

func (r *PostgresRepo) CreateOvertimeRule(ctx context.Context, rule *hr.OvertimeRule) error {
	query := `
		INSERT INTO hr_overtime_rules (
			name, base_off, timing_type, timing_start, multiplier, active, company_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, NOW(), NOW()
		)
		RETURNING id
	`
	return r.pool.QueryRow(ctx, query,
		rule.Name, rule.BaseOff, rule.TimingType, rule.TimingStart, rule.Multiplier, rule.Active, rule.CompanyID,
	).Scan(&rule.ID)
}

func (r *PostgresRepo) GetOvertimeRuleByID(ctx context.Context, id int64) (*hr.OvertimeRule, error) {
	query := `
		SELECT id, name, base_off, timing_type, timing_start, multiplier, active, company_id
		FROM hr_overtime_rules
		WHERE id = $1
	`
	var ru hr.OvertimeRule
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ru.ID, &ru.Name, &ru.BaseOff, &ru.TimingType, &ru.TimingStart, &ru.Multiplier, &ru.Active, &ru.CompanyID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, platformerrors.NotFound(fmt.Sprintf("overtime rule with ID %d not found", id))
		}
		return nil, platformerrors.Internal("failed to get overtime rule", err)
	}
	return &ru, nil
}

func (r *PostgresRepo) UpdateOvertimeRule(ctx context.Context, rule *hr.OvertimeRule) error {
	query := `
		UPDATE hr_overtime_rules
		SET name = $1, base_off = $2, timing_type = $3, timing_start = $4,
		    multiplier = $5, active = $6, company_id = $7, updated_at = NOW()
		WHERE id = $8
	`
	tag, err := r.pool.Exec(ctx, query,
		rule.Name, rule.BaseOff, rule.TimingType, rule.TimingStart, rule.Multiplier, rule.Active, rule.CompanyID, rule.ID,
	)
	if err != nil {
		return platformerrors.Internal("failed to update overtime rule", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("overtime rule with ID %d not found", rule.ID))
	}
	return nil
}

func (r *PostgresRepo) DeleteOvertimeRule(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM hr_overtime_rules WHERE id = $1", id)
	if err != nil {
		return platformerrors.Internal("failed to delete overtime rule", err)
	}
	if tag.RowsAffected() == 0 {
		return platformerrors.NotFound(fmt.Sprintf("overtime rule with ID %d not found", id))
	}
	return nil
}

func (r *PostgresRepo) ListOvertimeRules(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.OvertimeRule], error) {
	whereClause, args, nextIdx, err := f.BuildWhereClause(allowedOvertimeRuleFilterFields, 1)
	if err != nil {
		return pagination.PageResult[hr.OvertimeRule]{}, platformerrors.Validation("invalid filter", map[string]string{"filter": err.Error()})
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM hr_overtime_rules %s", whereClause)
	var totalItems int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return pagination.PageResult[hr.OvertimeRule]{}, platformerrors.Internal("failed to count overtime rules", err)
	}

	orderDir := page.OrderDirection()
	sortBy := "id"
	if page.SortBy != "" {
		if col, ok := allowedOvertimeRuleFilterFields[page.SortBy]; ok {
			sortBy = col
		}
	}

	limit := page.LimitClamped()
	offset := page.Offset()

	dataQuery := fmt.Sprintf(`
		SELECT id, name, base_off, timing_type, timing_start, multiplier, active, company_id
		FROM hr_overtime_rules
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, orderDir, nextIdx, nextIdx+1)

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return pagination.PageResult[hr.OvertimeRule]{}, platformerrors.Internal("failed to list overtime rules", err)
	}
	defer rows.Close()

	var list []hr.OvertimeRule
	for rows.Next() {
		var ru hr.OvertimeRule
		if err := rows.Scan(
			&ru.ID, &ru.Name, &ru.BaseOff, &ru.TimingType, &ru.TimingStart, &ru.Multiplier, &ru.Active, &ru.CompanyID,
		); err != nil {
			return pagination.PageResult[hr.OvertimeRule]{}, platformerrors.Internal("failed to scan overtime rule row", err)
		}
		list = append(list, ru)
	}

	return pagination.NewPageResult(list, totalItems, page), nil
}
