package qualityhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"cashflow_backend/internal/domain/quality"
	"cashflow_backend/internal/platform/response"
	qualityusecase "cashflow_backend/internal/usecase/quality"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *qualityusecase.UseCase
	logger  *slog.Logger
}

func NewHandler(useCase *qualityusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *Handler) ListPoints(w http.ResponseWriter, r *http.Request) {
	points, err := h.useCase.ListControlPoints(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, points)
}

func (h *Handler) CreatePoint(w http.ResponseWriter, r *http.Request) {
	var p quality.ControlPoint
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.useCase.CreateControlPoint(r.Context(), &p)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

func (h *Handler) ExecuteCheck(w http.ResponseWriter, r *http.Request) {
	var c quality.Check
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		response.Error(w, err)
		return
	}
	result, err := h.useCase.ExecuteCheck(r.Context(), &c)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.useCase.ListAlerts(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, alerts)
}

func (h *Handler) CreateAlert(w http.ResponseWriter, r *http.Request) {
	var a quality.Alert
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.useCase.CreateAlert(r.Context(), &a)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}
