package purchasehttp

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	purchaseusecase "cashflow_backend/internal/usecase/purchase"

	"github.com/go-chi/chi/v5"
)

// Handler serves HTTP requests for the Purchase domain.
type Handler struct {
	useCase *purchaseusecase.UseCase
	logger  *slog.Logger
}

// NewHandler constructs a Purchase HTTP Handler.
func NewHandler(useCase *purchaseusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func parseID(param string) (int64, error) {
	return strconv.ParseInt(param, 10, 64)
}

// CreateOrder handles POST /api/v1/purchase-orders
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req CreatePurchaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	order, err := h.useCase.CreateOrder(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToPurchaseOrderResponse(order))
}

// GetOrder handles GET /api/v1/purchase-orders/{id}
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase order ID in path", err))
		return
	}

	order, err := h.useCase.GetOrderByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPurchaseOrderResponse(order))
}

// ListOrders handles GET /api/v1/purchase-orders
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	pageReq := pagination.Parse(r)

	f := filter.NewFilter()
	q := r.URL.Query()
	if partnerID := q.Get("partner_id"); partnerID != "" {
		f.Add("partner_id", filter.OpEqual, partnerID)
	}
	if state := q.Get("state"); state != "" {
		f.Add("state", filter.OpEqual, state)
	}
	if invStatus := q.Get("invoice_status"); invStatus != "" {
		f.Add("invoice_status", filter.OpEqual, invStatus)
	}
	if name := q.Get("name"); name != "" {
		f.Add("name", filter.OpLike, name)
	}

	pageRes, err := h.useCase.ListOrders(r.Context(), f, pageReq)
	if err != nil {
		response.Error(w, err)
		return
	}

	mappedItems := ToPurchaseOrderListResponse(pageRes.Items)
	res := pagination.NewPageResult(mappedItems, pageRes.TotalItems, pageReq)
	response.Paginated(w, http.StatusOK, mappedItems, res)
}

// UpdateOrder handles PUT /api/v1/purchase-orders/{id}
func (h *Handler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase order ID in path", err))
		return
	}

	var req UpdatePurchaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	order, err := h.useCase.UpdateOrder(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPurchaseOrderResponse(order))
}

// DeleteOrder handles DELETE /api/v1/purchase-orders/{id}
func (h *Handler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase order ID in path", err))
		return
	}

	if err := h.useCase.DeleteOrder(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// ActionSend handles POST /api/v1/purchase-orders/{id}/send
func (h *Handler) ActionSend(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase order ID in path", err))
		return
	}

	order, err := h.useCase.ActionSend(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPurchaseOrderResponse(order))
}

// ConfirmOrder handles POST /api/v1/purchase-orders/{id}/confirm
func (h *Handler) ConfirmOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase order ID in path", err))
		return
	}

	order, err := h.useCase.ConfirmOrder(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPurchaseOrderResponse(order))
}

// CancelOrder handles POST /api/v1/purchase-orders/{id}/cancel
func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase order ID in path", err))
		return
	}

	order, err := h.useCase.CancelOrder(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPurchaseOrderResponse(order))
}

// ResetToDraft handles POST /api/v1/purchase-orders/{id}/draft
func (h *Handler) ResetToDraft(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase order ID in path", err))
		return
	}

	order, err := h.useCase.ResetToDraft(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPurchaseOrderResponse(order))
}

// CreateBill handles POST /api/v1/purchase-orders/{id}/bill
func (h *Handler) CreateBill(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase order ID in path", err))
		return
	}

	var req CreateBillFromOrderRequest
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &req); err != nil {
				response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
				return
			}
		}
	}

	move, err := h.useCase.CreateBillFromOrder(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, move)
}

// GetOrderBills handles GET /api/v1/purchase-orders/{id}/bills
func (h *Handler) GetOrderBills(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase order ID in path", err))
		return
	}

	bills, err := h.useCase.GetOrderBills(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, bills)
}
