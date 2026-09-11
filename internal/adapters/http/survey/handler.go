package surveyhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/survey"
	"cashflow_backend/internal/platform/response"
	surveyusecase "cashflow_backend/internal/usecase/survey"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *surveyusecase.UseCase
	logger  *slog.Logger
}

func NewHandler(useCase *surveyusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	surveys, err := h.useCase.ListSurveys(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, surveys)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var s survey.Survey
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.useCase.CreateSurvey(r.Context(), &s)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	s, err := h.useCase.GetSurvey(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, s)
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var input survey.SurveyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, err)
		return
	}
	input.SurveyID = id
	result, err := h.useCase.SubmitSurvey(r.Context(), &input)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
