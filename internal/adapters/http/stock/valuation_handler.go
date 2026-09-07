package stockhttp

import (
	"encoding/json"
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

// ─────────────────────────────────────────────────────────────────────────────
// Valuations (Phase 12 — stock-account integration)
// ─────────────────────────────────────────────────────────────────────────────

// GetValuationSummaries returns the current stock valuation snapshot.
func (h *Handler) GetValuationSummaries(w http.ResponseWriter, r *http.Request) {
	var productID, locationID *int64

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

	summaries, err := h.uc.GetValuationSummaries(r.Context(), productID, locationID)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]ValuationSummaryResponse, len(summaries))
	for i, s := range summaries {
		dtos[i] = ToValuationSummaryResponse(s)
	}
	if dtos == nil {
		dtos = []ValuationSummaryResponse{}
	}

	response.JSON(w, http.StatusOK, dtos)
}

// ListProductValues returns the manual valuation history.
func (h *Handler) ListProductValues(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()

	if pStr := strings.TrimSpace(r.URL.Query().Get("product_id")); pStr != "" {
		if pid, err := strconv.ParseInt(pStr, 10, 64); err == nil && pid > 0 {
			f.Add("product_id", filter.OpEqual, pid)
		}
	}
	if mStr := strings.TrimSpace(r.URL.Query().Get("move_id")); mStr != "" {
		if mid, err := strconv.ParseInt(mStr, 10, 64); err == nil && mid > 0 {
			f.Add("move_id", filter.OpEqual, mid)
		}
	}

	res, err := h.uc.ListProductValues(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]stock.ProductValue, len(res.Items))
	copy(dtos, res.Items)

	pageRes := pagination.NewPageResult(dtos, res.TotalItems, page)
	response.Paginated(w, http.StatusOK, dtos, pageRes)
}

// AdjustMoveValue overrides a move's valuation value manually.
func (h *Handler) AdjustMoveValue(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid move id", nil))
		return
	}

	var req AdjustMoveValueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	move, err := h.uc.AdjustMoveValue(r.Context(), req.ToInput(id))
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToMoveResponse(move))
}

// CreateAccountingPeriod opens a new closing period.
func (h *Handler) CreateAccountingPeriod(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountingPeriodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	p, err := h.uc.CreateAccountingPeriod(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToAccountingPeriodResponse(p))
}

// GetAccountingPeriod fetches a single closing period.
func (h *Handler) GetAccountingPeriod(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid period id", nil))
		return
	}

	p, err := h.uc.GetAccountingPeriod(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToAccountingPeriodResponse(p))
}

// ListAccountingPeriods returns a paginated list of closing periods.
func (h *Handler) ListAccountingPeriods(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()

	if state := strings.TrimSpace(r.URL.Query().Get("state")); state != "" {
		f.Add("state", filter.OpEqual, state)
	}
	if jStr := strings.TrimSpace(r.URL.Query().Get("journal_id")); jStr != "" {
		if jid, err := strconv.ParseInt(jStr, 10, 64); err == nil && jid > 0 {
			f.Add("journal_id", filter.OpEqual, jid)
		}
	}

	res, err := h.uc.ListAccountingPeriods(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]AccountingPeriodResponse, len(res.Items))
	for i := range res.Items {
		dtos[i] = ToAccountingPeriodResponse(&res.Items[i])
	}

	pageRes := pagination.NewPageResult(dtos, res.TotalItems, page)
	response.Paginated(w, http.StatusOK, dtos, pageRes)
}

// ClosePeriodValuation closes a period and posts the closing entry.
func (h *Handler) ClosePeriodValuation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid period id", nil))
		return
	}

	p, err := h.uc.ClosePeriodValuation(r.Context(), stockusecase.ClosePeriodValuationInput{PeriodID: id})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToAccountingPeriodResponse(p))
}