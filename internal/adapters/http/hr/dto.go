package hrhttp

import (
	"time"

	"cashflow_backend/internal/domain/hr"
	hrusecase "cashflow_backend/internal/usecase/hr"
)

// ─────────────────────────────────────────────────────────────────────────────
// Departments
// ─────────────────────────────────────────────────────────────────────────────

type CreateDepartmentRequest struct {
	Name      string `json:"name"`
	ParentID  *int64 `json:"parent_id"`
	ManagerID *int64 `json:"manager_id"`
	CompanyID *int64 `json:"company_id"`
	Color     int    `json:"color"`
}

func (r CreateDepartmentRequest) ToInput() hrusecase.CreateDepartmentInput {
	return hrusecase.CreateDepartmentInput{
		Name:      r.Name,
		ParentID:  r.ParentID,
		ManagerID: r.ManagerID,
		CompanyID: r.CompanyID,
		Color:     r.Color,
	}
}

type UpdateDepartmentRequest struct {
	Name      *string `json:"name"`
	ParentID  *int64  `json:"parent_id"`
	ManagerID *int64  `json:"manager_id"`
	CompanyID *int64  `json:"company_id"`
	Color     *int    `json:"color"`
	Active    *bool   `json:"active"`
}

func (r UpdateDepartmentRequest) ToInput() hrusecase.UpdateDepartmentInput {
	return hrusecase.UpdateDepartmentInput{
		Name:      r.Name,
		ParentID:  r.ParentID,
		ManagerID: r.ManagerID,
		CompanyID: r.CompanyID,
		Color:     r.Color,
		Active:    r.Active,
	}
}

