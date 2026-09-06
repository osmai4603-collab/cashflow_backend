package hrstorage

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/hr"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo provides a thread-safe in-memory implementation of hr.Repository.
type MemoryRepo struct {
	mu             sync.RWMutex
	departments    map[int64]*hr.Department
	jobs           map[int64]*hr.Job
	employees      map[int64]*hr.Employee
	allocations    map[int64]*hr.LeaveAllocation
	leaveRequests  map[int64]*hr.LeaveRequest
	lastDeptID     int64
	lastJobID      int64
	lastEmpID      int64
	lastAllocID    int64
	lastLeaveReqID int64
}

// NewMemoryRepo creates an initialized MemoryRepo with standard initial departments and jobs.
func NewMemoryRepo() *MemoryRepo {
	repo := &MemoryRepo{
		departments:   make(map[int64]*hr.Department),
		jobs:          make(map[int64]*hr.Job),
		employees:     make(map[int64]*hr.Employee),
		allocations:   make(map[int64]*hr.LeaveAllocation),
		leaveRequests: make(map[int64]*hr.LeaveRequest),
	}

	now := time.Now().UTC()

	// Seed Standard Departments
	initDepts := []hr.Department{
		{ID: 1, Name: "Management", CompleteName: "Management", Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Finance & Accounting", CompleteName: "Finance & Accounting", Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "Human Resources", CompleteName: "Human Resources", Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "Information Technology", CompleteName: "Information Technology", Active: true, CreatedAt: now, UpdatedAt: now},
	}
	for _, d := range initDepts {
		clone := d
		repo.departments[d.ID] = &clone
		if d.ID > repo.lastDeptID {
			repo.lastDeptID = d.ID
		}
	}

	// Seed Standard Jobs
	initJobs := []hr.Job{
		{ID: 1, Name: "General Manager", ExpectedEmployees: 1, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Senior Accountant", DepartmentID: func() *int64 { id := int64(2); return &id }(), ExpectedEmployees: 2, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "HR Officer", DepartmentID: func() *int64 { id := int64(3); return &id }(), ExpectedEmployees: 2, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "Software Engineer", DepartmentID: func() *int64 { id := int64(4); return &id }(), ExpectedEmployees: 5, Active: true, CreatedAt: now, UpdatedAt: now},
	}
	for _, j := range initJobs {
		clone := j
		repo.jobs[j.ID] = &clone
		if j.ID > repo.lastJobID {
			repo.lastJobID = j.ID
		}
	}

	return repo
}

// ─────────────────────────────────────────────────────────────────────────────
// Departments CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateDepartment(ctx context.Context, dept *hr.Department) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastDeptID++
	dept.ID = r.lastDeptID
	now := time.Now().UTC()
	dept.CreatedAt = now
	dept.UpdatedAt = now

	// Update complete name if parent exists
	if dept.ParentID != nil && *dept.ParentID > 0 {
		if parent, ok := r.departments[*dept.ParentID]; ok {
			dept.CompleteName = fmt.Sprintf("%s / %s", parent.CompleteName, dept.Name)
		}
	}
	if dept.CompleteName == "" {
		dept.CompleteName = dept.Name
	}

	clone := *dept
	r.departments[dept.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetDepartmentByID(ctx context.Context, id int64) (*hr.Department, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dept, ok := r.departments[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("department with ID %d not found", id))
	}
	clone := *dept
	return &clone, nil
}

func (r *MemoryRepo) UpdateDepartment(ctx context.Context, dept *hr.Department) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.departments[dept.ID]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("department with ID %d not found", dept.ID))
	}

	dept.CreatedAt = existing.CreatedAt
	dept.UpdatedAt = time.Now().UTC()
	if dept.ParentID != nil && *dept.ParentID > 0 {
		if parent, ok := r.departments[*dept.ParentID]; ok {
			dept.CompleteName = fmt.Sprintf("%s / %s", parent.CompleteName, dept.Name)
		}
	}
	if dept.CompleteName == "" {
		dept.CompleteName = dept.Name
	}

	clone := *dept
	r.departments[dept.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteDepartment(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.departments[id]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("department with ID %d not found", id))
	}
	delete(r.departments, id)
	return nil
}

