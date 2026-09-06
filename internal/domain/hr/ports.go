package hr

import (
	"context"
	"time"

	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// Repository defines all persistent storage operations required by the HR domain.
type Repository interface {
	// Departments
	CreateDepartment(ctx context.Context, dept *Department) error
	GetDepartmentByID(ctx context.Context, id int64) (*Department, error)
	UpdateDepartment(ctx context.Context, dept *Department) error
	DeleteDepartment(ctx context.Context, id int64) error
	ListDepartments(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Department], error)
	GetSubDepartments(ctx context.Context, parentID *int64) ([]Department, error)
	CountEmployeesByDepartment(ctx context.Context, deptID int64) (int, error)

	// Jobs
	CreateJob(ctx context.Context, job *Job) error
	GetJobByID(ctx context.Context, id int64) (*Job, error)
	UpdateJob(ctx context.Context, job *Job) error
	DeleteJob(ctx context.Context, id int64) error
	ListJobs(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Job], error)
	CountEmployeesByJob(ctx context.Context, jobID int64) (int, error)

	// Employees
	CreateEmployee(ctx context.Context, emp *Employee) error
	GetEmployeeByID(ctx context.Context, id int64) (*Employee, error)
	GetEmployeeByPartnerID(ctx context.Context, partnerID int64) (*Employee, error)
	UpdateEmployee(ctx context.Context, emp *Employee) error
	DeleteEmployee(ctx context.Context, id int64) error
	ListEmployees(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[Employee], error)

	// Leave Allocations
	CreateAllocation(ctx context.Context, alloc *LeaveAllocation) error
	GetAllocationByID(ctx context.Context, id int64) (*LeaveAllocation, error)
	UpdateAllocation(ctx context.Context, alloc *LeaveAllocation) error
	DeleteAllocation(ctx context.Context, id int64) error
	ListAllocations(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[LeaveAllocation], error)
	GetAllocationsByEmployee(ctx context.Context, employeeID int64, year int) ([]LeaveAllocation, error)
	GetTotalAllocatedDays(ctx context.Context, employeeID int64, leaveType string, year int) (float64, error)

	// Leave Requests
	CreateLeaveRequest(ctx context.Context, req *LeaveRequest) error
	GetLeaveRequestByID(ctx context.Context, id int64) (*LeaveRequest, error)
	UpdateLeaveRequest(ctx context.Context, req *LeaveRequest) error
	DeleteLeaveRequest(ctx context.Context, id int64) error
	ListLeaveRequests(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[LeaveRequest], error)
	GetApprovedLeaveDays(ctx context.Context, employeeID int64, leaveType string, year int) (float64, error)
	GetPendingLeaveDays(ctx context.Context, employeeID int64, leaveType string, year int) (float64, error)
	HasOverlappingLeave(ctx context.Context, employeeID int64, from, to time.Time, excludeID int64) (bool, error)
}
