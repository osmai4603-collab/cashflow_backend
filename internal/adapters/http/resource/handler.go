package resourcehttp

import (
	"encoding/json"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/resource"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
	resourceusecase "cashflow_backend/internal/usecase/resource"
)

type Handler struct{ useCase *resourceusecase.UseCase }

func NewHandler(useCase *resourceusecase.UseCase) *Handler { return &Handler{useCase: useCase} }
func (handler *Handler) CreateCalendar(writer http.ResponseWriter, request *http.Request) {
	var calendarValue resource.Calendar
	if err := json.NewDecoder(request.Body).Decode(&calendarValue); err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	created, err := handler.useCase.CreateCalendar(request.Context(), &calendarValue)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.Created(writer, created)
}
func (handler *Handler) CreateWorkEntry(writer http.ResponseWriter, request *http.Request) {
	var entry resource.WorkEntry
	if err := json.NewDecoder(request.Body).Decode(&entry); err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	created, err := handler.useCase.CreateWorkEntry(request.Context(), &entry)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.Created(writer, created)
}
func (handler *Handler) ListWorkEntries(writer http.ResponseWriter, request *http.Request) {
	employeeID, err := strconv.ParseInt(request.URL.Query().Get("employee_id"), 10, 64)
	if err != nil {
		response.Error(writer, platformerrors.BadRequest("employee_id is required", err))
		return
	}
	result, err := handler.useCase.ListWorkEntries(request.Context(), employeeID)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, result)
}
