package hrhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"cashflow_backend/internal/domain/hr"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	hrusecase "cashflow_backend/internal/usecase/hr"

	"github.com/go-chi/chi/v5"
)

// Handler serves HTTP requests for the Human Resources (HR) domain.
type Handler struct {
	useCase           *hrusecase.UseCase
	attendanceUseCase hr.AttendanceUseCase
	logger            *slog.Logger
}

// NewHandler constructs a new HR HTTP Handler.
func NewHandler(useCase *hrusecase.UseCase, attendanceUseCase hr.AttendanceUseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase:           useCase,
		attendanceUseCase: attendanceUseCase,
		logger:            logger,
	}
}

func parseID(param string) (int64, error) {
	return strconv.ParseInt(param, 10, 64)
}

// ─────────────────────────────────────────────────────────────────────────────
// Departments
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	var req CreateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	dept, err := h.useCase.CreateDepartment(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToDepartmentResponse(dept))
}

func (h *Handler) GetDepartment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid department ID", err))
		return
	}

	dept, err := h.useCase.GetDepartment(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToDepartmentResponse(dept))
}

func (h *Handler) UpdateDepartment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid department ID", err))
		return
	}

	var req UpdateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	dept, err := h.useCase.UpdateDepartment(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToDepartmentResponse(dept))
}

func (h *Handler) DeleteDepartment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid department ID", err))
		return
	}

	if err := h.useCase.DeleteDepartment(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ListDepartments(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()
	if name := r.URL.Query().Get("name"); name != "" {
		f.Add("name", filter.OpILike, name)
	}
	if active := r.URL.Query().Get("active"); active != "" {
		if b, err := strconv.ParseBool(active); err == nil {
			f.Add("active", filter.OpEqual, b)
		}
	}

	res, err := h.useCase.ListDepartments(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]DepartmentResponse, len(res.Items))
	for i := range res.Items {
		dtos[i] = ToDepartmentResponse(&res.Items[i])
	}

	response.JSON(w, http.StatusOK, pagination.PageResult[DepartmentResponse]{
		Items:      dtos,
		TotalItems: res.TotalItems,
		Page:       res.Page,
		Limit:      res.Limit,
		TotalPages: res.TotalPages,
	})
}

func (h *Handler) GetDepartmentTree(w http.ResponseWriter, r *http.Request) {
	var parentID *int64
	if p := r.URL.Query().Get("parent_id"); p != "" {
		if id, err := parseID(p); err == nil {
			parentID = &id
		}
	}

	tree, err := h.useCase.GetDepartmentTree(r.Context(), parentID)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]DepartmentNodeResponse, len(tree))
	for i := range tree {
		dtos[i] = ToDepartmentNodeResponse(tree[i])
	}

	response.JSON(w, http.StatusOK, dtos)
}

// ─────────────────────────────────────────────────────────────────────────────
// Jobs
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	job, err := h.useCase.CreateJob(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToJobResponse(job))
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid job ID", err))
		return
	}

	job, err := h.useCase.GetJob(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToJobResponse(job))
}

func (h *Handler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid job ID", err))
		return
	}

	var req UpdateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	job, err := h.useCase.UpdateJob(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToJobResponse(job))
}

func (h *Handler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid job ID", err))
		return
	}

	if err := h.useCase.DeleteJob(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()
	if name := r.URL.Query().Get("name"); name != "" {
		f.Add("name", filter.OpILike, name)
	}
	if deptID := r.URL.Query().Get("department_id"); deptID != "" {
		if id, err := strconv.ParseInt(deptID, 10, 64); err == nil {
			f.Add("department_id", filter.OpEqual, id)
		}
	}
	if active := r.URL.Query().Get("active"); active != "" {
		if b, err := strconv.ParseBool(active); err == nil {
			f.Add("active", filter.OpEqual, b)
		}
	}

	res, err := h.useCase.ListJobs(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]JobResponse, len(res.Items))
	for i := range res.Items {
		dtos[i] = ToJobResponse(&res.Items[i])
	}

	response.JSON(w, http.StatusOK, pagination.PageResult[JobResponse]{
		Items:      dtos,
		TotalItems: res.TotalItems,
		Page:       res.Page,
		Limit:      res.Limit,
		TotalPages: res.TotalPages,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Employees
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var req CreateEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	emp, err := h.useCase.CreateEmployee(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToEmployeeResponse(emp))
}

func (h *Handler) GetEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid employee ID", err))
		return
	}

	emp, err := h.useCase.GetEmployee(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToEmployeeResponse(emp))
}

func (h *Handler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid employee ID", err))
		return
	}

	var req UpdateEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	emp, err := h.useCase.UpdateEmployee(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToEmployeeResponse(emp))
}

func (h *Handler) DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid employee ID", err))
		return
	}

	if err := h.useCase.DeleteEmployee(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()
	if name := r.URL.Query().Get("name"); name != "" {
		f.Add("name", filter.OpILike, name)
	}
	if deptID := r.URL.Query().Get("department_id"); deptID != "" {
		if id, err := strconv.ParseInt(deptID, 10, 64); err == nil {
			f.Add("department_id", filter.OpEqual, id)
		}
	}
	if jobID := r.URL.Query().Get("job_id"); jobID != "" {
		if id, err := strconv.ParseInt(jobID, 10, 64); err == nil {
			f.Add("job_id", filter.OpEqual, id)
		}
	}
	if managerID := r.URL.Query().Get("manager_id"); managerID != "" {
		if id, err := strconv.ParseInt(managerID, 10, 64); err == nil {
			f.Add("manager_id", filter.OpEqual, id)
		}
	}
	if active := r.URL.Query().Get("active"); active != "" {
		if b, err := strconv.ParseBool(active); err == nil {
			f.Add("active", filter.OpEqual, b)
		}
	}

	res, err := h.useCase.ListEmployees(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]EmployeeResponse, len(res.Items))
	for i := range res.Items {
		dtos[i] = ToEmployeeResponse(&res.Items[i])
	}

	response.JSON(w, http.StatusOK, pagination.PageResult[EmployeeResponse]{
		Items:      dtos,
		TotalItems: res.TotalItems,
		Page:       res.Page,
		Limit:      res.Limit,
		TotalPages: res.TotalPages,
	})
}

