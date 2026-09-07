package analytichttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cashflow_backend/internal/domain/analytic"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	analyticusecase "cashflow_backend/internal/usecase/analytic"

	"github.com/go-chi/chi/v5"
)

// Handler serves HTTP requests for the Analytic Accounting domain.
type Handler struct {
	useCase *analyticusecase.UseCase
	logger  *slog.Logger
}

// NewHandler constructs an analytic Handler.
func NewHandler(useCase *analyticusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func parseID(param string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(param), 10, 64)
}

func parseDateParam(val string) *time.Time {
	if val == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", val)
	if err != nil {
		return nil
	}
	return &t
}

func parseCompanyID(r *http.Request) int64 {
	id, _ := parseID(r.URL.Query().Get("company_id"))
	return id
}

func parseIDList(val string) []int64 {
	if strings.TrimSpace(val) == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		if id, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// ─────────────────────────────────────────────────────────────────────────────
// Plans Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var req CreatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	plan, err := h.useCase.CreatePlan(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, ToPlanResponse(plan))
}

func (h *Handler) GetPlan(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid plan ID in path", err))
		return
	}
	plan, err := h.useCase.GetPlan(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToPlanResponse(plan))
}

func (h *Handler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid plan ID in path", err))
		return
	}
	var req UpdatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	plan, err := h.useCase.UpdatePlan(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToPlanResponse(plan))
}

func (h *Handler) DeletePlan(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid plan ID in path", err))
		return
	}
	if err := h.useCase.DeletePlan(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) ListPlans(w http.ResponseWriter, r *http.Request) {
	includeInactive := r.URL.Query().Get("include_inactive") == "true"
	plans, err := h.useCase.ListPlans(r.Context(), includeInactive)
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]PlanResponse, len(plans))
	for i := range plans {
		items[i] = ToPlanResponse(&plans[i])
	}
	response.JSON(w, http.StatusOK, items)
}

func (h *Handler) GetPlanStructure(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid plan ID in path", err))
		return
	}
	structure, err := h.useCase.GetPlanStructure(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToPlanStructureResponse(structure))
}

// SetApplicability handles PUT /api/v1/analytic-plans/{id}/applicability (G2).
func (h *Handler) SetApplicability(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid plan ID in path", err))
		return
	}
	var req SetApplicabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	rule, err := h.useCase.SetApplicability(r.Context(), req.ToInput(id))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToApplicabilityResponse(*rule))
}

func (h *Handler) GetRelevantPlans(w http.ResponseWriter, r *http.Request) {
	companyID := parseCompanyID(r)
	businessDomain := analytic.BusinessDomain(r.URL.Query().Get("business_domain"))
	if businessDomain == "" {
		businessDomain = analytic.DomainGeneral
	}
	forced := parseIDList(r.URL.Query().Get("forced_plan_ids"))
	plans, err := h.useCase.GetRelevantPlans(r.Context(), companyID, businessDomain, forced)
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]RelevantPlanResponse, len(plans))
	for i, p := range plans {
		items[i] = RelevantPlanResponse{
			ID:            p.ID,
			Name:          p.Name,
			Applicability: string(p.Applicability),
			ColumnName:    p.ColumnName,
		}
	}
	response.JSON(w, http.StatusOK, items)
}

// ─────────────────────────────────────────────────────────────────────────────
// Accounts Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	acc, err := h.useCase.CreateAccount(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, ToAccountResponse(acc))
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid account ID in path", err))
		return
	}
	acc, err := h.useCase.GetAccount(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToAccountResponse(acc))
}

func (h *Handler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid account ID in path", err))
		return
	}
	var req UpdateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	acc, err := h.useCase.UpdateAccount(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToAccountResponse(acc))
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid account ID in path", err))
		return
	}
	if err := h.useCase.DeleteAccount(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	result, err := h.useCase.ListAccounts(r.Context(), parseFilterFromQuery(r), page)
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]AccountResponse, len(result.Items))
	for i := range result.Items {
		items[i] = ToAccountResponse(&result.Items[i])
	}
	response.Paginated(w, http.StatusOK, items, result)
}

// GetAccountBalance handles GET /api/v1/analytic-accounts/{id}/balance.
func (h *Handler) GetAccountBalance(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid account ID in path", err))
		return
	}
	balance, err := h.useCase.GetAccountBalance(r.Context(), id,
		parseDateParam(r.URL.Query().Get("date_from")),
		parseDateParam(r.URL.Query().Get("date_to")),
	)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, BalanceResponse{
		Debit:    balance.Debit,
		Credit:   balance.Credit,
		Balance:  balance.Balance,
		Currency: balance.Currency,
	})
}

// ListAccountLines handles GET /api/v1/analytic-accounts/{id}/lines.
func (h *Handler) ListAccountLines(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid account ID in path", err))
		return
	}
	page := pagination.Parse(r)
	cf := filter.NewFilter(filter.Criterion{Field: "account_id", Operator: filter.OpEqual, Value: id})
	result, err := h.useCase.ListLines(r.Context(), cf, page)
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]LineResponse, len(result.Items))
	for i := range result.Items {
		items[i] = ToLineResponse(&result.Items[i])
	}
	response.Paginated(w, http.StatusOK, items, result)
}

