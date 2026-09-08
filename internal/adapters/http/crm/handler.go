package crmhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	crmusecase "cashflow_backend/internal/usecase/crm"

	"github.com/go-chi/chi/v5"
)

// Handler serves HTTP endpoints for the CRM & Sales Pipeline domain.
type Handler struct {
	useCase *crmusecase.UseCase
	logger  *slog.Logger
}

// NewHandler constructs a new CRM HTTP Handler.
func NewHandler(useCase *crmusecase.UseCase, logger *slog.Logger) *Handler {
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

// ─────────────────────────────────────────────────────────────────────────────
// Leads & Opportunities
// ─────────────────────────────────────────────────────────────────────────────

// CreateLead handles POST /api/v1/leads
func (h *Handler) CreateLead(w http.ResponseWriter, r *http.Request) {
	var req CreateLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	lead, err := h.useCase.CreateLead(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, ToLeadResponse(lead))
}

// GetLead handles GET /api/v1/leads/{id}
func (h *Handler) GetLead(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid lead ID in path", err))
		return
	}

	lead, err := h.useCase.GetLead(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLeadResponse(lead))
}

// UpdateLead handles PUT /api/v1/leads/{id}
func (h *Handler) UpdateLead(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid lead ID in path", err))
		return
	}

	var req UpdateLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	lead, err := h.useCase.UpdateLead(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLeadResponse(lead))
}

// DeleteLead handles DELETE /api/v1/leads/{id}
func (h *Handler) DeleteLead(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid lead ID in path", err))
		return
	}

	if err := h.useCase.DeleteLead(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

// ListLeads handles GET /api/v1/leads
func (h *Handler) ListLeads(w http.ResponseWriter, r *http.Request) {
	pageReq := pagination.Parse(r)
	f := filter.NewFilter()
	q := r.URL.Query()

	if t := q.Get("type"); t != "" {
		f.Add("type", filter.OpEqual, t)
	}
	if stageID := q.Get("stage_id"); stageID != "" {
		f.Add("stage_id", filter.OpEqual, stageID)
	}
	if spID := q.Get("salesperson_id"); spID != "" {
		f.Add("salesperson_id", filter.OpEqual, spID)
	}
	if pID := q.Get("partner_id"); pID != "" {
		f.Add("partner_id", filter.OpEqual, pID)
	}
	if priority := q.Get("priority"); priority != "" {
		f.Add("priority", filter.OpEqual, priority)
	}
	if active := q.Get("active"); active != "" {
		f.Add("active", filter.OpEqual, active)
	}
	if search := q.Get("search"); search != "" {
		f.Add("name", filter.OpILike, "%"+search+"%")
	}

	pageRes, err := h.useCase.ListLeads(r.Context(), f, pageReq)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	items := make([]LeadResponse, len(pageRes.Items))
	for i := range pageRes.Items {
		items[i] = ToLeadResponse(&pageRes.Items[i])
	}

	res := pagination.NewPageResult(items, pageRes.TotalItems, pageReq)
	response.Paginated(w, http.StatusOK, items, res)
}

// ConvertLead handles POST /api/v1/leads/{id}/convert
func (h *Handler) ConvertLead(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid lead ID in path", err))
		return
	}

	var req ConvertLeadRequest
	if r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	lead, err := h.useCase.ConvertLead(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLeadResponse(lead))
}

// MarkWon handles POST /api/v1/leads/{id}/won
func (h *Handler) MarkWon(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid lead ID in path", err))
		return
	}

	var req MarkWonRequest
	if r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	lead, order, err := h.useCase.MarkLeadWon(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, MarkWonResponse{
		Lead:      ToLeadResponse(lead),
		SaleOrder: order,
	})
}

// MarkLost handles POST /api/v1/leads/{id}/lost
func (h *Handler) MarkLost(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid lead ID in path", err))
		return
	}

	var req MarkLostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	lead, err := h.useCase.MarkLeadLost(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLeadResponse(lead))
}

// ─────────────────────────────────────────────────────────────────────────────
// Pipeline & Analytics
// ─────────────────────────────────────────────────────────────────────────────

// GetPipeline handles GET /api/v1/crm/pipeline
func (h *Handler) GetPipeline(w http.ResponseWriter, r *http.Request) {
	var salespersonID *int64
	if spStr := r.URL.Query().Get("salesperson_id"); spStr != "" {
		if spVal, err := strconv.ParseInt(spStr, 10, 64); err == nil {
			salespersonID = &spVal
		}
	}

	pipelineData, err := h.useCase.GetPipelineView(r.Context(), salespersonID)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPipelineResponse(pipelineData))
}

