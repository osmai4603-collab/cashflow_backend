package maintenance

import (
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// Maintenance types
const (
	MaintenanceTypeCorrective = "corrective"
	MaintenanceTypePreventive = "preventive"
)

// Priority levels
const (
	PriorityLow      = "0"
	PriorityMedium   = "1"
	PriorityHigh     = "2"
	PriorityVeryHigh = "3"
)

// AssignTo values
const (
	AssignToEmployee   = "employee"
	AssignToDepartment = "department"
	AssignToOther      = "other"
)

// Kanban states (Odoo 19.0 maintenance.request kanban_state)
const (
	KanbanStateNormal  = "normal"
	KanbanStateBlocked = "blocked"
	KanbanStateDone    = "done"
)

// RepeatUnit values for recurring preventive maintenance.
type RepeatUnit string

const (
	RepeatUnitDay   RepeatUnit = "day"
	RepeatUnitWeek  RepeatUnit = "week"
	RepeatUnitMonth RepeatUnit = "month"
	RepeatUnitYear  RepeatUnit = "year"
)

// RepeatType values control when a recurring maintenance chain stops.
type RepeatType string

const (
	RepeatTypeForever RepeatType = "forever"
	RepeatTypeUntil   RepeatType = "until"
)

// EquipmentCategory represents a category for maintenance equipment.
type EquipmentCategory struct {
	ID        int64        `json:"id"`
	Name      string       `json:"name"`
	Color     int          `json:"color"`
	Active    bool         `json:"active"`
	CompanyID int64        `json:"company_id"`
	Audit     audit.Fields `json:"audit"`
}

// EquipmentStage represents a stage in the maintenance process.
type EquipmentStage struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Sequence int    `json:"sequence"`
	Fold     bool   `json:"fold"`
	Done     bool   `json:"done"`
}

// Equipment represents a piece of equipment that requires maintenance.
type Equipment struct {
	ID               int64        `json:"id"`
	Name             string       `json:"name"`
	CategoryID       *int64       `json:"category_id,omitempty"`
	TeamID           *int64       `json:"team_id,omitempty"`
	TechnicianUserID *int64       `json:"technician_user_id,omitempty"`
	OwnerUserID      *int64       `json:"owner_user_id,omitempty"`
	EmployeeID       *int64       `json:"employee_id,omitempty"`
	DepartmentID     *int64       `json:"department_id,omitempty"`
	AssignTo         string       `json:"assign_to"` // employee, department, other
	PartnerID        *int64       `json:"partner_id,omitempty"`
	PartnerRef       string       `json:"partner_ref,omitempty"`
	LocationID       *int64       `json:"location_id,omitempty"`
	SerialNo         string       `json:"serial_no,omitempty"`
	Model            string       `json:"model,omitempty"`
	WarrantyDate     *time.Time   `json:"warranty_date,omitempty"`
	EffectiveDate    *time.Time   `json:"effective_date,omitempty"`
	NextActionDate   *time.Time   `json:"next_action_date,omitempty"`
	Period           int          `json:"period"` // Days
	Cost             float64      `json:"cost"`
	Notes            string       `json:"notes,omitempty"`
	AssignDate       *time.Time   `json:"assign_date,omitempty"`
	ScrapDate        *time.Time   `json:"scrap_date,omitempty"`
	Active           bool         `json:"active"`
	CompanyID        int64        `json:"company_id"`
	Audit            audit.Fields `json:"audit"`
}

// MaintenanceRequest represents a request for maintenance on a piece of equipment.
type MaintenanceRequest struct {
	ID                   int64        `json:"id"`
	Name                 string       `json:"name"`
	EquipmentID          *int64       `json:"equipment_id,omitempty"`
	TeamID               *int64       `json:"team_id,omitempty"`
	RequestDate          time.Time    `json:"request_date"`
	CloseDate            *time.Time   `json:"close_date,omitempty"`
	ScheduleDate         *time.Time   `json:"schedule_date,omitempty"`
	ScheduleEnd          *time.Time   `json:"schedule_end,omitempty"`
	MaintenanceType      string       `json:"maintenance_type"` // corrective, preventive
	Priority             string       `json:"priority"`         // 0, 1, 2, 3
	StageID              *int64       `json:"stage_id,omitempty"`
	KanbanState          string       `json:"kanban_state"` // normal, blocked, done
	TechnicianUserID     *int64       `json:"technician_user_id,omitempty"`
	OwnerUserID          *int64       `json:"owner_user_id,omitempty"`
	EmployeeID           *int64       `json:"employee_id,omitempty"`
	DepartmentID         *int64       `json:"department_id,omitempty"`
	Duration             float64      `json:"duration"`
	Description          string       `json:"description,omitempty"`
	RecurringMaintenance bool         `json:"recurring_maintenance"`
	RepeatInterval       int          `json:"repeat_interval"`
	RepeatUnit           RepeatUnit   `json:"repeat_unit"`
	RepeatType           RepeatType   `json:"repeat_type"`
	RepeatUntil          *time.Time   `json:"repeat_until,omitempty"`
	Archived             bool         `json:"archived"`
	CompanyID            int64        `json:"company_id"`
	Audit                audit.Fields `json:"audit"`
}

