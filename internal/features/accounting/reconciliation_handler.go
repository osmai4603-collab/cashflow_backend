package accounting

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	platformerr "cashflow_backend/internal/platform/errors"

	"github.com/google/uuid"
)

type ReconciliationHandler struct {
	uc     *UseCases
	logger *slog.Logger
}

func NewReconciliationHandler(uc *UseCases, logger *slog.Logger) *ReconciliationHandler {
	return &ReconciliationHandler{uc: uc, logger: handlerLogger(logger, "reconciliation")}
}

func (h *ReconciliationHandler) HandleReconcile(w http.ResponseWriter, r *http.Request) {
	var request ReconcileMoveLinesRequest
	if err := decodeJSON(r, &request); err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	if err := request.Validate(); err != nil {
		writeValidationError(w, r, h.logger, err)
		return
	}
	actorID, err := actorIDFromRequest(r)
	if err != nil {
		platformerr.WriteHTTPError(w, r, platformerr.Unauthorized("accounting.actor", "invalid authenticated user", err), h.logger)
		return
	}
	response, err := h.uc.ReconcileMoveLines.Execute(r.Context(), actorID, request)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *ReconciliationHandler) HandleUnreconcile(w http.ResponseWriter, r *http.Request) {
	var request UnreconcileMoveLinesRequest
	if err := decodeJSON(r, &request); err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	if err := request.Validate(); err != nil {
		writeValidationError(w, r, h.logger, err)
		return
	}
	actorID, err := actorIDFromRequest(r)
	if err != nil {
		platformerr.WriteHTTPError(w, r, platformerr.Unauthorized("accounting.actor", "invalid authenticated user", err), h.logger)
		return
	}
	if err := h.uc.UnreconcileMoveLines.Execute(r.Context(), actorID, request); err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ReconciliationHandler) HandleListUnreconciled(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := UnreconciledLineFilter{CurrencyID: query.Get("currency"), Limit: 50}
	if value := query.Get("account_id"); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			writeBadRequest(w, r, h.logger, errors.New("account_id must be a UUID"))
			return
		}
		filter.AccountID = &id
	}
	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			writeBadRequest(w, r, h.logger, errors.New("limit must be an integer between 1 and 200"))
			return
		}
		filter.Limit = limit
	}
	if value := query.Get("offset"); value != "" {
		offset, err := strconv.Atoi(value)
		if err != nil {
			writeBadRequest(w, r, h.logger, errors.New("offset must be a non-negative integer"))
			return
		}
		filter.Offset = offset
	}
	lines, total, err := h.uc.ListUnreconciledLines.Execute(r.Context(), filter)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"lines": lines, "total": total, "limit": filter.Limit, "offset": filter.Offset,
	})
}
