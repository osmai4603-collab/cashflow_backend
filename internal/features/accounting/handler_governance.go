package accounting

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
)

type GovernanceHandler struct {
	uc     *UseCases
	logger *slog.Logger
}

func NewGovernanceHandler(uc *UseCases, logger *slog.Logger) *GovernanceHandler {
	return &GovernanceHandler{uc: uc, logger: handlerLogger(logger, "governance")}
}

func (h *GovernanceHandler) HandleVerifyIntegrity(w http.ResponseWriter, r *http.Request) {
	response, err := h.uc.VerifyAuditIntegrity.Execute(r.Context())
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *GovernanceHandler) HandleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit := 50
	offset := 0
	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 200 {
			writeBadRequest(w, r, h.logger, errors.New("limit must be an integer between 1 and 200"))
			return
		}
		limit = parsed
	}
	if value := query.Get("offset"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			writeBadRequest(w, r, h.logger, errors.New("offset must be a non-negative integer"))
			return
		}
		offset = parsed
	}

	response, err := h.uc.ListAuditLogs.Execute(r.Context(), limit, offset)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

