package hrusecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cashflow_backend/internal/domain/hr"
	"cashflow_backend/internal/domain/partner"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// PartnerRepository abstracts partner management operations.
type PartnerRepository interface {
	GetByID(ctx context.Context, id int64) (*partner.Partner, error)
	Create(ctx context.Context, p *partner.Partner) error
}

// UseCase orchestrates business workflows for the Human Resources (HR) domain.
type UseCase struct {
	repo        hr.Repository
	partnerRepo PartnerRepository
	logger      *slog.Logger
}

// New constructs a new HR UseCase.
func New(repo hr.Repository, partnerRepo PartnerRepository, logger *slog.Logger) *UseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &UseCase{
		repo:        repo,
		partnerRepo: partnerRepo,
		logger:      logger,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Departments DTOs & Workflows
// ─────────────────────────────────────────────────────────────────────────────

type CreateDepartmentInput struct {
	Name      string `json:"name"`
	ParentID  *int64 `json:"parent_id"`
	ManagerID *int64 `json:"manager_id"`
	CompanyID *int64 `json:"company_id"`
	Color     int    `json:"color"`
}

type UpdateDepartmentInput struct {
	Name      *string `json:"name"`
	ParentID  *int64  `json:"parent_id"`
	ManagerID *int64  `json:"manager_id"`
	CompanyID *int64  `json:"company_id"`
	Color     *int    `json:"color"`
	Active    *bool   `json:"active"`
}

func (u *UseCase) CreateDepartment(ctx context.Context, in CreateDepartmentInput) (*hr.Department, error) {
	dept := &hr.Department{
		Name:      strings.TrimSpace(in.Name),
		ParentID:  in.ParentID,
		ManagerID: in.ManagerID,
		CompanyID: in.CompanyID,
		Color:     in.Color,
		Active:    true,
	}

	if err := dept.Validate(); err != nil {
		return nil, err
	}

	if dept.ParentID != nil && *dept.ParentID > 0 {
		parent, err := u.repo.GetDepartmentByID(ctx, *dept.ParentID)
		if err != nil {
			return nil, platformerrors.Validation("parent department does not exist", map[string]string{
				"parent_id": fmt.Sprintf("department %d not found", *dept.ParentID),
			})
		}
		dept.CompleteName = fmt.Sprintf("%s / %s", parent.CompleteName, dept.Name)
	} else {
		dept.CompleteName = dept.Name
	}

	if dept.ManagerID != nil && *dept.ManagerID > 0 {
		if _, err := u.repo.GetEmployeeByID(ctx, *dept.ManagerID); err != nil {
			return nil, platformerrors.Validation("manager employee does not exist", map[string]string{
				"manager_id": fmt.Sprintf("employee %d not found", *dept.ManagerID),
			})
		}
	}

	if err := u.repo.CreateDepartment(ctx, dept); err != nil {
		u.logger.Error("failed to create department", "error", err, "name", dept.Name)
		return nil, err
	}

	u.logger.Info("department created successfully", "id", dept.ID, "name", dept.Name)
	return dept, nil
}

func (u *UseCase) GetDepartment(ctx context.Context, id int64) (*hr.Department, error) {
	return u.repo.GetDepartmentByID(ctx, id)
}

func (u *UseCase) UpdateDepartment(ctx context.Context, id int64, in UpdateDepartmentInput) (*hr.Department, error) {
	dept, err := u.repo.GetDepartmentByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		dept.Name = strings.TrimSpace(*in.Name)
	}
	if in.ParentID != nil {
		if *in.ParentID == id {
			return nil, platformerrors.Validation("department cannot be its own parent", map[string]string{
				"parent_id": "self-referential hierarchy is invalid",
			})
		}
		dept.ParentID = in.ParentID
	}
	if in.ManagerID != nil {
		dept.ManagerID = in.ManagerID
	}
	if in.CompanyID != nil {
		dept.CompanyID = in.CompanyID
	}
	if in.Color != nil {
		dept.Color = *in.Color
	}
	if in.Active != nil {
		dept.Active = *in.Active
	}

	if err := dept.Validate(); err != nil {
		return nil, err
	}

	if dept.ParentID != nil && *dept.ParentID > 0 {
		parent, err := u.repo.GetDepartmentByID(ctx, *dept.ParentID)
		if err != nil {
			return nil, platformerrors.Validation("parent department does not exist", map[string]string{
				"parent_id": fmt.Sprintf("department %d not found", *dept.ParentID),
			})
		}
		dept.CompleteName = fmt.Sprintf("%s / %s", parent.CompleteName, dept.Name)
	} else {
		dept.CompleteName = dept.Name
	}

	if dept.ManagerID != nil && *dept.ManagerID > 0 {
		if _, err := u.repo.GetEmployeeByID(ctx, *dept.ManagerID); err != nil {
			return nil, platformerrors.Validation("manager employee does not exist", map[string]string{
				"manager_id": fmt.Sprintf("employee %d not found", *dept.ManagerID),
			})
		}
	}

	if err := u.repo.UpdateDepartment(ctx, dept); err != nil {
		return nil, err
	}

	return dept, nil
}