// ─────────────────────────────────────────────────────────────────────────────
// Lines Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) RegisterManualLine(w http.ResponseWriter, r *http.Request) {
	var req CreateLineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	line, err := h.useCase.RegisterManualLine(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, ToLineResponse(line))
}

func (h *Handler) GetLine(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid line ID in path", err))
		return
	}
	line, err := h.useCase.GetLine(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToLineResponse(line))
}

func (h *Handler) UpdateLine(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid line ID in path", err))
		return
	}
	var req UpdateLineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	line, err := h.useCase.UpdateLine(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToLineResponse(line))
}

func (h *Handler) ListLines(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	result, err := h.useCase.ListLines(r.Context(), parseFilterFromQuery(r), page)
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]LineResponse, len(result.Items))
	for i := range result.Items {
		items[i] = ToLineResponse(&result.Items[i])
	}
	response.Paginated(w, http.StatusOK, items, result)
}

// ─────────────────────────────────────────────────────────────────────────────
// Distribution Model Handlers (G7)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) CreateDistributionModel(w http.ResponseWriter, r *http.Request) {
	var req CreateDistributionModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	model, err := h.useCase.CreateDistributionModel(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, ToDistributionModelResponse(model))
}

func (h *Handler) GetDistributionModel(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid distribution model ID in path", err))
		return
	}
	model, err := h.useCase.GetDistributionModel(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToDistributionModelResponse(model))
}

func (h *Handler) UpdateDistributionModel(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid distribution model ID in path", err))
		return
	}
	var req UpdateDistributionModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	model, err := h.useCase.UpdateDistributionModel(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToDistributionModelResponse(model))
}

func (h *Handler) DeleteDistributionModel(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid distribution model ID in path", err))
		return
	}
	if err := h.useCase.DeleteDistributionModel(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) ListDistributionModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.useCase.ListDistributionModels(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]DistributionModelResponse, len(models))
	for i := range models {
		items[i] = ToDistributionModelResponse(&models[i])
	}
	response.JSON(w, http.StatusOK, items)
}

// MatchDistributionModel handles POST /api/v1/analytic-distribution-models/match.
func (h *Handler) MatchDistributionModel(w http.ResponseWriter, r *http.Request) {
	var req MatchDistributionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	matched, _, err := h.useCase.AutoCompleteDistribution(r.Context(),
		req.PartnerID, req.PartnerCategoryID, req.CompanyID, nil)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, matched)
}

// ─────────────────────────────────────────────────────────────────────────────
// Move-Line Integration Handlers
// ─────────────────────────────────────────────────────────────────────────────

// DistributeMoveLine handles POST /api/v1/move-lines/{id}/analytic.
func (h *Handler) DistributeMoveLine(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid move line ID in path", err))
		return
	}
	var req DistributeMoveLineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	lines, err := h.useCase.CreateLinesFromMoveLine(r.Context(), req.ToInput(id))
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]LineResponse, len(lines))
	for i := range lines {
		items[i] = ToLineResponse(&lines[i])
	}
	response.Created(w, items)
}

// ListMoveLineAnalytic handles GET /api/v1/move-lines/{id}/analytic.
func (h *Handler) ListMoveLineAnalytic(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid move line ID in path", err))
		return
	}
	lines, err := h.useCase.ListLinesByMoveLine(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]LineResponse, len(lines))
	for i := range lines {
		items[i] = ToLineResponse(&lines[i])
	}
	response.JSON(w, http.StatusOK, items)
}

// ─────────────────────────────────────────────────────────────────────────────
// Project Plan Config (G8)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) GetProjectPlan(w http.ResponseWriter, r *http.Request) {
	planID, err := h.useCase.GetProjectPlanID(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ProjectPlanResponse{PlanID: planID})
}

func (h *Handler) SetProjectPlan(w http.ResponseWriter, r *http.Request) {
	var req ProjectPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	if err := h.useCase.SetProjectPlanID(r.Context(), req.PlanID); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ProjectPlanResponse{PlanID: req.PlanID})
}

// ─────────────────────────────────────────────────────────────────────────────
// Filtering helper
// ─────────────────────────────────────────────────────────────────────────────

func parseFilterFromQuery(r *http.Request) *filter.Filter {
	q := r.URL.Query()
	f := filter.NewFilter()

	if v := q.Get("plan_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.Add("plan_id", filter.OpEqual, id)
		}
	}
	if v := q.Get("account_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.Add("account_id", filter.OpEqual, id)
		}
	}
	if v := q.Get("partner_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.Add("partner_id", filter.OpEqual, id)
		}
	}
	if v := q.Get("user_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.Add("user_id", filter.OpEqual, id)
		}
	}
	if v := q.Get("company_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.Add("company_id", filter.OpEqual, id)
		}
	}
	if v := q.Get("source"); v != "" {
		f.Add("source", filter.OpEqual, strings.ToLower(v))
	}
	if v := q.Get("date_from"); v != "" {
		f.Add("date_from", filter.OpGreaterThanOrEqual, v)
	}
	if v := q.Get("date_to"); v != "" {
		f.Add("date_to", filter.OpLessThanOrEqual, v)
	}
	if v := q.Get("name"); v != "" {
		f.Add("name", filter.OpILike, strings.ToLower(v))
	}
	if v := q.Get("active"); v != "" {
		f.Add("active", filter.OpEqual, v)
	}
	return f
}
