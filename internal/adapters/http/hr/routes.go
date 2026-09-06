package hrhttp

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers all HR domain endpoints onto the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler) {
	// Departments
	r.Route("/departments", func(dr chi.Router) {
		dr.Post("/", h.CreateDepartment)
		dr.Get("/", h.ListDepartments)
		dr.Get("/tree", h.GetDepartmentTree)
		dr.Get("/{id}", h.GetDepartment)
		dr.Put("/{id}", h.UpdateDepartment)
		dr.Delete("/{id}", h.DeleteDepartment)
	})

	// Jobs
	r.Route("/jobs", func(jr chi.Router) {
		jr.Post("/", h.CreateJob)
		jr.Get("/", h.ListJobs)
		jr.Get("/{id}", h.GetJob)
		jr.Put("/{id}", h.UpdateJob)
		jr.Delete("/{id}", h.DeleteJob)
	})

	// Employees
	r.Route("/employees", func(er chi.Router) {
		er.Post("/", h.CreateEmployee)
		er.Get("/", h.ListEmployees)
		er.Get("/{id}/leave-balance", h.GetEmployeeLeaveBalance)
		er.Get("/{id}", h.GetEmployee)
		er.Put("/{id}", h.UpdateEmployee)
		er.Delete("/{id}", h.DeleteEmployee)
	})

	// Leave Allocations
	r.Route("/leave-allocations", func(ar chi.Router) {
		ar.Post("/", h.CreateAllocation)
		ar.Get("/", h.ListAllocations)
		ar.Get("/{id}", h.GetAllocation)
	})

	// Leave Requests
	r.Route("/leave-requests", func(lr chi.Router) {
		lr.Post("/", h.CreateLeaveRequest)
		lr.Get("/", h.ListLeaveRequests)
		lr.Get("/{id}", h.GetLeaveRequest)
		lr.Put("/{id}", h.UpdateLeaveRequest)
		lr.Delete("/{id}", h.DeleteLeaveRequest)
		lr.Post("/{id}/confirm", h.ConfirmLeaveRequest)
		lr.Post("/{id}/approve", h.ApproveLeaveRequest)
		lr.Post("/{id}/refuse", h.RefuseLeaveRequest)
		lr.Post("/{id}/cancel", h.CancelLeaveRequest)
	})
}
