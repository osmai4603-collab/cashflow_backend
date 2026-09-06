package stockhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"cashflow_backend/internal/domain/stock"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

// Handler handles HTTP requests for warehouses, stock locations, pickings, quants, and moves.
type Handler struct {
	uc     *stockusecase.UseCase
	logger *slog.Logger
}

// NewHandler initializes a new stock Handler.
func NewHandler(uc *stockusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		uc:     uc,
		logger: logger,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Locations
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	var req CreateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	loc, err := h.uc.CreateLocation(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToLocationResponse(loc))
}

func (h *Handler) GetLocation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid location id", nil))
		return
	}

	loc, err := h.uc.GetLocation(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLocationResponse(loc))
}

func (h *Handler) ListLocations(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()

	if usage := strings.TrimSpace(r.URL.Query().Get("usage")); usage != "" {
		f.Add("usage", filter.OpEqual, usage)
	}
	if search := strings.TrimSpace(r.URL.Query().Get("search")); search != "" {
		f.Add("search", filter.OpILike, search)
	}

	res, err := h.uc.ListLocations(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]LocationResponse, len(res.Items))
	for i := range res.Items {
		dtos[i] = ToLocationResponse(&res.Items[i])
	}

	pageRes := pagination.NewPageResult(dtos, res.TotalItems, page)
	response.Paginated(w, http.StatusOK, dtos, pageRes)
}

func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid location id", nil))
		return
	}

	var req UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	loc, err := h.uc.UpdateLocation(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLocationResponse(loc))
}

func (h *Handler) DeleteLocation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid location id", nil))
		return
	}

	if err := h.uc.DeleteLocation(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// ─────────────────────────────────────────────────────────────────────────────
// Warehouses
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateWarehouse(w http.ResponseWriter, r *http.Request) {
	var req CreateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	wh, err := h.uc.CreateWarehouse(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToWarehouseResponse(wh))
}

func (h *Handler) GetWarehouse(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid warehouse id", nil))
		return
	}

	wh, err := h.uc.GetWarehouse(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToWarehouseResponse(wh))
}

func (h *Handler) ListWarehouses(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()

	if code := strings.TrimSpace(r.URL.Query().Get("code")); code != "" {
		f.Add("code", filter.OpEqual, code)
	}
	if search := strings.TrimSpace(r.URL.Query().Get("search")); search != "" {
		f.Add("search", filter.OpILike, search)
	}

	res, err := h.uc.ListWarehouses(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]WarehouseResponse, len(res.Items))
	for i := range res.Items {
		dtos[i] = ToWarehouseResponse(&res.Items[i])
	}

	pageRes := pagination.NewPageResult(dtos, res.TotalItems, page)
	response.Paginated(w, http.StatusOK, dtos, pageRes)
}

func (h *Handler) UpdateWarehouse(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid warehouse id", nil))
		return
	}

	var req UpdateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	wh, err := h.uc.UpdateWarehouse(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToWarehouseResponse(wh))
}

func (h *Handler) DeleteWarehouse(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid warehouse id", nil))
		return
	}

	if err := h.uc.DeleteWarehouse(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// ─────────────────────────────────────────────────────────────────────────────
// Pickings
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreatePicking(w http.ResponseWriter, r *http.Request) {
	var req CreatePickingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	picking, err := h.uc.CreatePicking(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToPickingResponse(picking))
}

func (h *Handler) GetPicking(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid picking id", nil))
		return
	}

	picking, err := h.uc.GetPicking(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPickingResponse(picking))
}

