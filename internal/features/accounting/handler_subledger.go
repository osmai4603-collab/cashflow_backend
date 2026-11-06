package accounting

import (
	"log/slog"
	"net/http"
	"strconv"
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
	response, err := h.uc.IngestSubledgerEvent.Execute(r.Context(), request)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *SubledgerHandler) HandleListMappings(w http.ResponseWriter, r *http.Request) {
	response, err := h.uc.ListSubledgerMappings.Execute(r.Context())
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *SubledgerHandler) HandleSaveMapping(w http.ResponseWriter, r *http.Request) {
	var request SaveSubledgerEventMappingRequest
	if err := decodeJSON(r, &request); err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	if err := request.Validate(); err != nil {
		writeValidationError(w, r, h.logger, err)
		return
	}
	response, err := h.uc.SaveSubledgerMapping.Execute(r.Context(), request)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *SubledgerHandler) HandleListEvents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit := 50
	offset := 0
	if value := query.Get("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 1 && parsed <= 200 {
			limit = parsed
		}
	}
	if value := query.Get("offset"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	response, err := h.uc.ListSubledgerEvents.Execute(r.Context(), limit, offset)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}
