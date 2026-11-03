package accounting

import (
	"log/slog"
	"net/http"

	platformerr "cashflow_backend/internal/platform/errors"
)

type PeriodHandler struct {
	uc     *UseCases
	logger *slog.Logger
}

func NewPeriodHandler(uc *UseCases, logger *slog.Logger) *PeriodHandler {
	return &PeriodHandler{uc: uc, logger: handlerLogger(logger, "period")}
}

func (h *PeriodHandler) HandleCreateYear(w http.ResponseWriter, r *http.Request) {
	var request CreateFiscalYearRequest
	if err := decodeJSON(r, &request); err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	if err := request.Validate(); err != nil {
		writeValidationError(w, r, h.logger, err)
		return
	}

	response, err := h.uc.CreateFiscalYear.Execute(r.Context(), request)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *PeriodHandler) HandleChangeState(w http.ResponseWriter, r *http.Request) {
	periodID, err := parseRouteUUID(r, "id")
	if err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	var request ChangePeriodStateRequest
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

	response, err := h.uc.ChangePeriodState.Execute(r.Context(), periodID, actorID, request)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}
