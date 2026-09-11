package resourcestorage

import (
	"context"
	"fmt"
	"sync"

	"cashflow_backend/internal/domain/resource"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type MemoryRepo struct {
	mu            sync.RWMutex
	calendars     map[int64]*resource.Calendar
	workEntries   map[int64]*resource.WorkEntry
	nextCalendar  int64
	nextWorkEntry int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{calendars: make(map[int64]*resource.Calendar), workEntries: make(map[int64]*resource.WorkEntry)}
}
func (repo *MemoryRepo) CreateCalendar(ctx context.Context, calendarValue *resource.Calendar) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.nextCalendar++
	calendarValue.ID = repo.nextCalendar
	clone := *calendarValue
	repo.calendars[calendarValue.ID] = &clone
	return nil
}
func (repo *MemoryRepo) GetCalendar(ctx context.Context, id int64) (*resource.Calendar, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	calendarValue, ok := repo.calendars[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("resource calendar %d not found", id))
	}
	clone := *calendarValue
	return &clone, nil
}
func (repo *MemoryRepo) CreateWorkEntry(ctx context.Context, entry *resource.WorkEntry) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.nextWorkEntry++
	entry.ID = repo.nextWorkEntry
	clone := *entry
	repo.workEntries[entry.ID] = &clone
	return nil
}
func (repo *MemoryRepo) ListWorkEntries(ctx context.Context, employeeID int64) ([]resource.WorkEntry, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	result := make([]resource.WorkEntry, 0)
	for _, entry := range repo.workEntries {
		if entry.EmployeeID == employeeID {
			result = append(result, *entry)
		}
	}
	return result, nil
}
