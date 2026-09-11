package timesheet

import (
	"context"
	"time"
)

type Filter struct {
	ProjectID  *int64
	EmployeeID *int64
	From       *time.Time
	To         *time.Time
}

type Repository interface {
	CreateEntry(ctx context.Context, entry *Entry) error
	GetEntry(ctx context.Context, id int64) (*Entry, error)
	UpdateEntry(ctx context.Context, entry *Entry) error
	ListEntries(ctx context.Context, filter Filter) ([]Entry, error)
	CreateTimer(ctx context.Context, timer *TaskTimer) error
	GetRunningTimer(ctx context.Context, employeeID int64) (*TaskTimer, error)
	DeleteTimer(ctx context.Context, id int64) error
}