func (h *Handler) GetEmployeeLeaveBalance(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid employee ID", err))
		return
	}

	year := time.Now().Year()
	if yStr := r.URL.Query().Get("year"); yStr != "" {
		if y, err := strconv.Atoi(yStr); err == nil && y > 1900 {
			year = y
		}
	}

	summary, err := h.useCase.GetEmployeeLeaveBalance(r.Context(), id, year)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToEmployeeLeaveSummaryResponse(summary))
}

// ─────────────────────────────────────────────────────────────────────────────
// Leave Allocations
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateAllocation(w http.ResponseWriter, r *http.Request) {
	var req CreateAllocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	alloc, err := h.useCase.CreateAllocation(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToAllocationResponse(alloc))
}

func (h *Handler) GetAllocation(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid allocation ID", err))
		return
	}

	alloc, err := h.useCase.GetAllocation(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToAllocationResponse(alloc))
}

func (h *Handler) ListAllocations(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()
	if empID := r.URL.Query().Get("employee_id"); empID != "" {
		if id, err := strconv.ParseInt(empID, 10, 64); err == nil {
			f.Add("employee_id", filter.OpEqual, id)
		}
	}
	if lType := r.URL.Query().Get("leave_type"); lType != "" {
		f.Add("leave_type", filter.OpEqual, lType)
	}
	if year := r.URL.Query().Get("year"); year != "" {
		if y, err := strconv.Atoi(year); err == nil {
			f.Add("year", filter.OpEqual, y)
		}
	}

	res, err := h.useCase.ListAllocations(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]AllocationResponse, len(res.Items))
	for i := range res.Items {
		dtos[i] = ToAllocationResponse(&res.Items[i])
	}

	response.JSON(w, http.StatusOK, pagination.PageResult[AllocationResponse]{
		Items:      dtos,
		TotalItems: res.TotalItems,
		Page:       res.Page,
		Limit:      res.Limit,
		TotalPages: res.TotalPages,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Leave Requests
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateLeaveRequest(w http.ResponseWriter, r *http.Request) {
	var req CreateLeaveRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	leaveReq, err := h.useCase.CreateLeaveRequest(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToLeaveRequestResponse(leaveReq))
}

func (h *Handler) GetLeaveRequest(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid leave request ID", err))
		return
	}

	leaveReq, err := h.useCase.GetLeaveRequest(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLeaveRequestResponse(leaveReq))
}

func (h *Handler) UpdateLeaveRequest(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid leave request ID", err))
		return
	}

	var req UpdateLeaveRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	leaveReq, err := h.useCase.UpdateLeaveRequest(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLeaveRequestResponse(leaveReq))
}

func (h *Handler) DeleteLeaveRequest(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid leave request ID", err))
		return
	}

	if err := h.useCase.DeleteLeaveRequest(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ListLeaveRequests(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()
	if empID := r.URL.Query().Get("employee_id"); empID != "" {
		if id, err := strconv.ParseInt(empID, 10, 64); err == nil {
			f.Add("employee_id", filter.OpEqual, id)
		}
	}
	if lType := r.URL.Query().Get("leave_type"); lType != "" {
		f.Add("leave_type", filter.OpEqual, lType)
	}
	if state := r.URL.Query().Get("state"); state != "" {
		f.Add("state", filter.OpEqual, state)
	}

	res, err := h.useCase.ListLeaveRequests(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]LeaveRequestResponse, len(res.Items))
	for i := range res.Items {
		dtos[i] = ToLeaveRequestResponse(&res.Items[i])
	}

	response.JSON(w, http.StatusOK, pagination.PageResult[LeaveRequestResponse]{
		Items:      dtos,
		TotalItems: res.TotalItems,
		Page:       res.Page,
		Limit:      res.Limit,
		TotalPages: res.TotalPages,
	})
}

func (h *Handler) ConfirmLeaveRequest(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid leave request ID", err))
		return
	}

	leaveReq, err := h.useCase.ConfirmLeaveRequest(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLeaveRequestResponse(leaveReq))
}

func (h *Handler) ApproveLeaveRequest(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid leave request ID", err))
		return
	}

	var req ApproveLeaveRequestRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // approver_id optional

	leaveReq, err := h.useCase.ApproveLeaveRequest(r.Context(), id, req.ApproverID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLeaveRequestResponse(leaveReq))
}

func (h *Handler) RefuseLeaveRequest(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid leave request ID", err))
		return
	}

	var req RefuseLeaveRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	leaveReq, err := h.useCase.RefuseLeaveRequest(r.Context(), id, req.ApproverID, req.Reason)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLeaveRequestResponse(leaveReq))
}

func (h *Handler) CancelLeaveRequest(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid leave request ID", err))
		return
	}

	leaveReq, err := h.useCase.CancelLeaveRequest(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLeaveRequestResponse(leaveReq))
}
