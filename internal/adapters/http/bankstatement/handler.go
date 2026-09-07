package bankstatementhttp

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
	bankstatementusecase "cashflow_backend/internal/usecase/bankstatement"

	"github.com/go-chi/chi/v5"
)

// Handler serves REST API endpoints for the Bank Statement domain.
type Handler struct {
	useCase *bankstatementusecase.UseCase
	logger  *slog.Logger
}

// NewHandler constructs a new Bank Statement HTTP Handler.
func NewHandler(useCase *bankstatementusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func idParam(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, name), 10, 64)
}

// ─────────────────────────────────────────────────────────────────────────────
// Statements
// ─────────────────────────────────────────────────────────────────────────────

// CreateStatement handles POST /api/v1/bank-statements
func (h *Handler) CreateStatement(w http.ResponseWriter, r *http.Request) {
	var req CreateStatementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	st, err := h.useCase.CreateStatement(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, ToStatementResponse(st))
}

// GetStatement handles GET /api/v1/bank-statements/{id}
func (h *Handler) GetStatement(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid statement ID", err))
		return
	}
	st, err := h.useCase.GetStatement(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToStatementResponse(st))
}

// UpdateStatement handles PUT /api/v1/bank-statements/{id}
func (h *Handler) UpdateStatement(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid statement ID", err))
		return
	}
	var req UpdateStatementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	st, err := h.useCase.UpdateStatement(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToStatementResponse(st))
}

// DeleteStatement handles DELETE /api/v1/bank-statements/{id}
func (h *Handler) DeleteStatement(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid statement ID", err))
		return
	}
	if err := h.useCase.DeleteStatement(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// ListStatements handles GET /api/v1/bank-statements
func (h *Handler) ListStatements(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	f := filter.NewFilter()
	res, err := h.useCase.ListStatements(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]StatementResponse, 0, len(res.Items))
	for i := range res.Items {
		items = append(items, ToStatementResponse(&res.Items[i]))
	}
	response.Paginated(w, http.StatusOK, items, res)
}

// AddLines handles POST /api/v1/bank-statements/{id}/lines
func (h *Handler) AddLines(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid statement ID", err))
		return
	}
	var req AddLinesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	inputs := make([]bankstatementusecase.AddStatementLineInput, 0, len(req.Lines))
	for _, l := range req.Lines {
		inputs = append(inputs, l.ToInput())
	}
	lines, err := h.useCase.AddLines(r.Context(), id, inputs)
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]StatementLineResponse, 0, len(lines))
	for i := range lines {
		items = append(items, ToStatementLineResponse(&lines[i]))
	}
	response.Created(w, items)
}

// ConfirmStatement handles POST /api/v1/bank-statements/{id}/confirm
func (h *Handler) ConfirmStatement(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid statement ID", err))
		return
	}
	st, err := h.useCase.ConfirmStatement(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToStatementResponse(st))
}

// AutoReconcile handles POST /api/v1/bank-statements/{id}/auto-reconcile
func (h *Handler) AutoReconcile(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid statement ID", err))
		return
	}
	count, err := h.useCase.AutoReconcile(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"statement_id": id, "reconciled_lines": count})
}

// ImportCSV handles POST /api/v1/bank-statements/{id}/import
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid statement ID", err))
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("could not read CSV body", err))
		return
	}
	lines, err := h.useCase.ImportCSV(r.Context(), id, data)
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]StatementLineResponse, 0, len(lines))
	for i := range lines {
		items = append(items, ToStatementLineResponse(&lines[i]))
	}
	response.Created(w, items)
}

// ─────────────────────────────────────────────────────────────────────────────
// Statement Lines & Reconciliation
// ─────────────────────────────────────────────────────────────────────────────

// FindCandidates handles GET /api/v1/bank-statement-lines/{id}/candidates
func (h *Handler) FindCandidates(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid statement line ID", err))
		return
	}
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	candidates, err := h.useCase.FindCandidates(r.Context(), id, limit)
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]CandidateResponse, 0, len(candidates))
	for _, c := range candidates {
		items = append(items, CandidateResponse{
			MoveLine:        c.MoveLine,
			SuggestedAmount: c.SuggestedAmount,
			ExactMatch:      c.ExactMatch,
		})
	}
	response.JSON(w, http.StatusOK, items)
}

