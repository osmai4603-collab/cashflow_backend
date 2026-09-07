package stockhttp

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

// ─────────────────────────────────────────────────────────────────────────────
// Orderpoints (Phase 13 — auto reorder)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateOrderpoint(w http.ResponseWriter, r *http.Request) {
	var req OrderpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	op, err := h.uc.CreateOrderpoint(r.Context(), req.ToCreateInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToOrderpointResponse(op))
}

func (h *Handler) GetOrderpoint(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid orderpoint id", nil))
		return
	}

	op, err := h.uc.GetOrderpoint(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToOrderpointResponse(op))
}

func (h *Handler) ListOrderpoints(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()

	if pid := strings.TrimSpace(r.URL.Query().Get("product_id")); pid != "" {
		if v, err := strconv.ParseInt(pid, 10, 64); err == nil && v > 0 {
			f.Add("product_id", filter.OpEqual, v)
		}
	}
	if wid := strings.TrimSpace(r.URL.Query().Get("warehouse_id")); wid != "" {
		if v, err := strconv.ParseInt(wid, 10, 64); err == nil && v > 0 {
			f.Add("warehouse_id", filter.OpEqual, v)
		}
	}
	if lid := strings.TrimSpace(r.URL.Query().Get("location_id")); lid != "" {
		if v, err := strconv.ParseInt(lid, 10, 64); err == nil && v > 0 {
			f.Add("location_id", filter.OpEqual, v)
		}
	}
	if trigger := strings.TrimSpace(r.URL.Query().Get("trigger")); trigger != "" {
		f.Add("trigger", filter.OpEqual, trigger)
	}

	res, err := h.uc.ListOrderpoints(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]OrderpointResponse, len(res.Items))
	for i := range res.Items {
		dtos[i] = ToOrderpointResponse(&res.Items[i])
	}

	pageRes := pagination.NewPageResult(dtos, res.TotalItems, page)
	response.Paginated(w, http.StatusOK, dtos, pageRes)
}

func (h *Handler) UpdateOrderpoint(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid orderpoint id", nil))
		return
	}

	var req OrderpointUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	op, err := h.uc.UpdateOrderpoint(r.Context(), id, req.ToUpdateInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToOrderpointResponse(op))
}

func (h *Handler) DeleteOrderpoint(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid orderpoint id", nil))
		return
	}

	if err := h.uc.DeleteOrderpoint(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// Suggestions returns the currently recommended reorder quantities without
// creating any purchase order.
func (h *Handler) OrderpointSuggestions(w http.ResponseWriter, r *http.Request) {
	items, err := h.uc.ReplenishmentSuggestions(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	if items == nil {
		items = []stockusecase.ReplenishmentItem{}
	}
	response.JSON(w, http.StatusOK, items)
}

// RunReplenishment executes every active rule and creates purchase proposals.
func (h *Handler) RunReplenishment(w http.ResponseWriter, r *http.Request) {
	items, err := h.uc.RunReplenishment(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	if items == nil {
		items = []stockusecase.ReplenishmentItem{}
	}
	response.JSON(w, http.StatusOK, items)
}
