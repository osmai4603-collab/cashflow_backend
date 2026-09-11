package resourcestorage

import (
	"context"
	"fmt"

	"cashflow_backend/internal/domain/resource"
	platformerrors "cashflow_backend/internal/platform/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct{ pool *pgxpool.Pool }

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo { return &PostgresRepo{pool: pool} }
func (repo *PostgresRepo) CreateCalendar(ctx context.Context, calendarValue *resource.Calendar) error {
	return repo.pool.QueryRow(ctx, `INSERT INTO resource_calendars (name, hours_per_day, full_time_required_hours, company_id, active) VALUES ($1,$2,$3,$4,$5) RETURNING id`, calendarValue.Name, calendarValue.HoursPerDay, calendarValue.FullTimeRequiredHours, calendarValue.CompanyID, calendarValue.Active).Scan(&calendarValue.ID)
}
func (repo *PostgresRepo) GetCalendar(ctx context.Context, id int64) (*resource.Calendar, error) {
	calendarValue := &resource.Calendar{ID: id}
	err := repo.pool.QueryRow(ctx, `SELECT name, hours_per_day, full_time_required_hours, company_id, active FROM resource_calendars WHERE id=$1`, id).Scan(&calendarValue.Name, &calendarValue.HoursPerDay, &calendarValue.FullTimeRequiredHours, &calendarValue.CompanyID, &calendarValue.Active)
	if err == pgx.ErrNoRows {
		return nil, platformerrors.NotFound(fmt.Sprintf("resource calendar %d not found", id))
	}
	return calendarValue, err
}
func (repo *PostgresRepo) CreateWorkEntry(ctx context.Context, entry *resource.WorkEntry) error {
	return repo.pool.QueryRow(ctx, `INSERT INTO hr_work_entries (name, employee_id, work_entry_type, date_start, date_stop, duration_hours, state, company_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, entry.Name, entry.EmployeeID, entry.WorkEntryType, entry.DateStart, entry.DateStop, entry.DurationHours, entry.State, entry.CompanyID).Scan(&entry.ID)
}
func (repo *PostgresRepo) ListWorkEntries(ctx context.Context, employeeID int64) ([]resource.WorkEntry, error) {
	rows, err := repo.pool.Query(ctx, `SELECT id, name, employee_id, work_entry_type, date_start, date_stop, duration_hours, state, company_id FROM hr_work_entries WHERE employee_id=$1 ORDER BY date_start`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]resource.WorkEntry, 0)
	for rows.Next() {
		var entry resource.WorkEntry
		if err := rows.Scan(&entry.ID, &entry.Name, &entry.EmployeeID, &entry.WorkEntryType, &entry.DateStart, &entry.DateStop, &entry.DurationHours, &entry.State, &entry.CompanyID); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}
