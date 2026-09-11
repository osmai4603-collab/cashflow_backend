package subscriptionhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/subscription"
	"cashflow_backend/internal/platform/response"
	subscriptionusecase "cashflow_backend/internal/usecase/subscription"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *subscriptionusecase.UseCase
	logger  *slog.Logger
}

func NewHandler(useCase *subscriptionusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	subs, err := h.useCase.ListSubscriptions(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, subs)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var s subscription.SaleSubscription
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.useCase.CreateSubscription(r.Context(), &s)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	sub, err := h.useCase.GetSubscription(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, sub)
}

func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.useCase.ActivateSubscription(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.useCase.CancelSubscription(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) ProcessBilling(w http.ResponseWriter, r *http.Request) {
	if err := h.useCase.ProcessBillingCycle(r.Context()); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) GetMRR(w http.ResponseWriter, r *http.Request) {
	mrr, err := h.useCase.GetMRR(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]float64{"mrr": mrr})
}
