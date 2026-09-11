package repairhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/repair"
	"cashflow_backend/internal/platform/response"
	repairusecase "cashflow_backend/internal/usecase/repair"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *repairusecase.UseCase
	logger  *slog.Logger
}

func NewHandler(useCase *repairusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orders, err := h.useCase.ListOrders(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, orders)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var o repair.Order
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		response.Error(w, err)
		return
	}
	created, err := h.useCase.CreateOrder(r.Context(), &o)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, created)
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.useCase.ConfirmOrder(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.useCase.CompleteOrder(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	invoiceID, err := h.useCase.CreateInvoice(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]int64{"invoice_id": invoiceID})
}
