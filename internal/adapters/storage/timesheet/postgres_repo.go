package timesheetstorage

import (
	"context"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/timesheet"
	platformerrors "cashflow_backend/internal/platform/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct{ pool *pgxpool.Pool }

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo { return &PostgresRepo{pool: pool} }
func (repo *PostgresRepo) CreateEntry(ctx context.Context, entry *timesheet.Entry) error {
	return repo.pool.QueryRow(ctx, `INSERT INTO project_timesheets (project_id, task_id, employee_id, user_id, date, unit_amount, name, hourly_cost, amount_total_cost, analytic_account_id, state, billable, invoiced_timesheet, company_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING id, created_at, updated_at`, entry.ProjectID, entry.TaskID, entry.EmployeeID, entry.UserID, entry.Date, entry.UnitAmount, entry.Name, entry.HourlyCost, entry.AmountTotalCost, entry.AnalyticAccountID, entry.State, entry.Billable, entry.InvoicedTimesheet, entry.CompanyID).Scan(&entry.ID, &entry.CreatedAt, &entry.UpdatedAt)
}
func (repo *PostgresRepo) GetEntry(ctx context.Context, id int64) (*timesheet.Entry, error) {
	entry := &timesheet.Entry{ID: id}
	err := repo.pool.QueryRow(ctx, `SELECT project_id, task_id, employee_id, user_id, date, unit_amount, name, hourly_cost, amount_total_cost, analytic_account_id, state, billable, invoiced_timesheet, company_id, created_at, updated_at FROM project_timesheets WHERE id=$1`, id).Scan(&entry.ProjectID, &entry.TaskID, &entry.EmployeeID, &entry.UserID, &entry.Date, &entry.UnitAmount, &entry.Name, &entry.HourlyCost, &entry.AmountTotalCost, &entry.AnalyticAccountID, &entry.State, &entry.Billable, &entry.InvoicedTimesheet, &entry.CompanyID, &entry.CreatedAt, &entry.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, platformerrors.NotFound(fmt.Sprintf("timesheet entry %d not found", id))
	}
	return entry, err
}
func (repo *PostgresRepo) UpdateEntry(ctx context.Context, entry *timesheet.Entry) error {
	result, err := repo.pool.Exec(ctx, `UPDATE project_timesheets SET state=$1, amount_total_cost=$2, updated_at=NOW() WHERE id=$3`, entry.State, entry.AmountTotalCost, entry.ID)
	if err == nil && result.RowsAffected() == 0 {
		return platformerrors.NotFound("timesheet entry not found")
	}
	return err
}
func (repo *PostgresRepo) ListEntries(ctx context.Context, filter timesheet.Filter) ([]timesheet.Entry, error) {
	query := `SELECT id, project_id, task_id, employee_id, user_id, date, unit_amount, name, hourly_cost, amount_total_cost, analytic_account_id, state, billable, invoiced_timesheet, company_id, created_at, updated_at FROM project_timesheets WHERE 1=1`
	args := make([]any, 0, 4)
	add := func(value any, clause string) {
		args = append(args, value)
		query += fmt.Sprintf(" AND %s=$%d", clause, len(args))
	}
	if filter.ProjectID != nil {
		add(*filter.ProjectID, "project_id")
	}
	if filter.EmployeeID != nil {
		add(*filter.EmployeeID, "employee_id")
	}
	if filter.From != nil {
		add(*filter.From, "date")
	}
	if filter.To != nil {
		add(*filter.To, "date")
	}
	query += " ORDER BY date DESC"
	rows, err := repo.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]timesheet.Entry, 0)
	for rows.Next() {
		var entry timesheet.Entry
		if err := rows.Scan(&entry.ID, &entry.ProjectID, &entry.TaskID, &entry.EmployeeID, &entry.UserID, &entry.Date, &entry.UnitAmount, &entry.Name, &entry.HourlyCost, &entry.AmountTotalCost, &entry.AnalyticAccountID, &entry.State, &entry.Billable, &entry.InvoicedTimesheet, &entry.CompanyID, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}
func (repo *PostgresRepo) CreateTimer(ctx context.Context, timer *timesheet.TaskTimer) error {
	return repo.pool.QueryRow(ctx, `INSERT INTO project_task_timers (task_id, employee_id, start_time, is_running) VALUES ($1,$2,$3,$4) RETURNING id`, timer.TaskID, timer.EmployeeID, timer.StartTime, timer.IsRunning).Scan(&timer.ID)
}
func (repo *PostgresRepo) GetRunningTimer(ctx context.Context, employeeID int64) (*timesheet.TaskTimer, error) {
	timer := &timesheet.TaskTimer{}
	err := repo.pool.QueryRow(ctx, `SELECT id, task_id, employee_id, start_time, is_running FROM project_task_timers WHERE employee_id=$1 AND is_running=TRUE`, employeeID).Scan(&timer.ID, &timer.TaskID, &timer.EmployeeID, &timer.StartTime, &timer.IsRunning)
	if err == pgx.ErrNoRows {
		return nil, platformerrors.NotFound("running timer not found")
	}
	return timer, err
}
func (repo *PostgresRepo) DeleteTimer(ctx context.Context, id int64) error {
	_, err := repo.pool.Exec(ctx, `DELETE FROM project_task_timers WHERE id=$1`, id)
	return err
}

var _ time.Time
