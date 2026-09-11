package calendarhttp

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"cashflow_backend/internal/domain/calendar"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
	calendarusecase "cashflow_backend/internal/usecase/calendar"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *calendarusecase.UseCase
}

func NewHandler(useCase *calendarusecase.UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, target any) bool {
	if err := json.NewDecoder(request.Body).Decode(target); err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid JSON request body", err))
		return false
	}
	return true
}

func parseCalendarID(writer http.ResponseWriter, request *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(request, "id"), 10, 64)
	if err != nil {
		response.Error(writer, platformerrors.BadRequest("invalid calendar ID", err))
		return 0, false
	}
	return id, true
}

func (handler *Handler) CreateEvent(writer http.ResponseWriter, request *http.Request) {
	var event calendar.CalendarEvent
	if !decodeJSON(writer, request, &event) {
		return
	}
	created, err := handler.useCase.CreateEvent(request.Context(), &event)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.Created(writer, created)
}

func (handler *Handler) GetEvent(writer http.ResponseWriter, request *http.Request) {
	id, ok := parseCalendarID(writer, request)
	if !ok {
		return
	}
	event, err := handler.useCase.GetEvent(request.Context(), id)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, event)
}

func (handler *Handler) UpdateEvent(writer http.ResponseWriter, request *http.Request) {
	var event calendar.CalendarEvent
	if !decodeJSON(writer, request, &event) {
		return
	}
	id, ok := parseCalendarID(writer, request)
	if !ok {
		return
	}
	event.ID = id
	updated, err := handler.useCase.UpdateEvent(request.Context(), &event)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, updated)
}

func (handler *Handler) DeleteEvent(writer http.ResponseWriter, request *http.Request) {
	id, ok := parseCalendarID(writer, request)
	if !ok {
		return
	}
	if err := handler.useCase.DeleteEvent(request.Context(), id); err != nil {
		response.Error(writer, err)
		return
	}
	response.NoContent(writer)
}

func (handler *Handler) ListEvents(writer http.ResponseWriter, request *http.Request) {
	filter := calendar.EventFilter{}
	if raw := request.URL.Query().Get("user_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			response.Error(writer, platformerrors.BadRequest("invalid user_id", err))
			return
		}
		filter.UserID = &id
	}
	if raw := request.URL.Query().Get("from"); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			response.Error(writer, platformerrors.BadRequest("invalid from timestamp", err))
			return
		}
		filter.From = &value
	}
	if raw := request.URL.Query().Get("to"); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			response.Error(writer, platformerrors.BadRequest("invalid to timestamp", err))
			return
		}
		filter.To = &value
	}
	events, err := handler.useCase.ListEvents(request.Context(), filter)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, events)
}

func (handler *Handler) ExportICS(writer http.ResponseWriter, request *http.Request) {
	id, ok := parseCalendarID(writer, request)
	if !ok {
		return
	}
	event, err := handler.useCase.GetEvent(request.Context(), id)
	if err != nil {
		response.Error(writer, err)
		return
	}
	ics, err := calendar.ExportICS(event)
	if err != nil {
		response.Error(writer, err)
		return
	}
	writer.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(ics)
}

func (handler *Handler) RespondAttendee(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Token  string                  `json:"token"`
		Status calendar.AttendeeStatus `json:"status"`
	}
	if !decodeJSON(writer, request, &input) {
		return
	}
	attendee, err := handler.useCase.RespondAttendee(request.Context(), input.Token, input.Status)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, attendee)
}

func (handler *Handler) CreateAppointmentType(writer http.ResponseWriter, request *http.Request) {
	var appointmentType calendar.AppointmentType
	if !decodeJSON(writer, request, &appointmentType) {
		return
	}
	created, err := handler.useCase.CreateAppointmentType(request.Context(), &appointmentType)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.Created(writer, created)
}

func (handler *Handler) ListAppointmentTypes(writer http.ResponseWriter, request *http.Request) {
	companyID, err := strconv.ParseInt(request.URL.Query().Get("company_id"), 10, 64)
	if err != nil {
		response.Error(writer, platformerrors.BadRequest("company_id is required", err))
		return
	}
	result, err := handler.useCase.ListAppointmentTypes(request.Context(), companyID)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, result)
}

func (handler *Handler) ListSlots(writer http.ResponseWriter, request *http.Request) {
	appointmentType, err := handler.useCase.GetAppointmentTypeBySlug(request.Context(), chi.URLParam(request, "slug"))
	if err != nil {
		response.Error(writer, err)
		return
	}
	from, err := time.Parse(time.RFC3339, request.URL.Query().Get("from"))
	if err != nil {
		response.Error(writer, platformerrors.BadRequest("from timestamp is required", err))
		return
	}
	to, err := time.Parse(time.RFC3339, request.URL.Query().Get("to"))
	if err != nil {
		response.Error(writer, platformerrors.BadRequest("to timestamp is required", err))
		return
	}
	slots, err := handler.useCase.GetAvailableSlots(request.Context(), appointmentType.ID, from, to)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.JSON(writer, http.StatusOK, slots)
}

func (handler *Handler) BookAppointment(writer http.ResponseWriter, request *http.Request) {
	var booking calendar.AppointmentBooking
	if !decodeJSON(writer, request, &booking) {
		return
	}
	created, err := handler.useCase.BookAppointment(request.Context(), chi.URLParam(request, "slug"), &booking)
	if err != nil {
		response.Error(writer, err)
		return
	}
	response.Created(writer, created)
}