func (u *UseCase) DeleteDepartment(ctx context.Context, id int64) error {
	count, err := u.repo.CountEmployeesByDepartment(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return platformerrors.Conflict(fmt.Sprintf("cannot delete department with %d active employees", count), nil)
	}

	return u.repo.DeleteDepartment(ctx, id)
}

func (u *UseCase) ListDepartments(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.Department], error) {
	return u.repo.ListDepartments(ctx, f, page)
}

func (u *UseCase) GetDepartmentTree(ctx context.Context, parentID *int64) ([]hr.DepartmentNode, error) {
	depts, err := u.repo.GetSubDepartments(ctx, parentID)
	if err != nil {
		return nil, err
	}

	var nodes []hr.DepartmentNode
	for _, d := range depts {
		empCount, _ := u.repo.CountEmployeesByDepartment(ctx, d.ID)
		subID := d.ID
		children, err := u.GetDepartmentTree(ctx, &subID)
		if err != nil {
			children = nil
		}
		nodes = append(nodes, hr.DepartmentNode{
			Department:    d,
			Children:      children,
			EmployeeCount: empCount,
		})
	}
	return nodes, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Jobs DTOs & Workflows
// ─────────────────────────────────────────────────────────────────────────────

type CreateJobInput struct {
	Name              string `json:"name"`
	DepartmentID      *int64 `json:"department_id"`
	Description       string `json:"description"`
	ExpectedEmployees int    `json:"expected_employees"`
	CompanyID         *int64 `json:"company_id"`
}

type UpdateJobInput struct {
	Name              *string `json:"name"`
	DepartmentID      *int64  `json:"department_id"`
	Description       *string `json:"description"`
	ExpectedEmployees *int    `json:"expected_employees"`
	CompanyID         *int64  `json:"company_id"`
	Active            *bool   `json:"active"`
}

func (u *UseCase) CreateJob(ctx context.Context, in CreateJobInput) (*hr.Job, error) {
	job := &hr.Job{
		Name:              strings.TrimSpace(in.Name),
		DepartmentID:      in.DepartmentID,
		Description:       strings.TrimSpace(in.Description),
		ExpectedEmployees: in.ExpectedEmployees,
		CompanyID:         in.CompanyID,
		Active:            true,
	}

	if err := job.Validate(); err != nil {
		return nil, err
	}

	if job.DepartmentID != nil && *job.DepartmentID > 0 {
		if _, err := u.repo.GetDepartmentByID(ctx, *job.DepartmentID); err != nil {
			return nil, platformerrors.Validation("department does not exist", map[string]string{
				"department_id": fmt.Sprintf("department %d not found", *job.DepartmentID),
			})
		}
	}

	if err := u.repo.CreateJob(ctx, job); err != nil {
		return nil, err
	}

	return job, nil
}

func (u *UseCase) GetJob(ctx context.Context, id int64) (*hr.Job, error) {
	job, err := u.repo.GetJobByID(ctx, id)
	if err != nil {
		return nil, err
	}
	empCount, _ := u.repo.CountEmployeesByJob(ctx, job.ID)
	job.NoOfEmployee = empCount
	return job, nil
}

func (u *UseCase) UpdateJob(ctx context.Context, id int64, in UpdateJobInput) (*hr.Job, error) {
	job, err := u.repo.GetJobByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		job.Name = strings.TrimSpace(*in.Name)
	}
	if in.DepartmentID != nil {
		job.DepartmentID = in.DepartmentID
	}
	if in.Description != nil {
		job.Description = strings.TrimSpace(*in.Description)
	}
	if in.ExpectedEmployees != nil {
		job.ExpectedEmployees = *in.ExpectedEmployees
	}
	if in.CompanyID != nil {
		job.CompanyID = in.CompanyID
	}
	if in.Active != nil {
		job.Active = *in.Active
	}

	if err := job.Validate(); err != nil {
		return nil, err
	}

	if job.DepartmentID != nil && *job.DepartmentID > 0 {
		if _, err := u.repo.GetDepartmentByID(ctx, *job.DepartmentID); err != nil {
			return nil, platformerrors.Validation("department does not exist", map[string]string{
				"department_id": fmt.Sprintf("department %d not found", *job.DepartmentID),
			})
		}
	}

	if err := u.repo.UpdateJob(ctx, job); err != nil {
		return nil, err
	}

	empCount, _ := u.repo.CountEmployeesByJob(ctx, job.ID)
	job.NoOfEmployee = empCount
	return job, nil
}