func (e *Equipment) Validate() error {
	e.Name = strings.TrimSpace(e.Name)
	if e.Name == "" {
		return platformerrors.Validation("equipment name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if e.AssignTo == "" {
		e.AssignTo = AssignToEmployee
	}
	return nil
}

func (r *MaintenanceRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return platformerrors.Validation("request name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	if r.MaintenanceType == "" {
		r.MaintenanceType = MaintenanceTypeCorrective
	}
	switch r.MaintenanceType {
	case MaintenanceTypeCorrective, MaintenanceTypePreventive:
	default:
		return platformerrors.Validation("invalid maintenance type", map[string]string{"maintenance_type": r.MaintenanceType})
	}
	if r.Priority == "" {
		r.Priority = PriorityLow
	}
	switch r.Priority {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityVeryHigh:
	default:
		return platformerrors.Validation("invalid priority", map[string]string{"priority": r.Priority})
	}
	if r.KanbanState == "" {
		r.KanbanState = KanbanStateNormal
	}
	switch r.KanbanState {
	case KanbanStateNormal, KanbanStateBlocked, KanbanStateDone:
	default:
		return platformerrors.Validation("invalid kanban state", map[string]string{"kanban_state": r.KanbanState})
	}
	if r.RepeatUnit == "" {
		r.RepeatUnit = RepeatUnitWeek
	}
	if r.RepeatType == "" {
		r.RepeatType = RepeatTypeForever
	}
	if r.RecurringMaintenance {
		if r.ScheduleDate == nil {
			return platformerrors.Validation("recurring maintenance requires a schedule date", nil)
		}
		if r.RepeatInterval < 1 {
			return platformerrors.Validation("repeat interval must be at least 1", nil)
		}
		switch r.RepeatUnit {
		case RepeatUnitDay, RepeatUnitWeek, RepeatUnitMonth, RepeatUnitYear:
		default:
			return platformerrors.Validation("invalid repeat unit", map[string]string{"repeat_unit": string(r.RepeatUnit)})
		}
		switch r.RepeatType {
		case RepeatTypeForever:
		case RepeatTypeUntil:
			if r.RepeatUntil == nil {
				return platformerrors.Validation("repeat type until requires a repeat until date", nil)
			}
		default:
			return platformerrors.Validation("invalid repeat type", map[string]string{"repeat_type": string(r.RepeatType)})
		}
	}
	return nil
}

// NextOccurrence computes the next schedule date for a recurring request.
// It returns nil when the chain is exhausted (repeat type "until" passed).
func (r *MaintenanceRequest) NextOccurrence() (*time.Time, error) {
	if !r.RecurringMaintenance || r.ScheduleDate == nil {
		return nil, nil
	}
	next := addRepeatInterval(*r.ScheduleDate, r.RepeatInterval, r.RepeatUnit)
	if r.RepeatType == RepeatTypeUntil && r.RepeatUntil != nil {
		until := time.Date(
			r.RepeatUntil.Year(), r.RepeatUntil.Month(), r.RepeatUntil.Day(),
			23, 59, 59, 0, time.UTC,
		)
		if next.After(until) {
			return nil, nil
		}
	}
	return &next, nil
}

// CurrentOccurrence computes the next schedule that has not been performed yet.
func (r *MaintenanceRequest) CurrentOccurrence() (time.Time, error) {
	if r.ScheduleDate != nil {
		return *r.ScheduleDate, nil
	}
	return time.Time{}, platformerrors.Validation("maintenance request has no schedule date", nil)
}

// addRepeatInterval advances the base schedule by repeatInterval of the given repeatUnit.
func addRepeatInterval(base time.Time, interval int, unit RepeatUnit) time.Time {
	switch unit {
	case RepeatUnitDay:
		return base.AddDate(0, 0, interval)
	case RepeatUnitWeek:
		return base.AddDate(0, 0, 7*interval)
	case RepeatUnitMonth:
		return base.AddDate(0, interval, 0)
	case RepeatUnitYear:
		return base.AddDate(interval, 0, 0)
	default:
		return base
	}
}
