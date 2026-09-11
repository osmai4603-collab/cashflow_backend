package resource

import "context"

type Repository interface {
	CreateCalendar(ctx context.Context, calendar *Calendar) error
	GetCalendar(ctx context.Context, id int64) (*Calendar, error)
	CreateWorkEntry(ctx context.Context, entry *WorkEntry) error
	ListWorkEntries(ctx context.Context, employeeID int64) ([]WorkEntry, error)
}
