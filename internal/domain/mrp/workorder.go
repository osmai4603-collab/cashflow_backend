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

	DateStart    *time.Time `json:"date_start,omitempty"`
	DateFinished *time.Time `json:"date_finished,omitempty"`

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
		WorkorderStateDone, WorkorderStateCancel:
	default:
		return platformerrors.Validation("invalid work order state", map[string]string{
			"state": "must be blocked, ready, progress, done or cancel",
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