type DepartmentResponse struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	CompleteName string    `json:"complete_name"`
	ParentID     *int64    `json:"parent_id,omitempty"`
	ManagerID    *int64    `json:"manager_id,omitempty"`
	CompanyID    *int64    `json:"company_id,omitempty"`
	Color        int       `json:"color"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func ToDepartmentResponse(d *hr.Department) DepartmentResponse {
	return DepartmentResponse{
		ID:           d.ID,
		Name:         d.Name,
		CompleteName: d.CompleteName,
		ParentID:     d.ParentID,
		ManagerID:    d.ManagerID,
		CompanyID:    d.CompanyID,
		Color:        d.Color,
		Active:       d.Active,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

type DepartmentNodeResponse struct {
	DepartmentResponse
	Children      []DepartmentNodeResponse `json:"children,omitempty"`
	EmployeeCount int                      `json:"employee_count"`
}

func ToDepartmentNodeResponse(n hr.DepartmentNode) DepartmentNodeResponse {
	var children []DepartmentNodeResponse
	for _, c := range n.Children {
		children = append(children, ToDepartmentNodeResponse(c))
	}
	return DepartmentNodeResponse{
		DepartmentResponse: ToDepartmentResponse(&n.Department),
		Children:           children,
		EmployeeCount:      n.EmployeeCount,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Jobs
// ─────────────────────────────────────────────────────────────────────────────

type CreateJobRequest struct {
	Name              string `json:"name"`
	DepartmentID      *int64 `json:"department_id"`
	Description       string `json:"description"`
	ExpectedEmployees int    `json:"expected_employees"`
	CompanyID         *int64 `json:"company_id"`
}

func (r CreateJobRequest) ToInput() hrusecase.CreateJobInput {
	return hrusecase.CreateJobInput{
		Name:              r.Name,
		DepartmentID:      r.DepartmentID,
		Description:       r.Description,
		ExpectedEmployees: r.ExpectedEmployees,
		CompanyID:         r.CompanyID,
	}
}

type UpdateJobRequest struct {
	Name              *string `json:"name"`
	DepartmentID      *int64  `json:"department_id"`
	Description       *string `json:"description"`
	ExpectedEmployees *int    `json:"expected_employees"`
	CompanyID         *int64  `json:"company_id"`
	Active            *bool   `json:"active"`
}

func (r UpdateJobRequest) ToInput() hrusecase.UpdateJobInput {
	return hrusecase.UpdateJobInput{
		Name:              r.Name,
		DepartmentID:      r.DepartmentID,
		Description:       r.Description,
		ExpectedEmployees: r.ExpectedEmployees,
		CompanyID:         r.CompanyID,
		Active:            r.Active,
	}
}

type JobResponse struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	DepartmentID      *int64    `json:"department_id,omitempty"`
	Description       string    `json:"description,omitempty"`
	ExpectedEmployees int       `json:"expected_employees"`
	NoOfEmployee      int       `json:"no_of_employee"`
	CompanyID         *int64    `json:"company_id,omitempty"`
	Active            bool      `json:"active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func ToJobResponse(j *hr.Job) JobResponse {
	return JobResponse{
		ID:                j.ID,
		Name:              j.Name,
		DepartmentID:      j.DepartmentID,
		Description:       j.Description,
		ExpectedEmployees: j.ExpectedEmployees,
		NoOfEmployee:      j.NoOfEmployee,
		CompanyID:         j.CompanyID,
		Active:            j.Active,
		CreatedAt:         j.CreatedAt,
		UpdatedAt:         j.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Employees
// ─────────────────────────────────────────────────────────────────────────────

type CreateEmployeeRequest struct {
	Name              string     `json:"name"`
	PartnerID         *int64     `json:"partner_id"`
	DepartmentID      *int64     `json:"department_id"`
	JobID             *int64     `json:"job_id"`
	JobTitle          string     `json:"job_title"`
	ManagerID         *int64     `json:"manager_id"`
	WorkEmail         string     `json:"work_email"`
	WorkPhone         string     `json:"work_phone"`
	WorkLocation      string     `json:"work_location"`
	HireDate          *time.Time `json:"hire_date"`
	Gender            string     `json:"gender"`
	MaritalStatus     string     `json:"marital_status"`
	IdentificationID  string     `json:"identification_id"`
	BankAccountNo     string     `json:"bank_account_no"`
	CompanyID         *int64     `json:"company_id"`
	AutoCreatePartner bool       `json:"auto_create_partner"`
}

func (r CreateEmployeeRequest) ToInput() hrusecase.CreateEmployeeInput {
	return hrusecase.CreateEmployeeInput{
		Name:              r.Name,
		PartnerID:         r.PartnerID,
		DepartmentID:      r.DepartmentID,
		JobID:             r.JobID,
		JobTitle:          r.JobTitle,
		ManagerID:         r.ManagerID,
		WorkEmail:         r.WorkEmail,
		WorkPhone:         r.WorkPhone,
		WorkLocation:      r.WorkLocation,
		HireDate:          r.HireDate,
		Gender:            r.Gender,
		MaritalStatus:     r.MaritalStatus,
		IdentificationID:  r.IdentificationID,
		BankAccountNo:     r.BankAccountNo,
		CompanyID:         r.CompanyID,
		AutoCreatePartner: r.AutoCreatePartner,
	}
}

type UpdateEmployeeRequest struct {
	Name             *string    `json:"name"`
	PartnerID        *int64     `json:"partner_id"`
	DepartmentID     *int64     `json:"department_id"`
	JobID            *int64     `json:"job_id"`
	JobTitle         *string    `json:"job_title"`
	ManagerID        *int64     `json:"manager_id"`
	WorkEmail        *string    `json:"work_email"`
	WorkPhone        *string    `json:"work_phone"`
	WorkLocation     *string    `json:"work_location"`
	HireDate         *time.Time `json:"hire_date"`
	Gender           *string    `json:"gender"`
	MaritalStatus    *string    `json:"marital_status"`
	IdentificationID *string    `json:"identification_id"`
	BankAccountNo    *string    `json:"bank_account_no"`
	CompanyID        *int64     `json:"company_id"`
	Active           *bool      `json:"active"`
}

func (r UpdateEmployeeRequest) ToInput() hrusecase.UpdateEmployeeInput {
	return hrusecase.UpdateEmployeeInput{
		Name:             r.Name,
		PartnerID:        r.PartnerID,
		DepartmentID:     r.DepartmentID,
		JobID:            r.JobID,
		JobTitle:         r.JobTitle,
		ManagerID:        r.ManagerID,
		WorkEmail:        r.WorkEmail,
		WorkPhone:        r.WorkPhone,
		WorkLocation:     r.WorkLocation,
		HireDate:         r.HireDate,
		Gender:           r.Gender,
		MaritalStatus:    r.MaritalStatus,
		IdentificationID: r.IdentificationID,
		BankAccountNo:    r.BankAccountNo,
		CompanyID:        r.CompanyID,
		Active:           r.Active,
	}
}

type EmployeeResponse struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	PartnerID        *int64     `json:"partner_id,omitempty"`
	DepartmentID     *int64     `json:"department_id,omitempty"`
	JobID            *int64     `json:"job_id,omitempty"`
	JobTitle         string     `json:"job_title,omitempty"`
	ManagerID        *int64     `json:"manager_id,omitempty"`
	WorkEmail        string     `json:"work_email,omitempty"`
	WorkPhone        string     `json:"work_phone,omitempty"`
	WorkLocation     string     `json:"work_location,omitempty"`
	HireDate         *time.Time `json:"hire_date,omitempty"`
	Gender           string     `json:"gender,omitempty"`
	MaritalStatus    string     `json:"marital_status,omitempty"`
	IdentificationID string     `json:"identification_id,omitempty"`
	BankAccountNo    string     `json:"bank_account_no,omitempty"`
	CompanyID        *int64     `json:"company_id,omitempty"`
	Active           bool       `json:"active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func ToEmployeeResponse(e *hr.Employee) EmployeeResponse {
	return EmployeeResponse{
		ID:               e.ID,
		Name:             e.Name,
		PartnerID:        e.PartnerID,
		DepartmentID:     e.DepartmentID,
		JobID:            e.JobID,
		JobTitle:         e.JobTitle,
		ManagerID:        e.ManagerID,
		WorkEmail:        e.WorkEmail,
		WorkPhone:        e.WorkPhone,
		WorkLocation:     e.WorkLocation,
		HireDate:         e.HireDate,
		Gender:           e.Gender,
		MaritalStatus:    e.MaritalStatus,
		IdentificationID: e.IdentificationID,
		BankAccountNo:    e.BankAccountNo,
		CompanyID:        e.CompanyID,
		Active:           e.Active,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Leave Allocations
// ─────────────────────────────────────────────────────────────────────────────

type CreateAllocationRequest struct {
	Name          string  `json:"name"`
	EmployeeID    int64   `json:"employee_id"`
	LeaveType     string  `json:"leave_type"`
	AllocatedDays float64 `json:"allocated_days"`
	Year          int     `json:"year"`
	Notes         string  `json:"notes"`
	CompanyID     *int64  `json:"company_id"`
}

func (r CreateAllocationRequest) ToInput() hrusecase.CreateAllocationInput {
	return hrusecase.CreateAllocationInput{
		Name:          r.Name,
		EmployeeID:    r.EmployeeID,
		LeaveType:     r.LeaveType,
		AllocatedDays: r.AllocatedDays,
		Year:          r.Year,
		Notes:         r.Notes,
		CompanyID:     r.CompanyID,
	}
}

type AllocationResponse struct {
	ID            int64              `json:"id"`
	Name          string             `json:"name"`
	EmployeeID    int64              `json:"employee_id"`
	LeaveType     string             `json:"leave_type"`
	AllocatedDays float64            `json:"allocated_days"`
	Year          int                `json:"year"`
	State         hr.AllocationState `json:"state"`
	Notes         string             `json:"notes,omitempty"`
	CompanyID     *int64             `json:"company_id,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

func ToAllocationResponse(a *hr.LeaveAllocation) AllocationResponse {
	return AllocationResponse{
		ID:            a.ID,
		Name:          a.Name,
		EmployeeID:    a.EmployeeID,
		LeaveType:     a.LeaveType,
		AllocatedDays: a.AllocatedDays,
		Year:          a.Year,
		State:         a.State,
		Notes:         a.Notes,
		CompanyID:     a.CompanyID,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Leave Requests
// ─────────────────────────────────────────────────────────────────────────────

type CreateLeaveRequestRequest struct {
	EmployeeID  int64      `json:"employee_id"`
	LeaveType   string     `json:"leave_type"`
	DateFrom    time.Time  `json:"date_from"`
	DateTo      time.Time  `json:"date_to"`
	Days        *float64   `json:"days"`
	Description string     `json:"description"`
	AutoConfirm bool       `json:"auto_confirm"`
}

func (r CreateLeaveRequestRequest) ToInput() hrusecase.CreateLeaveRequestInput {
	return hrusecase.CreateLeaveRequestInput{
		EmployeeID:  r.EmployeeID,
		LeaveType:   r.LeaveType,
		DateFrom:    r.DateFrom,
		DateTo:      r.DateTo,
		Days:        r.Days,
		Description: r.Description,
		AutoConfirm: r.AutoConfirm,
	}
}

type UpdateLeaveRequestRequest struct {
	LeaveType   *string    `json:"leave_type"`
	DateFrom    *time.Time `json:"date_from"`
	DateTo      *time.Time `json:"date_to"`
	Days        *float64   `json:"days"`
	Description *string    `json:"description"`
}

func (r UpdateLeaveRequestRequest) ToInput() hrusecase.UpdateLeaveRequestInput {
	return hrusecase.UpdateLeaveRequestInput{
		LeaveType:   r.LeaveType,
		DateFrom:    r.DateFrom,
		DateTo:      r.DateTo,
		Days:        r.Days,
		Description: r.Description,
	}
}

type ApproveLeaveRequestRequest struct {
	ApproverID int64 `json:"approver_id"`
}

type RefuseLeaveRequestRequest struct {
	ApproverID int64  `json:"approver_id"`
	Reason      string `json:"reason"`
}

type LeaveRequestResponse struct {
	ID            int64         `json:"id"`
	Name          string        `json:"name"`
	EmployeeID    int64         `json:"employee_id"`
	LeaveType     string        `json:"leave_type"`
	DateFrom      time.Time     `json:"date_from"`
	DateTo        time.Time     `json:"date_to"`
	Days          float64       `json:"days"`
	State         hr.LeaveState `json:"state"`
	Description   string        `json:"description,omitempty"`
	ApproverID    *int64        `json:"approver_id,omitempty"`
	RefusalReason string        `json:"refusal_reason,omitempty"`
	CompanyID     *int64        `json:"company_id,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

func ToLeaveRequestResponse(r *hr.LeaveRequest) LeaveRequestResponse {
	return LeaveRequestResponse{
		ID:            r.ID,
		Name:          r.Name,
		EmployeeID:    r.EmployeeID,
		LeaveType:     r.LeaveType,
		DateFrom:      r.DateFrom,
		DateTo:        r.DateTo,
		Days:          r.Days,
		State:         r.State,
		Description:   r.Description,
		ApproverID:    r.ApproverID,
		RefusalReason: r.RefusalReason,
		CompanyID:     r.CompanyID,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Leave Balances
// ─────────────────────────────────────────────────────────────────────────────

type LeaveBalanceResponse struct {
	EmployeeID    int64   `json:"employee_id"`
	LeaveType     string  `json:"leave_type"`
	AllocatedDays float64 `json:"allocated_days"`
	UsedDays      float64 `json:"used_days"`
	PendingDays   float64 `json:"pending_days"`
	RemainingDays float64 `json:"remaining_days"`
}

type EmployeeLeaveSummaryResponse struct {
	EmployeeID   int64                  `json:"employee_id"`
	EmployeeName string                 `json:"employee_name"`
	Year         int                    `json:"year"`
	Balances     []LeaveBalanceResponse `json:"balances"`
}

func ToEmployeeLeaveSummaryResponse(s *hr.EmployeeLeaveSummary) EmployeeLeaveSummaryResponse {
	var balances []LeaveBalanceResponse
	for _, b := range s.Balances {
		balances = append(balances, LeaveBalanceResponse{
			EmployeeID:    b.EmployeeID,
			LeaveType:     b.LeaveType,
			AllocatedDays: b.AllocatedDays,
			UsedDays:      b.UsedDays,
			PendingDays:   b.PendingDays,
			RemainingDays: b.RemainingDays,
		})
	}
	return EmployeeLeaveSummaryResponse{
		EmployeeID:   s.EmployeeID,
		EmployeeName: s.EmployeeName,
		Year:         s.Year,
		Balances:     balances,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Attendance
// ─────────────────────────────────────────────────────────────────────────────

type CheckInRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	IPAddress string  `json:"ip_address"`
	Browser   string  `json:"browser"`
	Mode      string  `json:"mode"`
}

type CheckOutRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	IPAddress string  `json:"ip_address"`
	Browser   string  `json:"browser"`
	Mode      string  `json:"mode"`
}

type AttendanceResponse struct {
	ID             int64      `json:"id"`
	EmployeeID     int64      `json:"employee_id"`
	CheckIn        time.Time  `json:"check_in"`
	CheckOut       *time.Time `json:"check_out,omitempty"`
	WorkedHours    float64    `json:"worked_hours"`
	OvertimeHours  float64    `json:"overtime_hours"`
	OvertimeStatus string     `json:"overtime_status"`
	InMode         string     `json:"in_mode"`
	OutMode        string     `json:"out_mode"`
	CreatedAt      time.Time  `json:"created_at"`
}

func ToAttendanceResponse(a *hr.Attendance) AttendanceResponse {
	return AttendanceResponse{
		ID:             a.ID,
		EmployeeID:     a.EmployeeID,
		CheckIn:        a.CheckIn,
		CheckOut:       a.CheckOut,
		WorkedHours:    a.WorkedHours,
		OvertimeHours:  a.OvertimeHours,
		OvertimeStatus: a.OvertimeStatus,
		InMode:         a.InMode,
		OutMode:        a.OutMode,
		CreatedAt:      a.CreatedAt,
	}
}

type ApproveOvertimeRequest struct {
	ManagerID int64 `json:"manager_id"`
}