// GetStats handles GET /api/v1/crm/stats
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.useCase.GetCRMStats(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, stats)
}

// ─────────────────────────────────────────────────────────────────────────────
// Stages Management
// ─────────────────────────────────────────────────────────────────────────────

// CreateStage handles POST /api/v1/crm/stages
func (h *Handler) CreateStage(w http.ResponseWriter, r *http.Request) {
	var req CreateStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	stage, err := h.useCase.CreateStage(r.Context(), crmusecase.CreateStageInput{
		Name:         req.Name,
		Sequence:     req.Sequence,
		IsWon:        req.IsWon,
		IsClosed:     req.IsClosed,
		Fold:         req.Fold,
		Requirements: req.Requirements,
		CompanyID:    req.CompanyID,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, ToStageResponse(stage))
}

// GetStage handles GET /api/v1/crm/stages/{id}
func (h *Handler) GetStage(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid stage ID in path", err))
		return
	}

	stage, err := h.useCase.GetStage(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToStageResponse(stage))
}

// UpdateStage handles PUT /api/v1/crm/stages/{id}
func (h *Handler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid stage ID in path", err))
		return
	}

	var req UpdateStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	stage, err := h.useCase.UpdateStage(r.Context(), id, crmusecase.UpdateStageInput{
		Name:         req.Name,
		Sequence:     req.Sequence,
		IsWon:        req.IsWon,
		IsClosed:     req.IsClosed,
		Fold:         req.Fold,
		Requirements: req.Requirements,
		CompanyID:    req.CompanyID,
		Active:       req.Active,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToStageResponse(stage))
}

// DeleteStage handles DELETE /api/v1/crm/stages/{id}
func (h *Handler) DeleteStage(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid stage ID in path", err))
		return
	}

	if err := h.useCase.DeleteStage(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

// ListStages handles GET /api/v1/crm/stages
func (h *Handler) ListStages(w http.ResponseWriter, r *http.Request) {
	stages, err := h.useCase.ListStages(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	res := make([]StageResponse, len(stages))
	for i := range stages {
		res[i] = ToStageResponse(&stages[i])
	}

	response.JSON(w, http.StatusOK, res)
}

// ─────────────────────────────────────────────────────────────────────────────
// Lost Reasons Management
// ─────────────────────────────────────────────────────────────────────────────

// CreateLostReason handles POST /api/v1/crm/lost-reasons
func (h *Handler) CreateLostReason(w http.ResponseWriter, r *http.Request) {
	var req CreateLostReasonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	reason, err := h.useCase.CreateLostReason(r.Context(), crmusecase.CreateLostReasonInput{Name: req.Name})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, ToLostReasonResponse(reason))
}

// GetLostReason handles GET /api/v1/crm/lost-reasons/{id}
func (h *Handler) GetLostReason(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid lost reason ID in path", err))
		return
	}

	reason, err := h.useCase.GetLostReason(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLostReasonResponse(reason))
}

// UpdateLostReason handles PUT /api/v1/crm/lost-reasons/{id}
func (h *Handler) UpdateLostReason(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid lost reason ID in path", err))
		return
	}

	var req UpdateLostReasonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	reason, err := h.useCase.UpdateLostReason(r.Context(), id, crmusecase.UpdateLostReasonInput{
		Name:   req.Name,
		Active: req.Active,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToLostReasonResponse(reason))
}

// DeleteLostReason handles DELETE /api/v1/crm/lost-reasons/{id}
func (h *Handler) DeleteLostReason(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid lost reason ID in path", err))
		return
	}

	if err := h.useCase.DeleteLostReason(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

// ListLostReasons handles GET /api/v1/crm/lost-reasons
func (h *Handler) ListLostReasons(w http.ResponseWriter, r *http.Request) {
	reasons, err := h.useCase.ListLostReasons(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	res := make([]LostReasonResponse, len(reasons))
	for i := range reasons {
		res[i] = ToLostReasonResponse(&reasons[i])
	}

	response.JSON(w, http.StatusOK, res)
}

// ─────────────────────────────────────────────────────────────────────────────
// Tags Management
// ─────────────────────────────────────────────────────────────────────────────

// CreateTag handles POST /api/v1/crm/tags
func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	tag, err := h.useCase.CreateTag(r.Context(), crmusecase.CreateTagInput{
		Name:  req.Name,
		Color: req.Color,
	})
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, ToTagResponse(tag))
}

// ListTags handles GET /api/v1/crm/tags
func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.useCase.ListTags(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	res := make([]TagResponse, len(tags))
	for i := range tags {
		res[i] = ToTagResponse(&tags[i])
	}

	response.JSON(w, http.StatusOK, res)
}
