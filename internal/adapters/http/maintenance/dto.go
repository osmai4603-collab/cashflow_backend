package maintenancehttp

import (
	"time"

	"cashflow_backend/internal/domain/maintenance"
	"cashflow_backend/internal/platform/i18n"
)

// CreateCategoryRequest is the request body for creating an equipment category.
type CreateCategoryRequest struct {
	Name  string `json:"name"`
	Color int    `json:"color"`
}

func (r CreateCategoryRequest) ToDomain(companyID int64) *maintenance.EquipmentCategory {
	return &maintenance.EquipmentCategory{
		Name:      i18n.NewTranslation(r.Name),
		Color:     r.Color,
		CompanyID: companyID,
	}
}

// UpdateCategoryRequest is the request body for updating an equipment category.
type UpdateCategoryRequest struct {
	Name  *string `json:"name"`
	Color *int    `json:"color"`
}

// CreateStageRequest is the request body for creating a stage.
type CreateStageRequest struct {
	Name     string `json:"name"`
	Sequence int    `json:"sequence"`
	Fold     bool   `json:"fold"`
	Done     bool   `json:"done"`
}

func (r CreateStageRequest) ToDomain() *maintenance.EquipmentStage {
	return &maintenance.EquipmentStage{
		Name:     i18n.NewTranslation(r.Name),
		Sequence: r.Sequence,
		Fold:     r.Fold,
		Done:     r.Done,
	}
}

// UpdateStageRequest is the request body for updating a stage.
type UpdateStageRequest struct {
	Name     *string `json:"name"`
	Sequence *int    `json:"sequence"`
	Fold     *bool   `json:"fold"`
	Done     *bool   `json:"done"`
}

// CreateTeamRequest is the request body for creating a maintenance team.
type CreateTeamRequest struct {
	Name      string  `json:"name"`
	MemberIDs []int64 `json:"member_ids"`
}

func (r CreateTeamRequest) ToDomain(companyID int64) (*maintenance.Team, []int64) {
	team := &maintenance.Team{
		Name:      i18n.NewTranslation(r.Name),
		CompanyID: &companyID,
	}
	return team, r.MemberIDs
}

// UpdateTeamRequest is the request body for updating a maintenance team.
type UpdateTeamRequest struct {
	Name      *string  `json:"name"`
	MemberIDs []int64  `json:"member_ids"`
}

// CreateEquipmentRequest is the request body for creating maintenance equipment.
type CreateEquipmentRequest struct {
	Name             string     `json:"name"`
	CategoryID       *int64     `json:"category_id"`
	TeamID           *int64     `json:"team_id"`
	TechnicianUserID *int64     `json:"technician_user_id"`
	OwnerUserID      *int64     `json:"owner_user_id"`
	EmployeeID       *int64     `json:"employee_id"`
	DepartmentID     *int64     `json:"department_id"`
	AssignTo         string     `json:"assign_to"`
	PartnerID        *int64     `json:"partner_id"`
	PartnerRef       string     `json:"partner_ref"`
	LocationID       *int64     `json:"location_id"`
	SerialNo         string     `json:"serial_no"`
	Model            string     `json:"model"`
	WarrantyDate     *time.Time `json:"warranty_date"`
	EffectiveDate    *time.Time `json:"effective_date"`
	NextActionDate   *time.Time `json:"next_action_date"`
	Period           int        `json:"period"`
	Cost             float64    `json:"cost"`
	Notes            string     `json:"notes"`
	AssignDate       *time.Time `json:"assign_date"`
	ScrapDate        *time.Time `json:"scrap_date"`
}

func (r CreateEquipmentRequest) ToDomain(companyID int64) *maintenance.Equipment {
	return &maintenance.Equipment{
		Name:             i18n.NewTranslation(r.Name),
		CategoryID:       r.CategoryID,
		TeamID:           r.TeamID,
		TechnicianUserID: r.TechnicianUserID,
		OwnerUserID:      r.OwnerUserID,
		EmployeeID:       r.EmployeeID,
		DepartmentID:     r.DepartmentID,
		AssignTo:         r.AssignTo,
		PartnerID:        r.PartnerID,
		PartnerRef:       r.PartnerRef,
		LocationID:       r.LocationID,
		SerialNo:         r.SerialNo,
		Model:            r.Model,
		WarrantyDate:     r.WarrantyDate,
		EffectiveDate:    r.EffectiveDate,
		NextActionDate:   r.NextActionDate,
		Period:           r.Period,
		Cost:             r.Cost,
		Notes:            r.Notes,
		AssignDate:       r.AssignDate,
		ScrapDate:        r.ScrapDate,
		CompanyID:        companyID,
	}
}