func (h *Handler) ListPickings(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()

	if pt := strings.TrimSpace(r.URL.Query().Get("picking_type")); pt != "" {
		f.Add("picking_type", filter.OpEqual, pt)
	}
	if state := strings.TrimSpace(r.URL.Query().Get("state")); state != "" {
		f.Add("state", filter.OpEqual, state)
	}
	if partnerIDStr := strings.TrimSpace(r.URL.Query().Get("partner_id")); partnerIDStr != "" {
		if pid, err := strconv.ParseInt(partnerIDStr, 10, 64); err == nil {
			f.Add("partner_id", filter.OpEqual, pid)
		}
	}
	if origin := strings.TrimSpace(r.URL.Query().Get("origin")); origin != "" {
		f.Add("origin", filter.OpILike, origin)
	}
	if search := strings.TrimSpace(r.URL.Query().Get("search")); search != "" {
		f.Add("search", filter.OpILike, search)
	}

	res, err := h.uc.ListPickings(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]PickingResponse, len(res.Items))
	for i := range res.Items {
		dtos[i] = ToPickingResponse(&res.Items[i])
	}

	pageRes := pagination.NewPageResult(dtos, res.TotalItems, page)
	response.Paginated(w, http.StatusOK, dtos, pageRes)
}

func (h *Handler) UpdatePicking(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid picking id", nil))
		return
	}

	var req UpdatePickingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	picking, err := h.uc.UpdatePicking(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPickingResponse(picking))
}

func (h *Handler) DeletePicking(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid picking id", nil))
		return
	}

	if err := h.uc.DeletePicking(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

func (h *Handler) ConfirmPicking(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid picking id", nil))
		return
	}

	picking, err := h.uc.ConfirmPicking(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPickingResponse(picking))
}

func (h *Handler) ValidatePicking(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid picking id", nil))
		return
	}

	var req ValidatePickingRequest
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	picking, err := h.uc.ValidatePicking(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPickingResponse(picking))
}

func (h *Handler) CancelPicking(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid picking id", nil))
		return
	}

	picking, err := h.uc.CancelPicking(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPickingResponse(picking))
}

// ─────────────────────────────────────────────────────────────────────────────
// Balances & Moves & Adjustments
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) GetOnHandStock(w http.ResponseWriter, r *http.Request) {
	var productID, locationID, warehouseID *int64

	if pStr := strings.TrimSpace(r.URL.Query().Get("product_id")); pStr != "" {
		if pid, err := strconv.ParseInt(pStr, 10, 64); err == nil && pid > 0 {
			productID = &pid
		}
	}
	if lStr := strings.TrimSpace(r.URL.Query().Get("location_id")); lStr != "" {
		if lid, err := strconv.ParseInt(lStr, 10, 64); err == nil && lid > 0 {
			locationID = &lid
		}
	}
	if wStr := strings.TrimSpace(r.URL.Query().Get("warehouse_id")); wStr != "" {
		if wid, err := strconv.ParseInt(wStr, 10, 64); err == nil && wid > 0 {
			warehouseID = &wid
		}
	}

	items, err := h.uc.GetOnHandStock(r.Context(), productID, locationID, warehouseID)
	if err != nil {
		response.Error(w, err)
		return
	}

	if items == nil {
		items = []stock.StockOnHandItem{}
	}

	response.JSON(w, http.StatusOK, items)
}

func (h *Handler) ListMoves(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()

	if pStr := strings.TrimSpace(r.URL.Query().Get("product_id")); pStr != "" {
		if pid, err := strconv.ParseInt(pStr, 10, 64); err == nil && pid > 0 {
			f.Add("product_id", filter.OpEqual, pid)
		}
	}
	if pickStr := strings.TrimSpace(r.URL.Query().Get("picking_id")); pickStr != "" {
		if pickID, err := strconv.ParseInt(pickStr, 10, 64); err == nil && pickID > 0 {
			f.Add("picking_id", filter.OpEqual, pickID)
		}
	}
	if state := strings.TrimSpace(r.URL.Query().Get("state")); state != "" {
		f.Add("state", filter.OpEqual, state)
	}

	res, err := h.uc.ListMoves(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]MoveResponse, len(res.Items))
	for i := range res.Items {
		dtos[i] = ToMoveResponse(&res.Items[i])
	}

	pageRes := pagination.NewPageResult(dtos, res.TotalItems, page)
	response.Paginated(w, http.StatusOK, dtos, pageRes)
}

func (h *Handler) AdjustStock(w http.ResponseWriter, r *http.Request) {
	var req StockAdjustmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	quant, err := h.uc.AdjustStock(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToQuantResponse(quant))
}