func (u *UseCase) DeleteJob(ctx context.Context, id int64) error {
	count, err := u.repo.CountEmployeesByJob(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return platformerrors.Conflict(fmt.Sprintf("cannot delete job assigned to %d active employees", count), nil)
	}

	return u.repo.DeleteJob(ctx, id)
}

func (u *UseCase) ListJobs(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.Job], error) {
	res, err := u.repo.ListJobs(ctx, f, page)
	if err != nil {
		return res, err
	}
	for i := range res.Items {
		c, _ := u.repo.CountEmployeesByJob(ctx, res.Items[i].ID)
		res.Items[i].NoOfEmployee = c
	}
	return res, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Employees DTOs & Workflows
// ─────────────────────────────────────────────────────────────────────────────

type CreateEmployeeInput struct {
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

type UpdateEmployeeInput struct {
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

func (u *UseCase) CreateEmployee(ctx context.Context, in CreateEmployeeInput) (*hr.Employee, error) {
	emp := &hr.Employee{
		Name:             strings.TrimSpace(in.Name),
		PartnerID:        in.PartnerID,
		DepartmentID:     in.DepartmentID,
		JobID:            in.JobID,
		JobTitle:         strings.TrimSpace(in.JobTitle),
		ManagerID:        in.ManagerID,
		WorkEmail:        strings.TrimSpace(in.WorkEmail),
		WorkPhone:        strings.TrimSpace(in.WorkPhone),
		WorkLocation:     strings.TrimSpace(in.WorkLocation),
		HireDate:         in.HireDate,
		Gender:           in.Gender,
		MaritalStatus:    in.MaritalStatus,
		IdentificationID: strings.TrimSpace(in.IdentificationID),
		BankAccountNo:    strings.TrimSpace(in.BankAccountNo),
		CompanyID:        in.CompanyID,
		Active:           true,
	}

	if err := emp.Validate(); err != nil {
		return nil, err
	}

	// Validate or create linked partner
	if emp.PartnerID != nil && *emp.PartnerID > 0 {
		if u.partnerRepo != nil {
			if _, err := u.partnerRepo.GetByID(ctx, *emp.PartnerID); err != nil {
				return nil, platformerrors.Validation("linked partner not found", map[string]string{
					"partner_id": fmt.Sprintf("partner %d does not exist", *emp.PartnerID),
				})
			}
		}
	} else if in.AutoCreatePartner && u.partnerRepo != nil {
		newPartner := &partner.Partner{
			Name:       emp.Name,
			Email:      emp.WorkEmail,
			Phone:      emp.WorkPhone,
			Type:       partner.PartnerTypeIndividual,
			IsCustomer: false,
			IsSupplier: false,
			Active:     true,
		}
		if err := u.partnerRepo.Create(ctx, newPartner); err == nil {
			emp.PartnerID = &newPartner.ID
		}
	}

	// Validate Department
	if emp.DepartmentID != nil && *emp.DepartmentID > 0 {
		if _, err := u.repo.GetDepartmentByID(ctx, *emp.DepartmentID); err != nil {
			return nil, platformerrors.Validation("department not found", map[string]string{
				"department_id": fmt.Sprintf("department %d does not exist", *emp.DepartmentID),
			})
		}
	}

	// Validate Job
	if emp.JobID != nil && *emp.JobID > 0 {
		job, err := u.repo.GetJobByID(ctx, *emp.JobID)
		if err != nil {
			return nil, platformerrors.Validation("job position not found", map[string]string{
				"job_id": fmt.Sprintf("job %d does not exist", *emp.JobID),
			})
		}
		if emp.JobTitle == "" {
			emp.JobTitle = job.Name
		}
	}

	// Validate Manager
	if emp.ManagerID != nil && *emp.ManagerID > 0 {
		if _, err := u.repo.GetEmployeeByID(ctx, *emp.ManagerID); err != nil {
			return nil, platformerrors.Validation("manager not found", map[string]string{
				"manager_id": fmt.Sprintf("employee %d does not exist", *emp.ManagerID),
			})
		}
	}

	if err := u.repo.CreateEmployee(ctx, emp); err != nil {
		u.logger.Error("failed to create employee", "error", err, "name", emp.Name)
		return nil, err
	}

	u.logger.Info("employee created successfully", "id", emp.ID, "name", emp.Name)
	return emp, nil
}

func (u *UseCase) GetEmployee(ctx context.Context, id int64) (*hr.Employee, error) {
	return u.repo.GetEmployeeByID(ctx, id)
}

func (u *UseCase) UpdateEmployee(ctx context.Context, id int64, in UpdateEmployeeInput) (*hr.Employee, error) {
	emp, err := u.repo.GetEmployeeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		emp.Name = strings.TrimSpace(*in.Name)
	}
	if in.PartnerID != nil {
		emp.PartnerID = in.PartnerID
	}
	if in.DepartmentID != nil {
		emp.DepartmentID = in.DepartmentID
	}
	if in.JobID != nil {
		emp.JobID = in.JobID
	}
	if in.JobTitle != nil {
		emp.JobTitle = strings.TrimSpace(*in.JobTitle)
	}
	if in.ManagerID != nil {
		if *in.ManagerID == id {
			return nil, platformerrors.Validation("an employee cannot be their own manager", map[string]string{
				"manager_id": "invalid manager reference",
			})
		}
		emp.ManagerID = in.ManagerID
	}
	if in.WorkEmail != nil {
		emp.WorkEmail = strings.TrimSpace(*in.WorkEmail)
	}
	if in.WorkPhone != nil {
		emp.WorkPhone = strings.TrimSpace(*in.WorkPhone)
	}
	if in.WorkLocation != nil {
		emp.WorkLocation = strings.TrimSpace(*in.WorkLocation)
	}
	if in.HireDate != nil {
		emp.HireDate = in.HireDate
	}
	if in.Gender != nil {
		emp.Gender = *in.Gender
	}
	if in.MaritalStatus != nil {
		emp.MaritalStatus = *in.MaritalStatus
	}
	if in.IdentificationID != nil {
		emp.IdentificationID = strings.TrimSpace(*in.IdentificationID)
	}
	if in.BankAccountNo != nil {
		emp.BankAccountNo = strings.TrimSpace(*in.BankAccountNo)
	}
	if in.CompanyID != nil {
		emp.CompanyID = in.CompanyID
	}
	if in.Active != nil {
		emp.Active = *in.Active
	}

	if err := emp.Validate(); err != nil {
		return nil, err
	}

	// Validate Manager if updated
	if emp.ManagerID != nil && *emp.ManagerID > 0 {
		if _, err := u.repo.GetEmployeeByID(ctx, *emp.ManagerID); err != nil {
			return nil, platformerrors.Validation("manager not found", map[string]string{
				"manager_id": fmt.Sprintf("employee %d does not exist", *emp.ManagerID),
			})
		}
	}

	if err := u.repo.UpdateEmployee(ctx, emp); err != nil {
		return nil, err
	}

	return emp, nil
}

func (u *UseCase) DeleteEmployee(ctx context.Context, id int64) error {
	return u.repo.DeleteEmployee(ctx, id)
}

func (u *UseCase) ListEmployees(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.Employee], error) {
	return u.repo.ListEmployees(ctx, f, page)
}

// ─────────────────────────────────────────────────────────────────────────────
// Leave Allocations DTOs & Workflows
// ─────────────────────────────────────────────────────────────────────────────

type CreateAllocationInput struct {
	Name          string  `json:"name"`
	EmployeeID    int64   `json:"employee_id"`
	LeaveType     string  `json:"leave_type"`
	AllocatedDays float64 `json:"allocated_days"`
	Year          int     `json:"year"`
	Notes         string  `json:"notes"`
	CompanyID     *int64  `json:"company_id"`
}

func (u *UseCase) CreateAllocation(ctx context.Context, in CreateAllocationInput) (*hr.LeaveAllocation, error) {
	if _, err := u.repo.GetEmployeeByID(ctx, in.EmployeeID); err != nil {
		return nil, platformerrors.Validation("employee not found", map[string]string{
			"employee_id": fmt.Sprintf("employee %d does not exist", in.EmployeeID),
		})
	}

	alloc := &hr.LeaveAllocation{
		Name:          strings.TrimSpace(in.Name),
		EmployeeID:    in.EmployeeID,
		LeaveType:     strings.TrimSpace(in.LeaveType),
		AllocatedDays: in.AllocatedDays,
		Year:          in.Year,
		Notes:         strings.TrimSpace(in.Notes),
		CompanyID:     in.CompanyID,
		State:         hr.AllocationStateApproved,
	}

	if err := alloc.Validate(); err != nil {
		return nil, err
	}

	if err := u.repo.CreateAllocation(ctx, alloc); err != nil {
		return nil, err
	}

	u.logger.Info("leave allocation created", "employee_id", alloc.EmployeeID, "type", alloc.LeaveType, "days", alloc.AllocatedDays)
	return alloc, nil
}

func (u *UseCase) GetAllocation(ctx context.Context, id int64) (*hr.LeaveAllocation, error) {
	return u.repo.GetAllocationByID(ctx, id)
}

func (u *UseCase) ListAllocations(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.LeaveAllocation], error) {
	return u.repo.ListAllocations(ctx, f, page)
}

