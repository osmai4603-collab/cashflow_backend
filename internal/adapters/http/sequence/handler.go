package sequencehttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
	sequenceusecase "cashflow_backend/internal/usecase/sequence"

	"github.com/go-chi/chi/v5"
)

// Handler serves HTTP requests for the Sequence domain.
type Handler struct {
	useCase sequenceusecase.UseCase
	logger  *slog.Logger
}

// NewHandler constructs a new Handler.
func NewHandler(useCase sequenceusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

// Create handles POST /api/v1/sequences
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateSequenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	created, err := h.useCase.CreateSequence(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToSequenceResponse(created))
}

// List handles GET /api/v1/sequences
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.useCase.ListSequences(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"items": ToSequenceResponseList(items),
		"count": len(items),
	})
}

// GetByID handles GET /api/v1/sequences/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseSequenceID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid sequence ID in path", err))
		return
	}

	s, err := h.useCase.GetSequence(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToSequenceResponse(s))
}

// Update handles PUT /api/v1/sequences/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseSequenceID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid sequence ID in path", err))
		return
	}

	var req UpdateSequenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdateSequence(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToSequenceResponse(updated))
}

// Delete handles DELETE /api/v1/sequences/{id} (soft-delete)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseSequenceID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid sequence ID in path", err))
		return
	}

	if err := h.useCase.DeleteSequence(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// NextNumber handles POST /api/v1/sequences/{id}/next
func (h *Handler) NextNumber(w http.ResponseWriter, r *http.Request) {
	id, err := parseSequenceID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid sequence ID in path", err))
		return
	}

	date := time.Now().UTC()
	if r.Body != nil {
		var req NextNumberRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && strings.TrimSpace(req.Date) != "" {
			if parsed, err := time.Parse("2006-01-02", strings.TrimSpace(req.Date)); err == nil {
				date = parsed.UTC()
			}
		}
	}

	ref, err := h.useCase.GenerateNextByID(r.Context(), id, date)
	if err != nil {
		response.Error(w, err)
		return
	}

	s, err := h.useCase.GetSequence(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, NextNumberResponse{
		SequenceID: s.ID,
		Code:       s.Code,
		Number:     s.CurrentNumber,
		Reference:  ref,
	})
}

func parseSequenceID(param string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(param), 10, 64)
}
