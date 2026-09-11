package timesheet

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type TaskTimer struct {
	ID         int64     `json:"id"`
	TaskID     int64     `json:"task_id"`
	EmployeeID int64     `json:"employee_id"`
	StartTime  time.Time `json:"start_time"`
	IsRunning  bool      `json:"is_running"`
}

func (timer *TaskTimer) Start(now time.Time) error {
	if timer.TaskID <= 0 || timer.EmployeeID <= 0 || timer.IsRunning {
		return platformerrors.Conflict("timer cannot be started in its current state")
	}
	timer.StartTime = now
	timer.IsRunning = true
	return nil
}

func (timer *TaskTimer) Stop(now time.Time) (float64, error) {
	if !timer.IsRunning || !now.After(timer.StartTime) {
		return 0, platformerrors.Conflict("timer is not running or stop time is invalid")
	}
	timer.IsRunning = false
	return now.Sub(timer.StartTime).Hours(), nil
}