// ─────────────────────────────────────────────────────────────────────────────
// Leave Requests DTOs & Workflows
// ─────────────────────────────────────────────────────────────────────────────

type CreateLeaveRequestInput struct {
	EmployeeID  int64     `json:"employee_id"`
	LeaveType   string    `json:"leave_type"`
	DateFrom    time.Time `json:"date_from"`
	DateTo      time.Time `json:"date_to"`
	Days        *float64  `json:"days"`
	Description string    `json:"description"`
	AutoConfirm bool      `json:"auto_confirm"`
}

type UpdateLeaveRequestInput struct {
	LeaveType   *string    `json:"leave_type"`
	DateFrom    *time.Time `json:"date_from"`
	DateTo      *time.Time `json:"date_to"`
	Days        *float64   `json:"days"`
	Description *string    `json:"description"`
}

func (u *UseCase) CreateLeaveRequest(ctx context.Context, in CreateLeaveRequestInput) (*hr.LeaveRequest, error) {
	if _, err := u.repo.GetEmployeeByID(ctx, in.EmployeeID); err != nil {
		return nil, platformerrors.Validation("employee not found", map[string]string{
			"employee_id": fmt.Sprintf("employee %d does not exist", in.EmployeeID),
		})
	}

	days := 0.0
	if in.Days != nil && *in.Days > 0 {
		days = *in.Days
	} else {
		days = hr.CalculateDays(in.DateFrom, in.DateTo)
	}

	req := &hr.LeaveRequest{
		EmployeeID:  in.EmployeeID,
		LeaveType:   strings.TrimSpace(in.LeaveType),
		DateFrom:    in.DateFrom,
		DateTo:      in.DateTo,
		Days:        days,
		State:       hr.LeaveStateDraft,
		Description: strings.TrimSpace(in.Description),
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Check overlap with existing active leaves
	overlap, err := u.repo.HasOverlappingLeave(ctx, req.EmployeeID, req.DateFrom, req.DateTo, 0)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, platformerrors.Conflict("employee already has a confirmed or approved leave during this time frame", nil)
	}

	if in.AutoConfirm {
		if err := req.Confirm(); err != nil {
			return nil, err
		}
	}

	if err := u.repo.CreateLeaveRequest(ctx, req); err != nil {
		return nil, err
	}

	u.logger.Info("leave request created", "id", req.ID, "employee_id", req.EmployeeID, "days", req.Days, "state", req.State)
	return req, nil
}

