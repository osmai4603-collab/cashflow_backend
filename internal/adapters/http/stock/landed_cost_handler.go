package stockhttp

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	stockusecase "cashflow_backend/internal/usecase/stock"
)

// ─────────────────────────────────────────────────────────────────────────────
// Landed Costs (Phase 13 — stock landed costs)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateLandedCost(w http.ResponseWriter, r *http.Request) {
	var req LandedCostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	lc, err := h.uc.CreateLandedCost(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToLandedCostResponse(lc))
}

func (h *Handler) GetLandedCost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid landed cost id", nil))
		return
	}

	lc, err := h.uc.GetLandedCost(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLandedCostResponse(lc))
}

func (h *Handler) ListLandedCosts(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()

	if state := r.URL.Query().Get("state"); state != "" {
		f.Add("state", filter.OpEqual, state)
	}
	if journalID := r.URL.Query().Get("journal_id"); journalID != "" {
		if v, err := strconv.ParseInt(journalID, 10, 64); err == nil && v > 0 {
			f.Add("journal_id", filter.OpEqual, v)
		}
	}
	if vendorBillID := r.URL.Query().Get("vendor_bill_id"); vendorBillID != "" {
		if v, err := strconv.ParseInt(vendorBillID, 10, 64); err == nil && v > 0 {
			f.Add("vendor_bill_id", filter.OpEqual, v)
		}
	}

	res, err := h.uc.ListLandedCosts(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	dtos := make([]LandedCostResponse, len(res.Items))
	for i := range res.Items {
		dtos[i] = ToLandedCostResponse(&res.Items[i])
	}

	pageRes := pagination.NewPageResult(dtos, res.TotalItems, page)
	response.Paginated(w, http.StatusOK, dtos, pageRes)
}

func (h *Handler) UpdateLandedCost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid landed cost id", nil))
		return
	}

	var req LandedCostUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	lc, err := h.uc.UpdateLandedCost(r.Context(), id, req.ToUpdateInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLandedCostResponse(lc))
}

func (h *Handler) DeleteLandedCost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid landed cost id", nil))
		return
	}

	if err := h.uc.DeleteLandedCost(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// ComputeLandedCost reallocates the cost lines across the received moves.
func (h *Handler) ComputeLandedCost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid landed cost id", nil))
		return
	}

	lc, err := h.uc.ComputeLandedCost(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLandedCostResponse(lc))
}

// ValidateLandedCost posts the valuation entry and revalues the moves.
func (h *Handler) ValidateLandedCost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid landed cost id", nil))
		return
	}

	lc, err := h.uc.ValidateLandedCost(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLandedCostResponse(lc))
}

// CancelLandedCost cancels a draft landed cost.
func (h *Handler) CancelLandedCost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid landed cost id", nil))
		return
	}

	lc, err := h.uc.CancelLandedCost(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLandedCostResponse(lc))
}

// CreateLandedCostFromVendorBill drafts a landed cost from a vendor bill's
// billable product lines (POST /invoices/{id}/create-landed-cost).
func (h *Handler) CreateLandedCostFromVendorBill(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, platformerrors.BadRequest("invalid invoice id", nil))
		return
	}

	var req CreateLandedCostFromBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("malformed request body", err))
		return
	}

	lc, err := h.uc.CreateLandedCostFromVendorBill(r.Context(), stockusecase.CreateLandedCostFromBillInput{
		VendorBillID: id,
		PickingIDs:   req.PickingIDs,
		Description:  req.Description,
		CompanyID:    req.CompanyID,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToLandedCostResponse(lc))
}
