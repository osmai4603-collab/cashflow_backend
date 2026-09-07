package mrp

import (
	"time"

	"cashflow_backend/internal/platform/audit"
)

// WorkorderState defines the state of a specific operation in production.
type WorkorderState string

const (
	WorkorderStatePending  WorkorderState = "pending"
	WorkorderStateReady    WorkorderState = "ready"
	WorkorderStateProgress WorkorderState = "progress"
	WorkorderStateDone     WorkorderState = "done"
	WorkorderStateCancel   WorkorderState = "cancel"
)

// Workorder represents a manufacturing step (mrp.workorder in Odoo).
type Workorder struct {
	ID              int64          `json:"id"`
	ProductionID    int64          `json:"production_id"`
	WorkcenterID    int64          `json:"workcenter_id"`
	OperationID     int64          `json:"operation_id"`
	Name            string         `json:"name"`
	Sequence        int            `json:"sequence"`
	State           WorkorderState `json:"state"`

	DurationExpected float64       `json:"duration_expected"` // expected duration in minutes
	Duration         float64       `json:"duration"`          // real duration in minutes

	DateStart        *time.Time    `json:"date_start,omitempty"`
	DateFinished     *time.Time    `json:"date_finished,omitempty"`

	Audit            audit.Fields  `json:"audit"`
}
