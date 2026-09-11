package timesheetstorage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cashflow_backend/internal/domain/timesheet"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type MemoryRepo struct {
	mu        sync.RWMutex
	entries   map[int64]*timesheet.Entry
	timers    map[int64]*timesheet.TaskTimer
	nextEntry int64
	nextTimer int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{entries: make(map[int64]*timesheet.Entry), timers: make(map[int64]*timesheet.TaskTimer)}
}

func (repo *MemoryRepo) CreateEntry(ctx context.Context, entry *timesheet.Entry) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.nextEntry++
	entry.ID = repo.nextEntry
	clone := *entry
	repo.entries[entry.ID] = &clone
	return nil
}
func (repo *MemoryRepo) GetEntry(ctx context.Context, id int64) (*timesheet.Entry, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	entry, ok := repo.entries[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("timesheet entry %d not found", id))
	}
	clone := *entry
	return &clone, nil
}
func (repo *MemoryRepo) UpdateEntry(ctx context.Context, entry *timesheet.Entry) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if _, ok := repo.entries[entry.ID]; !ok {
		return platformerrors.NotFound("timesheet entry not found")
	}
	clone := *entry
	repo.entries[entry.ID] = &clone
	return nil
}
func (repo *MemoryRepo) ListEntries(ctx context.Context, filter timesheet.Filter) ([]timesheet.Entry, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	result := make([]timesheet.Entry, 0)
	for _, entry := range repo.entries {
		if filter.ProjectID != nil && entry.ProjectID != *filter.ProjectID {
			continue
		}
		if filter.EmployeeID != nil && entry.EmployeeID != *filter.EmployeeID {
			continue
		}
		if filter.From != nil && entry.Date.Before(*filter.From) {
			continue
		}
		if filter.To != nil && entry.Date.After(*filter.To) {
			continue
		}
		result = append(result, *entry)
	}
	return result, nil
}
func (repo *MemoryRepo) CreateTimer(ctx context.Context, timer *timesheet.TaskTimer) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, existing := range repo.timers {
		if existing.EmployeeID == timer.EmployeeID && existing.IsRunning {
			return platformerrors.Conflict("employee already has a running timer")
		}
	}
	repo.nextTimer++
	timer.ID = repo.nextTimer
	clone := *timer
	repo.timers[timer.ID] = &clone
	return nil
}
func (repo *MemoryRepo) GetRunningTimer(ctx context.Context, employeeID int64) (*timesheet.TaskTimer, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	for _, timer := range repo.timers {
		if timer.EmployeeID == employeeID && timer.IsRunning {
			clone := *timer
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound("running timer not found")
}
func (repo *MemoryRepo) DeleteTimer(ctx context.Context, id int64) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	delete(repo.timers, id)
	return nil
}

var _ time.Time
