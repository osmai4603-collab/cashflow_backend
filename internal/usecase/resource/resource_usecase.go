package resourceusecase

import (
	"context"
	"time"

	"cashflow_backend/internal/domain/resource"
)

type UseCase struct{ repo resource.Repository }

func New(repo resource.Repository) *UseCase { return &UseCase{repo: repo} }

func (useCase *UseCase) CreateCalendar(ctx context.Context, calendarValue *resource.Calendar) (*resource.Calendar, error) {
	if err := calendarValue.Validate(); err != nil {
		return nil, err
	}
	if err := useCase.repo.CreateCalendar(ctx, calendarValue); err != nil {
		return nil, err
	}
	return calendarValue, nil
}

func (useCase *UseCase) CreateWorkEntry(ctx context.Context, entry *resource.WorkEntry) (*resource.WorkEntry, error) {
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	if err := useCase.repo.CreateWorkEntry(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (useCase *UseCase) GenerateAttendanceEntry(ctx context.Context, employeeID, companyID int64, start, stop time.Time) (*resource.WorkEntry, error) {
	entry := &resource.WorkEntry{Name: "Attendance", EmployeeID: employeeID, CompanyID: companyID, WorkEntryType: "attendance", DateStart: start, DateStop: stop, State: "validated"}
	return useCase.CreateWorkEntry(ctx, entry)
}

func (useCase *UseCase) ListWorkEntries(ctx context.Context, employeeID int64) ([]resource.WorkEntry, error) {
	return useCase.repo.ListWorkEntries(ctx, employeeID)
}
