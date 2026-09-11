package timesheethttp

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"cashflow_backend/internal/domain/timesheet"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
	timesheetusecase "cashflow_backend/internal/usecase/timesheet"
)

type Handler struct{ useCase *timesheetusecase.UseCase }

func NewHandler(useCase *timesheetusecase.UseCase) *Handler { return &Handler{useCase: useCase} }
func (handler *Handler) CreateEntry(writer http.ResponseWriter, request *http.Request) {
	var entry timesheet.Entry
	if err := json.NewDecoder(request.Body).Decode(&entry); err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	created, err := handler.useCase.CreateEntry(request.Context(), &entry)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.Created(writer, created)
}
func (handler *Handler) ListEntries(writer http.ResponseWriter, request *http.Request) {
	filter := timesheet.Filter{}
	if raw := request.URL.Query().Get("project_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			response.Error(writer, platformerrors.BadRequest("invalid project_id", err))
			return
		}
		filter.ProjectID = &id
	}
	result, err := handler.useCase.ListEntries(request.Context(), filter)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, result)
}
func (handler *Handler) StartTimer(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		TaskID     int64 `json:"task_id"`
		EmployeeID int64 `json:"employee_id"`
	}
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	timer, err := handler.useCase.StartTimer(request.Context(), input.TaskID, input.EmployeeID, time.Now().UTC())
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, timer)
}
func (handler *Handler) StopTimer(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		EmployeeID int64   `json:"employee_id"`
		ProjectID  int64   `json:"project_id"`
		UserID     int64   `json:"user_id"`
		CompanyID  int64   `json:"company_id"`
		HourlyCost float64 `json:"hourly_cost"`
	}
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	entry, err := handler.useCase.StopTimer(request.Context(), input.EmployeeID, time.Now().UTC(), input.ProjectID, input.UserID, input.CompanyID, input.HourlyCost)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.Created(writer, entry)
}
func (handler *Handler) Submit(writer http.ResponseWriter, request *http.Request) {
	id, err := strconv.ParseInt(request.URL.Query().Get("id"), 10, 64)
	if err != nil {
		response.Error(writer, platformerrors.BadRequest("id is required", err))
		return
	}
	entry, err := handler.useCase.Submit(request.Context(), id)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, entry)
}
func (handler *Handler) Approve(writer http.ResponseWriter, request *http.Request) {
	id, err := strconv.ParseInt(request.URL.Query().Get("id"), 10, 64)
	if err != nil {
		response.Error(writer, platformerrors.BadRequest("id is required", err))
		return
	}
	entry, err := handler.useCase.Approve(request.Context(), id)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, entry)
}
