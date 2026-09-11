package timesheetusecase

import (
	"context"
	"time"

	"cashflow_backend/internal/domain/timesheet"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type UseCase struct{ repo timesheet.Repository }

func New(repo timesheet.Repository) *UseCase { return &UseCase{repo: repo} }

func (useCase *UseCase) CreateEntry(ctx context.Context, entry *timesheet.Entry) (*timesheet.Entry, error) {
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	if err := useCase.repo.CreateEntry(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (useCase *UseCase) ListEntries(ctx context.Context, filter timesheet.Filter) ([]timesheet.Entry, error) {
	return useCase.repo.ListEntries(ctx, filter)
}

func (useCase *UseCase) Submit(ctx context.Context, id int64) (*timesheet.Entry, error) {
	entry, err := useCase.repo.GetEntry(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entry.Submit(); err != nil {
		return nil, err
	}
	if err := useCase.repo.UpdateEntry(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (useCase *UseCase) Approve(ctx context.Context, id int64) (*timesheet.Entry, error) {
	entry, err := useCase.repo.GetEntry(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := entry.Approve(); err != nil {
		return nil, err
	}
	if err := useCase.repo.UpdateEntry(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (useCase *UseCase) StartTimer(ctx context.Context, taskID, employeeID int64, now time.Time) (*timesheet.TaskTimer, error) {
	timer := &timesheet.TaskTimer{TaskID: taskID, EmployeeID: employeeID}
	if err := timer.Start(now); err != nil {
		return nil, err
	}
	if err := useCase.repo.CreateTimer(ctx, timer); err != nil {
		return nil, err
	}
	return timer, nil
}

func (useCase *UseCase) StopTimer(ctx context.Context, employeeID int64, now time.Time, projectID, userID, companyID int64, hourlyCost float64) (*timesheet.Entry, error) {
	timer, err := useCase.repo.GetRunningTimer(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	hours, err := timer.Stop(now)
	if err != nil {
		return nil, err
	}
	if projectID <= 0 || userID <= 0 || companyID <= 0 {
		return nil, platformerrors.Validation("project, user, and company are required to stop a timer", nil)
	}
	entry := &timesheet.Entry{ProjectID: projectID, TaskID: &timer.TaskID, EmployeeID: employeeID, UserID: userID, Date: now, UnitAmount: hours, Name: "Task timer", HourlyCost: hourlyCost, CompanyID: companyID}
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	if err := useCase.repo.CreateEntry(ctx, entry); err != nil {
		return nil, err
	}
	if err := useCase.repo.DeleteTimer(ctx, timer.ID); err != nil {
		return nil, err
	}
	return entry, nil
}