func (r *MemoryRepo) ListDepartments(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.Department], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []hr.Department
	for _, d := range r.departments {
		matches := true
		if f != nil && len(f.Criteria) > 0 {
			for _, crit := range f.Criteria {
				valStr := fmt.Sprintf("%v", crit.Value)
				switch crit.Field {
				case "name":
					if !strings.Contains(strings.ToLower(d.Name), strings.ToLower(valStr)) {
						matches = false
					}
				case "active":
					isActive := valStr == "true"
					if d.Active != isActive {
						matches = false
					}
				}
			}
		}
		if matches {
			result = append(result, *d)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	total := int64(len(result))
	offset := page.Offset()
	limit := page.LimitClamped()

	if offset >= int(total) {
		return pagination.NewPageResult([]hr.Department{}, total, page), nil
	}

	end := offset + limit
	if end > int(total) {
		end = int(total)
	}

	return pagination.NewPageResult(result[offset:end], total, page), nil
}

func (r *MemoryRepo) GetSubDepartments(ctx context.Context, parentID *int64) ([]hr.Department, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []hr.Department
	for _, d := range r.departments {
		if parentID == nil {
			if d.ParentID == nil || *d.ParentID == 0 {
				result = append(result, *d)
			}
		} else {
			if d.ParentID != nil && *d.ParentID == *parentID {
				result = append(result, *d)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (r *MemoryRepo) CountEmployeesByDepartment(ctx context.Context, deptID int64) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, emp := range r.employees {
		if emp.DepartmentID != nil && *emp.DepartmentID == deptID && emp.Active {
			count++
		}
	}
	return count, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Jobs CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateJob(ctx context.Context, job *hr.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastJobID++
	job.ID = r.lastJobID
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now

	clone := *job
	r.jobs[job.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetJobByID(ctx context.Context, id int64) (*hr.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, ok := r.jobs[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("job with ID %d not found", id))
	}
	clone := *job
	return &clone, nil
}

func (r *MemoryRepo) UpdateJob(ctx context.Context, job *hr.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.jobs[job.ID]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("job with ID %d not found", job.ID))
	}

	job.CreatedAt = existing.CreatedAt
	job.UpdatedAt = time.Now().UTC()

	clone := *job
	r.jobs[job.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteJob(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.jobs[id]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("job with ID %d not found", id))
	}
	delete(r.jobs, id)
	return nil
}

func (r *MemoryRepo) ListJobs(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.Job], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []hr.Job
	for _, j := range r.jobs {
		matches := true
		if f != nil && len(f.Criteria) > 0 {
			for _, crit := range f.Criteria {
				valStr := fmt.Sprintf("%v", crit.Value)
				switch crit.Field {
				case "name":
					if !strings.Contains(strings.ToLower(j.Name), strings.ToLower(valStr)) {
						matches = false
					}
				case "department_id":
					if j.DepartmentID == nil || fmt.Sprintf("%d", *j.DepartmentID) != valStr {
						matches = false
					}
				case "active":
					isActive := valStr == "true"
					if j.Active != isActive {
						matches = false
					}
				}
			}
		}
		if matches {
			result = append(result, *j)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	total := int64(len(result))
	offset := page.Offset()
	limit := page.LimitClamped()

	if offset >= int(total) {
		return pagination.NewPageResult([]hr.Job{}, total, page), nil
	}

	end := offset + limit
	if end > int(total) {
		end = int(total)
	}

	return pagination.NewPageResult(result[offset:end], total, page), nil
}

func (r *MemoryRepo) CountEmployeesByJob(ctx context.Context, jobID int64) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, emp := range r.employees {
		if emp.JobID != nil && *emp.JobID == jobID && emp.Active {
			count++
		}
	}
	return count, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Employees CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateEmployee(ctx context.Context, emp *hr.Employee) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastEmpID++
	emp.ID = r.lastEmpID
	now := time.Now().UTC()
	emp.CreatedAt = now
	emp.UpdatedAt = now

	clone := *emp
	r.employees[emp.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetEmployeeByID(ctx context.Context, id int64) (*hr.Employee, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	emp, ok := r.employees[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("employee with ID %d not found", id))
	}
	clone := *emp
	return &clone, nil
}

func (r *MemoryRepo) GetEmployeeByPartnerID(ctx context.Context, partnerID int64) (*hr.Employee, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, emp := range r.employees {
		if emp.PartnerID != nil && *emp.PartnerID == partnerID {
			clone := *emp
			return &clone, nil
		}
	}
	return nil, platformerrors.NotFound(fmt.Sprintf("employee with partner ID %d not found", partnerID))
}

func (r *MemoryRepo) UpdateEmployee(ctx context.Context, emp *hr.Employee) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.employees[emp.ID]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("employee with ID %d not found", emp.ID))
	}

	emp.CreatedAt = existing.CreatedAt
	emp.UpdatedAt = time.Now().UTC()

	clone := *emp
	r.employees[emp.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteEmployee(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.employees[id]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("employee with ID %d not found", id))
	}
	delete(r.employees, id)
	return nil
}

func (r *MemoryRepo) ListEmployees(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.Employee], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []hr.Employee
	for _, e := range r.employees {
		matches := true
		if f != nil && len(f.Criteria) > 0 {
			for _, crit := range f.Criteria {
				valStr := fmt.Sprintf("%v", crit.Value)
				switch crit.Field {
				case "name":
					if !strings.Contains(strings.ToLower(e.Name), strings.ToLower(valStr)) {
						matches = false
					}
				case "department_id":
					if e.DepartmentID == nil || fmt.Sprintf("%d", *e.DepartmentID) != valStr {
						matches = false
					}
				case "job_id":
					if e.JobID == nil || fmt.Sprintf("%d", *e.JobID) != valStr {
						matches = false
					}
				case "manager_id":
					if e.ManagerID == nil || fmt.Sprintf("%d", *e.ManagerID) != valStr {
						matches = false
					}
				case "active":
					isActive := valStr == "true"
					if e.Active != isActive {
						matches = false
					}
				}
			}
		}
		if matches {
			result = append(result, *e)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	total := int64(len(result))
	offset := page.Offset()
	limit := page.LimitClamped()

	if offset >= int(total) {
		return pagination.NewPageResult([]hr.Employee{}, total, page), nil
	}

	end := offset + limit
	if end > int(total) {
		end = int(total)
	}

	return pagination.NewPageResult(result[offset:end], total, page), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Leave Allocations CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateAllocation(ctx context.Context, alloc *hr.LeaveAllocation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastAllocID++
	alloc.ID = r.lastAllocID
	now := time.Now().UTC()
	alloc.CreatedAt = now
	alloc.UpdatedAt = now

	clone := *alloc
	r.allocations[alloc.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetAllocationByID(ctx context.Context, id int64) (*hr.LeaveAllocation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	alloc, ok := r.allocations[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("allocation with ID %d not found", id))
	}
	clone := *alloc
	return &clone, nil
}

func (r *MemoryRepo) UpdateAllocation(ctx context.Context, alloc *hr.LeaveAllocation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.allocations[alloc.ID]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("allocation with ID %d not found", alloc.ID))
	}

	alloc.CreatedAt = existing.CreatedAt
	alloc.UpdatedAt = time.Now().UTC()

	clone := *alloc
	r.allocations[alloc.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteAllocation(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.allocations[id]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("allocation with ID %d not found", id))
	}
	delete(r.allocations, id)
	return nil
}

func (r *MemoryRepo) ListAllocations(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.LeaveAllocation], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []hr.LeaveAllocation
	for _, a := range r.allocations {
		matches := true
		if f != nil && len(f.Criteria) > 0 {
			for _, crit := range f.Criteria {
				valStr := fmt.Sprintf("%v", crit.Value)
				switch crit.Field {
				case "employee_id":
					if fmt.Sprintf("%d", a.EmployeeID) != valStr {
						matches = false
					}
				case "leave_type":
					if a.LeaveType != valStr {
						matches = false
					}
				case "year":
					if fmt.Sprintf("%d", a.Year) != valStr {
						matches = false
					}
				}
			}
		}
		if matches {
			result = append(result, *a)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	total := int64(len(result))
	offset := page.Offset()
	limit := page.LimitClamped()

	if offset >= int(total) {
		return pagination.NewPageResult([]hr.LeaveAllocation{}, total, page), nil
	}

	end := offset + limit
	if end > int(total) {
		end = int(total)
	}

	return pagination.NewPageResult(result[offset:end], total, page), nil
}

func (r *MemoryRepo) GetAllocationsByEmployee(ctx context.Context, employeeID int64, year int) ([]hr.LeaveAllocation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []hr.LeaveAllocation
	for _, a := range r.allocations {
		if a.EmployeeID == employeeID && (year == 0 || a.Year == year) && a.State == hr.AllocationStateApproved {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (r *MemoryRepo) GetTotalAllocatedDays(ctx context.Context, employeeID int64, leaveType string, year int) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var total float64
	for _, a := range r.allocations {
		if a.EmployeeID == employeeID && a.LeaveType == leaveType && (year == 0 || a.Year == year) && a.State == hr.AllocationStateApproved {
			total += a.AllocatedDays
		}
	}
	return total, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Leave Requests CRUD & Queries
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateLeaveRequest(ctx context.Context, req *hr.LeaveRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastLeaveReqID++
	req.ID = r.lastLeaveReqID
	now := time.Now().UTC()
	req.CreatedAt = now
	req.UpdatedAt = now

	clone := *req
	r.leaveRequests[req.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetLeaveRequestByID(ctx context.Context, id int64) (*hr.LeaveRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	req, ok := r.leaveRequests[id]
	if !ok {
		return nil, platformerrors.NotFound(fmt.Sprintf("leave request with ID %d not found", id))
	}
	clone := *req
	return &clone, nil
}

func (r *MemoryRepo) UpdateLeaveRequest(ctx context.Context, req *hr.LeaveRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.leaveRequests[req.ID]
	if !ok {
		return platformerrors.NotFound(fmt.Sprintf("leave request with ID %d not found", req.ID))
	}

	req.CreatedAt = existing.CreatedAt
	req.UpdatedAt = time.Now().UTC()

	clone := *req
	r.leaveRequests[req.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteLeaveRequest(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.leaveRequests[id]; !ok {
		return platformerrors.NotFound(fmt.Sprintf("leave request with ID %d not found", id))
	}
	delete(r.leaveRequests, id)
	return nil
}

func (r *MemoryRepo) ListLeaveRequests(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[hr.LeaveRequest], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []hr.LeaveRequest
	for _, lr := range r.leaveRequests {
		matches := true
		if f != nil && len(f.Criteria) > 0 {
			for _, crit := range f.Criteria {
				valStr := fmt.Sprintf("%v", crit.Value)
				switch crit.Field {
				case "employee_id":
					if fmt.Sprintf("%d", lr.EmployeeID) != valStr {
						matches = false
					}
				case "leave_type":
					if lr.LeaveType != valStr {
						matches = false
					}
				case "state":
					if string(lr.State) != valStr {
						matches = false
					}
				}
			}
		}
		if matches {
			result = append(result, *lr)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	total := int64(len(result))
	offset := page.Offset()
	limit := page.LimitClamped()

	if offset >= int(total) {
		return pagination.NewPageResult([]hr.LeaveRequest{}, total, page), nil
	}

	end := offset + limit
	if end > int(total) {
		end = int(total)
	}

	return pagination.NewPageResult(result[offset:end], total, page), nil
}

func (r *MemoryRepo) GetApprovedLeaveDays(ctx context.Context, employeeID int64, leaveType string, year int) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var total float64
	for _, req := range r.leaveRequests {
		if req.EmployeeID == employeeID && req.LeaveType == leaveType && req.State == hr.LeaveStateValidate {
			if year == 0 || req.DateFrom.Year() == year {
				total += req.Days
			}
		}
	}
	return total, nil
}

func (r *MemoryRepo) GetPendingLeaveDays(ctx context.Context, employeeID int64, leaveType string, year int) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var total float64
	for _, req := range r.leaveRequests {
		if req.EmployeeID == employeeID && req.LeaveType == leaveType && req.State == hr.LeaveStateConfirm {
			if year == 0 || req.DateFrom.Year() == year {
				total += req.Days
			}
		}
	}
	return total, nil
}

func (r *MemoryRepo) HasOverlappingLeave(ctx context.Context, employeeID int64, from, to time.Time, excludeID int64) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Normalise to dates without hours
	fromDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toDay := time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.UTC)

	for _, req := range r.leaveRequests {
		if req.EmployeeID != employeeID || req.ID == excludeID {
			continue
		}
		// Only check active/pending requests
		if req.State != hr.LeaveStateConfirm && req.State != hr.LeaveStateValidate {
			continue
		}

		reqFrom := time.Date(req.DateFrom.Year(), req.DateFrom.Month(), req.DateFrom.Day(), 0, 0, 0, 0, time.UTC)
		reqTo := time.Date(req.DateTo.Year(), req.DateTo.Month(), req.DateTo.Day(), 23, 59, 59, 0, time.UTC)

		// Overlap check: !(reqTo < from || reqFrom > to)
		if !reqTo.Before(fromDay) && !reqFrom.After(toDay) {
			return true, nil
		}
	}
	return false, nil
}