func (u *UseCase) GetLeaveRequest(ctx context.Context, id int64) (*hr.LeaveRequest, error) {
	return u.repo.GetLeaveRequestByID(ctx, id)
}

func (u *UseCase) UpdateLeaveRequest(ctx context.Context, id int64, in UpdateLeaveRequestInput) (*hr.LeaveRequest, error) {
	req, err := u.repo.GetLeaveRequestByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.State != hr.LeaveStateDraft {
		return nil, platformerrors.Validation(fmt.Sprintf("cannot modify leave request in '%s' status; only 'draft' requests can be edited", req.State), nil)
	}

	if in.LeaveType != nil {
		req.LeaveType = strings.TrimSpace(*in.LeaveType)
	}
	if in.DateFrom != nil {
		req.DateFrom = *in.DateFrom
	}
	if in.DateTo != nil {
		req.DateTo = *in.DateTo
	}
	if in.Days != nil && *in.Days > 0 {
		req.Days = *in.Days
	} else if in.DateFrom != nil || in.DateTo != nil {
		req.Days = hr.CalculateDays(req.DateFrom, req.DateTo)
	}
	if in.Description != nil {
		req.Description = strings.TrimSpace(*in.Description)
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	overlap, err := u.repo.HasOverlappingLeave(ctx, req.EmployeeID, req.DateFrom, req.DateTo, req.ID)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, platformerrors.Conflict("overlapping leave detected for this time range", nil)
	}

	if err := u.repo.UpdateLeaveRequest(ctx, req); err != nil {
		return nil, err
	}

	return req, nil
}

