package mrphttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/mrp"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
	mrpusecase "cashflow_backend/internal/usecase/mrp"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	usecase *mrpusecase.Usecase
	logger  *slog.Logger
}

func NewHandler(usecase *mrpusecase.Usecase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		usecase: usecase,
		logger:  logger,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Workcenter Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateWorkcenter(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkcenterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}

	// TODO: Get real userID from context
	userID := int64(1)

	wc, err := h.usecase.CreateWorkcenter(r.Context(), userID, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, wc)
}

func (h *Handler) GetWorkcenter(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	wc, err := h.usecase.GetWorkcenter(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, wc)
}

func (h *Handler) ListWorkcenters(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	f := mrp.WorkcenterFilter{
		Search: q.Get("search"),
		Limit:  limit,
		Offset: offset,
	}

	res, total, err := h.usecase.ListWorkcenters(r.Context(), f)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"items": res,
		"total": total,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Workorders
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListWorkorders(w http.ResponseWriter, r *http.Request) {
	productionID, _ := strconv.ParseInt(r.URL.Query().Get("production_id"), 10, 64)
	if productionID == 0 {
		response.Error(w, platformerrors.BadRequest("production_id query param is required", nil))
		return
	}

	res, err := h.usecase.ListWorkorders(r.Context(), productionID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *Handler) StartWorkorder(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	userID := int64(1)
	wo, err := h.usecase.StartWorkorder(r.Context(), userID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, wo)
}

func (h *Handler) DoneWorkorder(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	userID := int64(1)

	var req struct {
		Duration float64 `json:"duration"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	wo, err := h.usecase.DoneWorkorder(r.Context(), userID, id, req.Duration)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, wo)
}

// ─────────────────────────────────────────────────────────────────────────────
// Unbuild Orders
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateUnbuild(w http.ResponseWriter, r *http.Request) {
	var req CreateUnbuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}

	userID := int64(1)
	uo, err := h.usecase.CreateUnbuild(r.Context(), userID, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, uo)
}

func (h *Handler) GetUnbuild(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	uo, err := h.usecase.GetUnbuild(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, uo)
}

// ─────────────────────────────────────────────────────────────────────────────
// Reporting
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) GetOEEReport(w http.ResponseWriter, r *http.Request) {
	companyID, _ := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
	if companyID == 0 {
		companyID = 1
	}
	res, err := h.usecase.GetWorkcenterOEEReport(r.Context(), companyID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

// ─────────────────────────────────────────────────────────────────────────────
// Production Orders (MO)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateProduction(w http.ResponseWriter, r *http.Request) {
	var req CreateProductionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}

	userID := int64(1)
	mo, err := h.usecase.CreateProduction(r.Context(), userID, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, mo)
}

func (h *Handler) ConfirmProduction(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	userID := int64(1)
	mo, err := h.usecase.ConfirmProduction(r.Context(), userID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, mo)
}

func (h *Handler) ProduceProduction(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	userID := int64(1)
	mo, err := h.usecase.ProduceProduction(r.Context(), userID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, mo)
}

func (h *Handler) GetProduction(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	mo, err := h.usecase.GetProduction(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, mo)
}

func (h *Handler) ListProductions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	f := mrp.ProductionFilter{
		Limit:  limit,
		Offset: offset,
	}

	res, total, err := h.usecase.ListProductions(r.Context(), f)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"items": res,
		"total": total,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// BoM Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateBoM(w http.ResponseWriter, r *http.Request) {
	var req CreateBoMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON", err))
		return
	}

	userID := int64(1)
	bom, err := h.usecase.CreateBoM(r.Context(), userID, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, bom)
}

func (h *Handler) GetBoM(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	bom, err := h.usecase.GetBoM(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, bom)
}

func (h *Handler) ListBoMs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	f := mrp.BoMFilter{
		Limit:  limit,
		Offset: offset,
	}

	res, total, err := h.usecase.ListBoMs(r.Context(), f)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"items": res,
		"total": total,
	})
}
