package mrp

import (
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// Workcenter represents a production unit where operations take place (mrp.workcenter in Odoo).
type Workcenter struct {
	ID            int64        `json:"id"`
	Name          string       `json:"name"`
	Code          string       `json:"code,omitempty"`
	Active        bool         `json:"active"`
	Sequence      int          `json:"sequence"`
	CompanyID     int64        `json:"company_id"`

	// Capacity and Timing
	TimeStart     float64      `json:"time_start"`      // Setup time (minutes)
	TimeStop      float64      `json:"time_stop"`       // Cleanup time (minutes)
	TimeEfficiency float64     `json:"time_efficiency"` // 100.0 = standard
	Capacity      float64      `json:"capacity"`        // Number of pieces that can be produced in parallel

	// Costing
	CostPerHour   float64      `json:"cost_per_hour"`

	Audit         audit.Fields `json:"audit"`
}

// Validate ensures Workcenter constraints are met.
func (w *Workcenter) Validate() error {
	w.Name = strings.TrimSpace(w.Name)
	if w.Name == "" {
		return platformerrors.Validation("workcenter name is required", map[string]string{
			"name": "cannot be empty",
		})
	}

	if w.TimeEfficiency <= 0 {
		w.TimeEfficiency = 100.0
	}

	if w.Capacity <= 0 {
		w.Capacity = 1.0
	}

	if w.CostPerHour < 0 {
		return platformerrors.Validation("cost per hour cannot be negative", map[string]string{
			"cost_per_hour": "must be >= 0",
		})
	}

	return nil
}

// OEE (Overall Equipment Effectiveness) - Placeholder for future metrics
type OEE struct {
	ID            int64     `json:"id"`
	WorkcenterID  int64     `json:"workcenter_id"`
	DateStart     time.Time `json:"date_start"`
	DateStop      time.Time `json:"date_stop"`
	Duration      float64   `json:"duration"` // minutes
	LossID        int64     `json:"loss_id"`   // reference to loss category
}