func (u *UseCase) DeleteLeaveRequest(ctx context.Context, id int64) error {
	req, err := u.repo.GetLeaveRequestByID(ctx, id)
	if err != nil {
		return err
	}
	if req.State == hr.LeaveStateValidate {
		return platformerrors.Conflict("cannot delete an approved leave request; cancel it instead", nil)
	}
	return u.repo.DeleteLeaveRequest(ctx, id)
}

func (u *UseCase) ListLeaveRequests(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.LeaveRequest], error) {
	return u.repo.ListLeaveRequests(ctx, f, page)
}

func (u *UseCase) ConfirmLeaveRequest(ctx context.Context, id int64) (*hr.LeaveRequest, error) {
	req, err := u.repo.GetLeaveRequestByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := req.Confirm(); err != nil {
		return nil, err
	}

	if err := u.repo.UpdateLeaveRequest(ctx, req); err != nil {
		return nil, err
	}

	u.logger.Info("leave request submitted for approval", "id", req.ID, "employee_id", req.EmployeeID)
	return req, nil
}

func (u *UseCase) ApproveLeaveRequest(ctx context.Context, id int64, approverID int64) (*hr.LeaveRequest, error) {
	req, err := u.repo.GetLeaveRequestByID(ctx, id)
	if err != nil {
		return nil, err
	}

	year := req.DateFrom.Year()

	// If annual leave or leave requiring quota, verify sufficient balance
	if req.LeaveType == hr.LeaveTypeAnnual {
		allocated, err := u.repo.GetTotalAllocatedDays(ctx, req.EmployeeID, req.LeaveType, year)
		if err != nil {
			return nil, err
		}
		used, err := u.repo.GetApprovedLeaveDays(ctx, req.EmployeeID, req.LeaveType, year)
		if err != nil {
			return nil, err
		}
		remaining := allocated - used
		if remaining < req.Days {
			return nil, platformerrors.Validation(
				fmt.Sprintf("insufficient leave balance: requested %.1f days, but only %.1f days remaining for %d", req.Days, remaining, year),
				map[string]string{
					"days": fmt.Sprintf("exceeds available balance (%.1f)", remaining),
				},
			)
		}
	}

	// Double-check overlap
	overlap, err := u.repo.HasOverlappingLeave(ctx, req.EmployeeID, req.DateFrom, req.DateTo, req.ID)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, platformerrors.Conflict("cannot approve: employee has an overlapping confirmed or approved leave", nil)
	}

	if err := req.Approve(approverID); err != nil {
		return nil, err
	}

	if err := u.repo.UpdateLeaveRequest(ctx, req); err != nil {
		return nil, err
	}

	u.logger.Info("leave request approved successfully", "id", req.ID, "employee_id", req.EmployeeID, "days", req.Days)
	return req, nil
}