// ReconcileLine handles POST /api/v1/bank-statement-lines/{id}/reconcile
func (h *Handler) ReconcileLine(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid statement line ID", err))
		return
	}
	var req ReconcileLineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	result, err := h.useCase.ReconcileLine(r.Context(), id, req.CandidateLineID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToReconcileResultResponse(result))
}

// UndoReconcile handles POST /api/v1/bank-statement-lines/{id}/undo-reconcile
func (h *Handler) UndoReconcile(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid statement line ID", err))
		return
	}
	if err := h.useCase.UndoReconcile(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// ─────────────────────────────────────────────────────────────────────────────
// Reconcile Models
// ─────────────────────────────────────────────────────────────────────────────

// CreateReconcileModel handles POST /api/v1/reconcile-models
func (h *Handler) CreateReconcileModel(w http.ResponseWriter, r *http.Request) {
	var req ReconcileModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	m, err := h.useCase.CreateReconcileModel(r.Context(), req.ToModel())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, ToReconcileModelResponse(m))
}

// GetReconcileModel handles GET /api/v1/reconcile-models/{id}
func (h *Handler) GetReconcileModel(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid reconcile model ID", err))
		return
	}
	m, err := h.useCase.GetReconcileModel(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToReconcileModelResponse(m))
}

// UpdateReconcileModel handles PUT /api/v1/reconcile-models/{id}
func (h *Handler) UpdateReconcileModel(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid reconcile model ID", err))
		return
	}
	var req ReconcileModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	m, err := h.useCase.UpdateReconcileModel(r.Context(), id, req.ToModel())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToReconcileModelResponse(m))
}

// DeleteReconcileModel handles DELETE /api/v1/reconcile-models/{id}
func (h *Handler) DeleteReconcileModel(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid reconcile model ID", err))
		return
	}
	if err := h.useCase.DeleteReconcileModel(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// ListReconcileModels handles GET /api/v1/reconcile-models
func (h *Handler) ListReconcileModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.useCase.ListReconcileModels(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]ReconcileModelResponse, 0, len(models))
	for i := range models {
		items = append(items, ToReconcileModelResponse(&models[i]))
	}
	response.JSON(w, http.StatusOK, items)
}

// ─────────────────────────────────────────────────────────────────────────────
// Cash Rounding
// ─────────────────────────────────────────────────────────────────────────────

// CreateCashRounding handles POST /api/v1/cash-roundings
func (h *Handler) CreateCashRounding(w http.ResponseWriter, r *http.Request) {
	var req CashRoundingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	cr, err := h.useCase.CreateCashRounding(r.Context(), req.ToCashRounding())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, ToCashRoundingResponse(cr))
}

// GetCashRounding handles GET /api/v1/cash-roundings/{id}
func (h *Handler) GetCashRounding(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid cash rounding ID", err))
		return
	}
	cr, err := h.useCase.GetCashRounding(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToCashRoundingResponse(cr))
}

// UpdateCashRounding handles PUT /api/v1/cash-roundings/{id}
func (h *Handler) UpdateCashRounding(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid cash rounding ID", err))
		return
	}
	var req CashRoundingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	cr, err := h.useCase.UpdateCashRounding(r.Context(), id, req.ToCashRounding())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ToCashRoundingResponse(cr))
}

// DeleteCashRounding handles DELETE /api/v1/cash-roundings/{id}
func (h *Handler) DeleteCashRounding(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r, "id")
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid cash rounding ID", err))
		return
	}
	if err := h.useCase.DeleteCashRounding(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// ListCashRoundings handles GET /api/v1/cash-roundings
func (h *Handler) ListCashRoundings(w http.ResponseWriter, r *http.Request) {
	roundings, err := h.useCase.ListCashRoundings(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	items := make([]CashRoundingResponse, 0, len(roundings))
	for i := range roundings {
		items = append(items, ToCashRoundingResponse(&roundings[i]))
	}
	response.JSON(w, http.StatusOK, items)
}
