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
	TechnicianUserID *int64       `json:"technician_user_id,omitempty"`
	OwnerUserID      *int64       `json:"owner_user_id,omitempty"`
	EmployeeID       *int64       `json:"employee_id,omitempty"`
	DepartmentID     *int64       `json:"department_id,omitempty"`
	AssignTo         string       `json:"assign_to"` // employee, department, other
	LocationID       *int64       `json:"location_id,omitempty"`
	SerialNo         string       `json:"serial_no,omitempty"`
	Model            string       `json:"model,omitempty"`
	WarrantyDate     *time.Time   `json:"warranty_date,omitempty"`
	EffectiveDate    *time.Time   `json:"effective_date,omitempty"`
	NextActionDate   *time.Time   `json:"next_action_date,omitempty"`
	Period           int          `json:"period"` // Days
	Active           bool         `json:"active"`
	CompanyID        int64        `json:"company_id"`
	Audit            audit.Fields `json:"audit"`
}

// MaintenanceRequest represents a request for maintenance on a piece of equipment.
type MaintenanceRequest struct {
	ID               int64        `json:"id"`
	Name             string       `json:"name"`
	EquipmentID      *int64       `json:"equipment_id,omitempty"`
	RequestDate      time.Time    `json:"request_date"`
	CloseDate        *time.Time   `json:"close_date,omitempty"`
	ScheduleDate     *time.Time   `json:"schedule_date,omitempty"`
	MaintenanceType  string       `json:"maintenance_type"` // corrective, preventive
	Priority         string       `json:"priority"`         // 0, 1, 2, 3
	StageID          *int64       `json:"stage_id,omitempty"`
	TechnicianUserID *int64       `json:"technician_user_id,omitempty"`
	OwnerUserID      *int64       `json:"owner_user_id,omitempty"`
	EmployeeID       *int64       `json:"employee_id,omitempty"`
	DepartmentID     *int64       `json:"department_id,omitempty"`
	Duration         float64      `json:"duration"`
	Description      string       `json:"description,omitempty"`
	CompanyID        int64        `json:"company_id"`
	Audit            audit.Fields `json:"audit"`
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
	if r.Priority == "" {
		r.Priority = PriorityLow
	}
	return nil
}