func (u *UseCase) RefuseLeaveRequest(ctx context.Context, id int64, approverID int64, reason string) (*hr.LeaveRequest, error) {
	req, err := u.repo.GetLeaveRequestByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := req.Refuse(approverID, reason); err != nil {
		return nil, err
	}

	if err := u.repo.UpdateLeaveRequest(ctx, req); err != nil {
		return nil, err
	}

	u.logger.Info("leave request refused", "id", req.ID, "reason", reason)
	return req, nil
}

func (u *UseCase) CancelLeaveRequest(ctx context.Context, id int64) (*hr.LeaveRequest, error) {
	req, err := u.repo.GetLeaveRequestByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := req.Cancel(); err != nil {
		return nil, err
	}

	if err := u.repo.UpdateLeaveRequest(ctx, req); err != nil {
		return nil, err
	}

	u.logger.Info("leave request cancelled", "id", req.ID)
	return req, nil
}

// GetEmployeeLeaveBalance calculates comprehensive leave balances across types for an employee in a given year.
func (u *UseCase) GetEmployeeLeaveBalance(ctx context.Context, employeeID int64, year int) (*hr.EmployeeLeaveSummary, error) {
	emp, err := u.repo.GetEmployeeByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}

	if year <= 1900 || year > 2200 {
		year = time.Now().Year()
	}

	leaveTypes := []string{
		hr.LeaveTypeAnnual,
		hr.LeaveTypeSick,
		hr.LeaveTypeUnpaid,
		hr.LeaveTypeEmergency,
	}

	// Also find any additional types from allocations
	allocs, err := u.repo.GetAllocationsByEmployee(ctx, employeeID, year)
	if err == nil {
		seen := make(map[string]bool)
		for _, lt := range leaveTypes {
			seen[lt] = true
		}
		for _, a := range allocs {
			if !seen[a.LeaveType] {
				leaveTypes = append(leaveTypes, a.LeaveType)
				seen[a.LeaveType] = true
			}
		}
	}

	var balances []hr.LeaveBalance
	for _, lt := range leaveTypes {
		allocated, _ := u.repo.GetTotalAllocatedDays(ctx, employeeID, lt, year)
		used, _ := u.repo.GetApprovedLeaveDays(ctx, employeeID, lt, year)
		pending, _ := u.repo.GetPendingLeaveDays(ctx, employeeID, lt, year)

		balances = append(balances, hr.LeaveBalance{
			EmployeeID:    employeeID,
			LeaveType:     lt,
			AllocatedDays: allocated,
			UsedDays:      used,
			PendingDays:   pending,
			RemainingDays: allocated - used,
		})
	}

	return &hr.EmployeeLeaveSummary{
		EmployeeID:   employeeID,
		EmployeeName: emp.Name,
		Year:         year,
		Balances:     balances,
	}, nil
}
