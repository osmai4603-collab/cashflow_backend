package mrp

import (
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// WorkorderState defines the state of a specific operation in production.
type WorkorderState string

const (
	WorkorderStateBlocked  WorkorderState = "blocked"
	WorkorderStateReady    WorkorderState = "ready"
	WorkorderStatePaused   WorkorderState = "paused"
	WorkorderStateProgress WorkorderState = "progress"
	WorkorderStateDone     WorkorderState = "done"
	WorkorderStateCancel   WorkorderState = "cancel"
)

// Workorder represents a manufacturing step (mrp.workorder in Odoo).
type Workorder struct {
	ID           int64          `json:"id"`
	ProductionID int64          `json:"production_id"`
	WorkcenterID int64          `json:"workcenter_id"`
	OperationID  int64          `json:"operation_id"`
	Name         string         `json:"name"`
	Sequence     int            `json:"sequence"`
	State        WorkorderState `json:"state"`

	DurationExpected float64 `json:"duration_expected"` // expected duration in minutes
	Duration         float64 `json:"duration"`          // real duration in minutes

	DateStart    *time.Time         `json:"date_start,omitempty"`
	DateFinished *time.Time         `json:"date_finished,omitempty"`
	QtyProduced  float64            `json:"qty_produced"`
	TimeLogs     []WorkorderTimeLog `json:"time_logs,omitempty"`

	Audit audit.Fields `json:"audit"`
}

// Validate ensures a work order has the minimal data required to execute.
func (w *Workorder) Validate() error {
	if w.ProductionID <= 0 {
		return platformerrors.Validation("work order needs a production", map[string]string{
			"production_id": "must be > 0",
		})
	}
	if w.WorkcenterID <= 0 {
		return platformerrors.Validation("work order needs a workcenter", map[string]string{
			"workcenter_id": "must be > 0",
		})
	}
	w.Name = strings.TrimSpace(w.Name)
	if w.Name == "" {
		return platformerrors.Validation("work order name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if w.State == "" {
		w.State = WorkorderStateReady
	}
	switch w.State {
	case WorkorderStateBlocked, WorkorderStateReady, WorkorderStateProgress,
		WorkorderStatePaused, WorkorderStateDone, WorkorderStateCancel:
	default:
		return platformerrors.Validation("invalid work order state", map[string]string{
			"state": "must be blocked, ready, paused, progress, done or cancel",
		})
	}
	if w.DurationExpected < 0 {
		return platformerrors.Validation("expected duration cannot be negative", map[string]string{
			"duration_expected": "must be >= 0",
		})
	}
	if w.Duration < 0 {
		return platformerrors.Validation("duration cannot be negative", map[string]string{
			"duration": "must be >= 0",
		})
	}
	return nil
}

// WorkorderTimeLog records one continuous work interval on a work order.
type WorkorderTimeLog struct {
	ID          int64      `json:"id"`
	WorkorderID int64      `json:"workorder_id"`
	UserID      int64      `json:"user_id"`
	DateStart   time.Time  `json:"date_start"`
	DateEnd     *time.Time `json:"date_end,omitempty"`
	Duration    float64    `json:"duration"`
	LossID      *int64     `json:"loss_id,omitempty"`
}

// Start begins a work interval.
func (w *Workorder) Start() error {
	if w.State != WorkorderStateReady && w.State != WorkorderStatePaused {
		return platformerrors.Validation("work order cannot start", map[string]string{"state": "must be ready or paused"})
	}
	now := time.Now()
	w.State = WorkorderStateProgress
	if w.DateStart == nil {
		w.DateStart = &now
	}
	w.TimeLogs = append(w.TimeLogs, WorkorderTimeLog{WorkorderID: w.ID, DateStart: now})
	return nil
}

// Pause closes the active work interval without finishing the work order.
func (w *Workorder) Pause() error {
	if w.State != WorkorderStateProgress {
		return platformerrors.Validation("work order cannot pause", map[string]string{"state": "must be progress"})
	}
	if err := w.closeActiveTimeLog(time.Now()); err != nil {
		return err
	}
	w.State = WorkorderStatePaused
	return nil
}

// Resume starts a new work interval after a pause.
func (w *Workorder) Resume() error {
	if w.State != WorkorderStatePaused {
		return platformerrors.Validation("work order cannot resume", map[string]string{"state": "must be paused"})
	}
	return w.Start()
}

// Finish closes the active interval, records produced quantity, and completes the work order.
func (w *Workorder) Finish(qtyProduced float64) error {
	if w.State != WorkorderStateProgress && w.State != WorkorderStatePaused {
		return platformerrors.Validation("work order cannot finish", map[string]string{"state": "must be progress or paused"})
	}
	if qtyProduced < 0 {
		return platformerrors.Validation("produced quantity cannot be negative", map[string]string{"qty_produced": "must be >= 0"})
	}
	if w.State == WorkorderStateProgress {
		if err := w.closeActiveTimeLog(time.Now()); err != nil {
			return err
		}
	}
	now := time.Now()
	w.QtyProduced = qtyProduced
	w.DateFinished = &now
	w.State = WorkorderStateDone
	return nil
}

func (w *Workorder) closeActiveTimeLog(end time.Time) error {
	if len(w.TimeLogs) == 0 || w.TimeLogs[len(w.TimeLogs)-1].DateEnd != nil {
		return platformerrors.Validation("work order has no active time log", nil)
	}
	log := &w.TimeLogs[len(w.TimeLogs)-1]
	if end.Before(log.DateStart) {
		return platformerrors.Validation("time log end cannot precede start", nil)
	}
	log.DateEnd = &end
	log.Duration = end.Sub(log.DateStart).Minutes()
	w.Duration += log.Duration
	return nil
}
