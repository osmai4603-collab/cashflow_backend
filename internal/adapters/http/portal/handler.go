package portalhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"cashflow_backend/internal/platform/response"
	portalusecase "cashflow_backend/internal/usecase/portal"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *portalusecase.UseCase
	logger  *slog.Logger
}

func NewHandler(useCase *portalusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	user, token, err := h.useCase.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

func (h *Handler) RegisterInvite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		InviteToken string `json:"invite_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.Register(r.Context(), req.Email, req.Password, req.InviteToken); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	// User ID from context (extracted by auth middleware)
	userID := int64(1) // stub
	summary, err := h.useCase.GetDashboardSummary(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, summary)
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	userID := int64(1) // stub
	orders, err := h.useCase.ListOrders(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, orders)
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	order, err := h.useCase.GetOrder(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, order)
}

func (h *Handler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	userID := int64(1) // stub
	invoices, err := h.useCase.ListInvoices(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, invoices)
}

func (h *Handler) GetInvoicePDF(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	pdf, err := h.useCase.GetInvoicePDF(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Write(pdf)
}

func (h *Handler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	userID := int64(1) // stub
	deliveries, err := h.useCase.ListDeliveries(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, deliveries)
}
