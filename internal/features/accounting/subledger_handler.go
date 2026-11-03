package accounting

import (
	"errors"
	"log/slog"
	"net/http"
)

type SubledgerHandler struct {
	uc     *UseCases
	logger *slog.Logger
}

func NewSubledgerHandler(uc *UseCases, logger *slog.Logger) *SubledgerHandler {
	return &SubledgerHandler{uc: uc, logger: handlerLogger(logger, "subledger")}
}

func (h *SubledgerHandler) HandleIngest(w http.ResponseWriter, r *http.Request) {
	var request IngestSubledgerEventRequest
	if err := decodeJSON(r, &request); err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	if err := request.Validate(); err != nil {
		writeValidationError(w, r, h.logger, err)
		return
	}
	if _, ok := subledgerMappings[request.EventType]; !ok {
		writeValidationError(w, r, h.logger, errors.New("unknown subledger event_type"))
		return
	}

	response, err := h.uc.IngestSubledgerEvent.Execute(r.Context(), request)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}
