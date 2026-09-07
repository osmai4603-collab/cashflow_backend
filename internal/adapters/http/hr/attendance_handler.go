package hrhttp

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"cashflow_backend/internal/domain/hr"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CheckIn(w http.ResponseWriter, r *http.Request) {
	var req CheckInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}

	// In a real app, employeeID would come from the auth context
	// For now, we might take it from query or assume one for testing
	empIDStr := r.URL.Query().Get("employee_id")
	empID, _ := strconv.ParseInt(empIDStr, 10, 64)
	if empID == 0 {
		response.Error(w, platformerrors.BadRequest("employee_id is required", nil))
		return
	}

	info := hr.Attendance{
		InLatitude:  req.Latitude,
		InLongitude: req.Longitude,
		InIPAddress: req.IPAddress,
		InBrowser:   req.Browser,
		InMode:      req.Mode,
	}

	att, err := h.attendanceUseCase.CheckIn(r.Context(), empID, info)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToAttendanceResponse(att))
}

func (h *Handler) CheckOut(w http.ResponseWriter, r *http.Request) {
	var req CheckOutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}

	empIDStr := r.URL.Query().Get("employee_id")
	empID, _ := strconv.ParseInt(empIDStr, 10, 64)
	if empID == 0 {
		response.Error(w, platformerrors.BadRequest("employee_id is required", nil))
		return
	}

	info := hr.Attendance{
		OutLatitude:  req.Latitude,
		OutLongitude: req.Longitude,
		OutIPAddress: req.IPAddress,
		OutBrowser:   req.Browser,
		OutMode:      req.Mode,
	}

	att, err := h.attendanceUseCase.CheckOut(r.Context(), empID, info)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToAttendanceResponse(att))
}

func (h *Handler) GetKioskConfig(w http.ResponseWriter, r *http.Request) {
	companyID, _ := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
	if companyID == 0 {
		companyID = 1 // Default
	}

	config, err := h.attendanceUseCase.GetKioskConfig(r.Context(), companyID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, config)
}

func (h *Handler) ApproveOvertime(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid overtime line ID", err))
		return
	}

	var req ApproveOvertimeRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	err = h.attendanceUseCase.ApproveOvertime(r.Context(), id, req.ManagerID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) GetAttendanceReport(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid employee ID", err))
		return
	}

	// Parse date range from query params
	// For now just returning empty list as placeholder
	report, err := h.attendanceUseCase.GetAttendanceReport(r.Context(), id, time.Time{}, time.Time{})
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]AttendanceResponse, len(report))
	for i := range report {
		dtos[i] = ToAttendanceResponse(&report[i])
	}

	response.JSON(w, http.StatusOK, dtos)
}
