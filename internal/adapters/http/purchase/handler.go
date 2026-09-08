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
	useCase            *purchaseusecase.UseCase
	requisitionUseCase *purchaseusecase.RequisitionUseCase
	logger             *slog.Logger
}

// NewHandler constructs a Purchase HTTP Handler.
func NewHandler(useCase *purchaseusecase.UseCase, logger *slog.Logger, requisitionUseCase ...*purchaseusecase.RequisitionUseCase) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	var reqUC *purchaseusecase.RequisitionUseCase
	if len(requisitionUseCase) > 0 {
		reqUC = requisitionUseCase[0]
	}
	return &Handler{
		useCase:            useCase,
		requisitionUseCase: reqUC,
		logger:             logger,
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

func (h *Handler) CreateAlternativeGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase order ID in path", err))
		return
	}
	var input struct {
		OrderIDs []int64 `json:"order_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	if err := h.useCase.CreateAlternativeGroup(r.Context(), id, input.OrderIDs); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"order_id": id, "alternative_order_ids": input.OrderIDs})
}

func (h *Handler) ListAlternativeOrders(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase order ID in path", err))
		return
	}
	ids, err := h.useCase.ListAlternativeOrderIDs(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"order_id": id, "alternative_order_ids": ids})
}

func (h *Handler) ClearAlternativeGroup(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase order ID in path", err))
		return
	}
	if err := h.useCase.ClearAlternativeGroup(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
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

// CreateRequisition handles POST /api/v1/purchase-requisitions
func (h *Handler) CreateRequisition(w http.ResponseWriter, r *http.Request) {
	if h.requisitionUseCase == nil {
		response.Error(w, platformerrors.Internal("purchase requisition use case is not configured"))
		return
	}

	var req CreatePurchaseRequisitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	requisition, err := h.requisitionUseCase.CreateRequisition(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToPurchaseRequisitionResponse(requisition))
}

// GetRequisition handles GET /api/v1/purchase-requisitions/{id}
func (h *Handler) GetRequisition(w http.ResponseWriter, r *http.Request) {
	if h.requisitionUseCase == nil {
		response.Error(w, platformerrors.Internal("purchase requisition use case is not configured"))
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase requisition ID in path", err))
		return
	}

	requisition, err := h.requisitionUseCase.GetRequisitionByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPurchaseRequisitionResponse(requisition))
}

// UpdateRequisition handles PUT /api/v1/purchase-requisitions/{id}.
func (h *Handler) UpdateRequisition(w http.ResponseWriter, r *http.Request) {
	if h.requisitionUseCase == nil {
		response.Error(w, platformerrors.Internal("purchase requisition use case is not configured"))
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase requisition ID in path", err))
		return
	}
	var req UpdatePurchaseRequisitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	requisition, err := h.requisitionUseCase.UpdateRequisition(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToPurchaseRequisitionResponse(requisition))
}

// DeleteRequisition handles DELETE /api/v1/purchase-requisitions/{id}.
func (h *Handler) DeleteRequisition(w http.ResponseWriter, r *http.Request) {
	if h.requisitionUseCase == nil {
		response.Error(w, platformerrors.Internal("purchase requisition use case is not configured"))
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase requisition ID in path", err))
		return
	}
	if err := h.requisitionUseCase.DeleteRequisition(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListRequisitions handles GET /api/v1/purchase-requisitions
func (h *Handler) ListRequisitions(w http.ResponseWriter, r *http.Request) {
	if h.requisitionUseCase == nil {
		response.Error(w, platformerrors.Internal("purchase requisition use case is not configured"))
		return
	}

	pageReq := pagination.Parse(r)
	result, err := h.requisitionUseCase.ListRequisitions(r.Context(), pageReq)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Paginated(w, http.StatusOK, result.Items, result)
}

// ConfirmRequisition handles POST /api/v1/purchase-requisitions/{id}/confirm
func (h *Handler) ConfirmRequisition(w http.ResponseWriter, r *http.Request) {
	if h.requisitionUseCase == nil {
		response.Error(w, platformerrors.Internal("purchase requisition use case is not configured"))
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase requisition ID in path", err))
		return
	}

	requisition, err := h.requisitionUseCase.ConfirmRequisition(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPurchaseRequisitionResponse(requisition))
}

// CloseRequisition handles POST /api/v1/purchase-requisitions/{id}/close
func (h *Handler) CloseRequisition(w http.ResponseWriter, r *http.Request) {
	if h.requisitionUseCase == nil {
		response.Error(w, platformerrors.Internal("purchase requisition use case is not configured"))
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase requisition ID in path", err))
		return
	}

	requisition, err := h.requisitionUseCase.CloseRequisition(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPurchaseRequisitionResponse(requisition))
}

// CancelRequisition handles POST /api/v1/purchase-requisitions/{id}/cancel
func (h *Handler) CancelRequisition(w http.ResponseWriter, r *http.Request) {
	if h.requisitionUseCase == nil {
		response.Error(w, platformerrors.Internal("purchase requisition use case is not configured"))
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase requisition ID in path", err))
		return
	}

	requisition, err := h.requisitionUseCase.CancelRequisition(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPurchaseRequisitionResponse(requisition))
}

// CreatePOFromRequisition handles POST /api/v1/purchase-requisitions/{id}/create-po
func (h *Handler) CreatePOFromRequisition(w http.ResponseWriter, r *http.Request) {
	if h.requisitionUseCase == nil {
		response.Error(w, platformerrors.Internal("purchase requisition use case is not configured"))
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid purchase requisition ID in path", err))
		return
	}

	order, err := h.requisitionUseCase.CreatePurchaseOrderFromRequisition(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToPurchaseOrderResponse(order))
}