// UpdateEquipmentRequest holds optional fields for updating maintenance equipment.
type UpdateEquipmentRequest struct {
	Name             *string     `json:"name"`
	CategoryID       *int64      `json:"category_id"`
	TeamID           *int64      `json:"team_id"`
	TechnicianUserID *int64      `json:"technician_user_id"`
	OwnerUserID      *int64      `json:"owner_user_id"`
	EmployeeID       *int64      `json:"employee_id"`
	DepartmentID     *int64      `json:"department_id"`
	AssignTo         *string     `json:"assign_to"`
	PartnerID        *int64      `json:"partner_id"`
	PartnerRef       *string     `json:"partner_ref"`
	LocationID       *int64      `json:"location_id"`
	SerialNo         *string     `json:"serial_no"`
	Model            *string     `json:"model"`
	WarrantyDate     *time.Time  `json:"warranty_date"`
	EffectiveDate    *time.Time  `json:"effective_date"`
	NextActionDate   *time.Time  `json:"next_action_date"`
	Period           *int        `json:"period"`
	Cost             *float64    `json:"cost"`
	Notes            *string     `json:"notes"`
	AssignDate       *time.Time  `json:"assign_date"`
	ScrapDate        *time.Time  `json:"scrap_date"`
}

// Apply fills the domain entity with the non-nil request fields.
func (r UpdateEquipmentRequest) Apply(e *maintenance.Equipment) {
	if r.Name != nil {
		e.Name = i18n.NewTranslation(*r.Name)
	}
	if r.CategoryID != nil {
		e.CategoryID = r.CategoryID
	}
	if r.TeamID != nil {
		e.TeamID = r.TeamID
	}
	if r.TechnicianUserID != nil {
		e.TechnicianUserID = r.TechnicianUserID
	}
	if r.OwnerUserID != nil {
		e.OwnerUserID = r.OwnerUserID
	}
	if r.EmployeeID != nil {
		e.EmployeeID = r.EmployeeID
	}
	if r.DepartmentID != nil {
		e.DepartmentID = r.DepartmentID
	}
	if r.AssignTo != nil {
		e.AssignTo = *r.AssignTo
	}
	if r.PartnerID != nil {
		e.PartnerID = r.PartnerID
	}
	if r.PartnerRef != nil {
		e.PartnerRef = *r.PartnerRef
	}
	if r.LocationID != nil {
		e.LocationID = r.LocationID
	}
	if r.SerialNo != nil {
		e.SerialNo = *r.SerialNo
	}
	if r.Model != nil {
		e.Model = *r.Model
	}
	if r.WarrantyDate != nil {
		e.WarrantyDate = r.WarrantyDate
	}
	if r.EffectiveDate != nil {
		e.EffectiveDate = r.EffectiveDate
	}
	if r.NextActionDate != nil {
		e.NextActionDate = r.NextActionDate
	}
	if r.Period != nil {
		e.Period = *r.Period
	}
	if r.Cost != nil {
		e.Cost = *r.Cost
	}
	if r.Notes != nil {
		e.Notes = *r.Notes
	}
	if r.AssignDate != nil {
		e.AssignDate = r.AssignDate
	}
	if r.ScrapDate != nil {
		e.ScrapDate = r.ScrapDate
	}
}

// CreateRequestRequest is the request body for creating a maintenance request.
type CreateRequestRequest struct {
	Name                 string     `json:"name"`
	EquipmentID          *int64     `json:"equipment_id"`
	TeamID               *int64     `json:"team_id"`
	RequestDate          *time.Time `json:"request_date"`
	CloseDate            *time.Time `json:"close_date"`
	ScheduleDate         *time.Time `json:"schedule_date"`
	ScheduleEnd          *time.Time `json:"schedule_end"`
	MaintenanceType      string     `json:"maintenance_type"`
	Priority             string     `json:"priority"`
	StageID              *int64     `json:"stage_id"`
	KanbanState          string     `json:"kanban_state"`
	TechnicianUserID     *int64     `json:"technician_user_id"`
	OwnerUserID          *int64     `json:"owner_user_id"`
	EmployeeID           *int64     `json:"employee_id"`
	DepartmentID         *int64     `json:"department_id"`
	Duration             float64    `json:"duration"`
	Description          string     `json:"description"`
	RecurringMaintenance bool       `json:"recurring_maintenance"`
	RepeatInterval       int        `json:"repeat_interval"`
	RepeatUnit           string     `json:"repeat_unit"`
	RepeatType           string     `json:"repeat_type"`
	RepeatUntil          *time.Time `json:"repeat_until"`
}

