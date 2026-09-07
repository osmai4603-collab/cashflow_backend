package hrhttp

import (
	"cashflow_backend/internal/platform/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers all HR domain endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler, authorizers ...auth.Authorizer) {
	var authorizer auth.Authorizer
	if len(authorizers) > 0 {
		authorizer = authorizers[0]
	}
	access := func(r chi.Router, model string, action auth.Action) chi.Router {
		if authorizer == nil {
			return r
		}
		return r.With(auth.RequireAccess(authorizer, model, action))
	}
	// Departments
	r.Route("/departments", func(dr chi.Router) {
		access(dr, "hr.department", auth.ActionCreate).Post("/", h.CreateDepartment)
		access(dr, "hr.department", auth.ActionRead).Get("/", h.ListDepartments)
		access(dr, "hr.department", auth.ActionRead).Get("/tree", h.GetDepartmentTree)
		access(dr, "hr.department", auth.ActionRead).Get("/{id}", h.GetDepartment)
		access(dr, "hr.department", auth.ActionWrite).Put("/{id}", h.UpdateDepartment)
		access(dr, "hr.department", auth.ActionUnlink).Delete("/{id}", h.DeleteDepartment)
	})

	// Jobs
	r.Route("/jobs", func(jr chi.Router) {
		access(jr, "hr.job", auth.ActionCreate).Post("/", h.CreateJob)
		access(jr, "hr.job", auth.ActionRead).Get("/", h.ListJobs)
		access(jr, "hr.job", auth.ActionRead).Get("/{id}", h.GetJob)
		access(jr, "hr.job", auth.ActionWrite).Put("/{id}", h.UpdateJob)
		access(jr, "hr.job", auth.ActionUnlink).Delete("/{id}", h.DeleteJob)
	})

	// Employees
	r.Route("/employees", func(er chi.Router) {
		access(er, "hr.employee", auth.ActionCreate).Post("/", h.CreateEmployee)
		access(er, "hr.employee", auth.ActionRead).Get("/", h.ListEmployees)
		access(er, "hr.employee", auth.ActionRead).Get("/{id}/leave-balance", h.GetEmployeeLeaveBalance)
		access(er, "hr.employee", auth.ActionRead).Get("/{id}", h.GetEmployee)
		access(er, "hr.employee", auth.ActionWrite).Put("/{id}", h.UpdateEmployee)
		access(er, "hr.employee", auth.ActionUnlink).Delete("/{id}", h.DeleteEmployee)
	})

	// Leave Allocations
	r.Route("/leave-allocations", func(ar chi.Router) {
		access(ar, "hr.leave_allocation", auth.ActionCreate).Post("/", h.CreateAllocation)
		access(ar, "hr.leave_allocation", auth.ActionRead).Get("/", h.ListAllocations)
		access(ar, "hr.leave_allocation", auth.ActionRead).Get("/{id}", h.GetAllocation)
	})

	// Leave Requests
	r.Route("/leave-requests", func(lr chi.Router) {
		access(lr, "hr.leave_request", auth.ActionCreate).Post("/", h.CreateLeaveRequest)
		access(lr, "hr.leave_request", auth.ActionRead).Get("/", h.ListLeaveRequests)
		access(lr, "hr.leave_request", auth.ActionRead).Get("/{id}", h.GetLeaveRequest)
		access(lr, "hr.leave_request", auth.ActionWrite).Put("/{id}", h.UpdateLeaveRequest)
		access(lr, "hr.leave_request", auth.ActionUnlink).Delete("/{id}", h.DeleteLeaveRequest)
		access(lr, "hr.leave_request", auth.ActionWrite).Post("/{id}/confirm", h.ConfirmLeaveRequest)
		access(lr, "hr.leave_request", auth.ActionWrite).Post("/{id}/approve", h.ApproveLeaveRequest)
		access(lr, "hr.leave_request", auth.ActionWrite).Post("/{id}/refuse", h.RefuseLeaveRequest)
		access(lr, "hr.leave_request", auth.ActionWrite).Post("/{id}/cancel", h.CancelLeaveRequest)
	})

	// Attendance
	r.Route("/attendance", func(ar chi.Router) {
		access(ar, "hr.attendance", auth.ActionWrite).Post("/check-in", h.CheckIn)
		access(ar, "hr.attendance", auth.ActionWrite).Post("/check-out", h.CheckOut)
		access(ar, "hr.attendance", auth.ActionRead).Get("/kiosk-config", h.GetKioskConfig)
	})

	// Overtime
	r.Route("/overtime", func(or chi.Router) {
		access(or, "hr.overtime", auth.ActionWrite).Post("/{id}/approve", h.ApproveOvertime)
	})

	// Employee Reports
	r.Get("/employees/{id}/attendance-report", h.GetAttendanceReport)
}
