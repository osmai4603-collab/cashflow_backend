package activityhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"cashflow_backend/internal/domain/activity"
	"cashflow_backend/internal/platform/auth"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/notificationbus"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	activityusecase "cashflow_backend/internal/usecase/activity"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *activityusecase.UseCase
	logger  *slog.Logger
	bus     *notificationbus.Bus
}

func NewHandler(useCase *activityusecase.UseCase, logger *slog.Logger, buses ...*notificationbus.Bus) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	var bus *notificationbus.Bus
	if len(buses) > 0 {
		bus = buses[0]
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
		bus:     bus,
	}
}

// --- Activity Type Handlers ---

func (h *Handler) ListTypes(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	var companyID *int64
	if claims != nil {
		companyID = &claims.CompanyID
	}

	types, err := h.useCase.ListTypes(r.Context(), companyID)
	if err != nil {
		response.Error(w, err)
		return
	}

	res := make([]ActivityTypeDTO, len(types))
	for i, t := range types {
		res[i] = ToActivityTypeDTO(t)
	}
	response.JSON(w, http.StatusOK, res)
}

// --- Activity Handlers ---

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, platformerrors.Unauthorized("unauthenticated"))
		return
	}

	var req CreateActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid body", err))
		return
	}

	a := &activity.Activity{
		ActivityTypeID: req.ActivityTypeID,
		Summary:        req.Summary,
		Note:           req.Note,
		DateDeadline:   req.DateDeadline,
		AssignedUserID: req.AssignedUserID,
		ResModel:       req.ResModel,
		ResID:          req.ResID,
		Active:         true,
		CompanyID:      claims.CompanyID,
		CreatedBy:      &claims.UserID,
		UpdatedBy:      &claims.UserID,
	}

	if err := h.useCase.Schedule(r.Context(), a); err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToActivityDTO(*a, time.Now().UTC(), time.UTC))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	a, err := h.useCase.GetActivity(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToActivityDTO(*a, time.Now().UTC(), time.UTC))
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req CompleteActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid body", err))
		return
	}

	a, err := h.useCase.Complete(r.Context(), id, req.Feedback)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToActivityDTO(*a, time.Now().UTC(), time.UTC))
}

func (h *Handler) ListMy(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, platformerrors.Unauthorized("unauthenticated"))
		return
	}

	page := pagination.Parse(r)
	active := true
	filter := activity.ActivityFilter{
		AssignedUserID: &claims.UserID,
		Active:         &active,
		Page:           page,
	}

	result, err := h.useCase.ListActivities(r.Context(), filter)
	if err != nil {
		response.Error(w, err)
		return
	}

	res := make([]ActivityDTO, len(result.Items))
	now := time.Now().UTC()
	for i, a := range result.Items {
		res[i] = ToActivityDTO(a, now, time.UTC)
	}
	response.Paginated(w, http.StatusOK, res, result)
}

// --- Notification Handlers ---

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, platformerrors.Unauthorized("unauthenticated"))
		return
	}

	page := pagination.Parse(r)
	var status *activity.NotificationStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := activity.NotificationStatus(s)
		status = &st
	}

	result, err := h.useCase.ListNotifications(r.Context(), claims.UserID, status, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	res := make([]NotificationDTO, len(result.Items))
	for i, n := range result.Items {
		res[i] = ToNotificationDTO(n)
	}
	response.Paginated(w, http.StatusOK, res, result)
}

func (h *Handler) StreamNotifications(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, platformerrors.Unauthorized("unauthenticated"))
		return
	}
	if h.bus == nil {
		response.Error(w, platformerrors.Internal("notification stream is unavailable", nil))
		return
	}

	events, unsubscribe := h.bus.Subscribe(claims.UserID)
	defer unsubscribe()

	flusher, ok := w.(http.Flusher)
	if !ok {
		response.Error(w, platformerrors.Internal("streaming is not supported", nil))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	encoder := json.NewEncoder(w)
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if _, err := w.Write([]byte("event: " + event.Channel + "\ndata: ")); err != nil {
				return
			}
			if err := encoder.Encode(event.Payload); err != nil {
				return
			}
			if _, err := w.Write([]byte("\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (h *Handler) MarkNotifRead(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.useCase.MarkRead(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) MarkAllNotifsRead(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, platformerrors.Unauthorized("unauthenticated"))
		return
	}

	if err := h.useCase.MarkAllRead(r.Context(), claims.UserID, claims.CompanyID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}