func (r CreateRequestRequest) ToDomain(companyID int64) *maintenance.MaintenanceRequest {
	request := &maintenance.MaintenanceRequest{
		Name:                 i18n.NewTranslation(r.Name),
		EquipmentID:          r.EquipmentID,
		TeamID:               r.TeamID,
		MaintenanceType:      r.MaintenanceType,
		Priority:             r.Priority,
		StageID:              r.StageID,
		KanbanState:          r.KanbanState,
		TechnicianUserID:     r.TechnicianUserID,
		OwnerUserID:          r.OwnerUserID,
		EmployeeID:           r.EmployeeID,
		DepartmentID:         r.DepartmentID,
		Duration:             r.Duration,
		Description:          r.Description,
		RecurringMaintenance: r.RecurringMaintenance,
		RepeatInterval:       r.RepeatInterval,
		RepeatUnit:           maintenance.RepeatUnit(r.RepeatUnit),
		RepeatType:           maintenance.RepeatType(r.RepeatType),
		RepeatUntil:          r.RepeatUntil,
		CloseDate:            r.CloseDate,
		ScheduleDate:         r.ScheduleDate,
		ScheduleEnd:          r.ScheduleEnd,
		CompanyID:            companyID,
	}
	if r.RequestDate != nil {
		request.RequestDate = *r.RequestDate
	}
	if request.RepeatInterval <= 0 {
		request.RepeatInterval = 1
	}
	return request
}

// UpdateRequestRequest holds optional fields for updating a maintenance request.
type UpdateRequestRequest struct {
	Name                 *string     `json:"name"`
	EquipmentID          *int64      `json:"equipment_id"`
	TeamID               *int64      `json:"team_id"`
	RequestDate          *time.Time  `json:"request_date"`
	CloseDate            *time.Time  `json:"close_date"`
	ScheduleDate         *time.Time  `json:"schedule_date"`
	ScheduleEnd          *time.Time  `json:"schedule_end"`
	MaintenanceType      *string     `json:"maintenance_type"`
	Priority             *string     `json:"priority"`
	StageID              *int64      `json:"stage_id"`
	KanbanState          *string     `json:"kanban_state"`
	TechnicianUserID     *int64      `json:"technician_user_id"`
	OwnerUserID          *int64      `json:"owner_user_id"`
	EmployeeID           *int64      `json:"employee_id"`
	DepartmentID         *int64      `json:"department_id"`
	Duration             *float64    `json:"duration"`
	Description          *string     `json:"description"`
	RecurringMaintenance *bool       `json:"recurring_maintenance"`
	RepeatInterval       *int        `json:"repeat_interval"`
	RepeatUnit           *string     `json:"repeat_unit"`
	RepeatType           *string     `json:"repeat_type"`
	RepeatUntil          *time.Time  `json:"repeat_until"`
	Archived             *bool       `json:"archived"`
}

// Apply fills the domain entity with the non-nil request fields.
func (r UpdateRequestRequest) Apply(req *maintenance.MaintenanceRequest) {
	if r.Name != nil {
		req.Name = i18n.NewTranslation(*r.Name)
	}
	if r.EquipmentID != nil {
		req.EquipmentID = r.EquipmentID
	}
	if r.TeamID != nil {
		req.TeamID = r.TeamID
	}
	if r.RequestDate != nil {
		req.RequestDate = *r.RequestDate
	}
	if r.CloseDate != nil {
		req.CloseDate = r.CloseDate
	}
	if r.ScheduleDate != nil {
		req.ScheduleDate = r.ScheduleDate
	}
	if r.ScheduleEnd != nil {
		req.ScheduleEnd = r.ScheduleEnd
	}
	if r.MaintenanceType != nil {
		req.MaintenanceType = *r.MaintenanceType
	}
	if r.Priority != nil {
		req.Priority = *r.Priority
	}
	if r.StageID != nil {
		req.StageID = r.StageID
	}
	if r.KanbanState != nil {
		req.KanbanState = *r.KanbanState
	}
	if r.TechnicianUserID != nil {
		req.TechnicianUserID = r.TechnicianUserID
	}
	if r.OwnerUserID != nil {
		req.OwnerUserID = r.OwnerUserID
	}
	if r.EmployeeID != nil {
		req.EmployeeID = r.EmployeeID
	}
	if r.DepartmentID != nil {
		req.DepartmentID = r.DepartmentID
	}
	if r.Duration != nil {
		req.Duration = *r.Duration
	}
	if r.Description != nil {
		req.Description = *r.Description
	}
	if r.RecurringMaintenance != nil {
		req.RecurringMaintenance = *r.RecurringMaintenance
	}
	if r.RepeatInterval != nil {
		req.RepeatInterval = *r.RepeatInterval
	}
	if r.RepeatUnit != nil {
		req.RepeatUnit = maintenance.RepeatUnit(*r.RepeatUnit)
	}
	if r.RepeatType != nil {
		req.RepeatType = maintenance.RepeatType(*r.RepeatType)
	}
	if r.RepeatUntil != nil {
		req.RepeatUntil = r.RepeatUntil
	}
	if r.Archived != nil {
		req.Archived = *r.Archived
	}
}