package accounting

import (
	"log/slog"
	"net/http"
)

type ConfigurationHandler struct {
	uc     *UseCases
	logger *slog.Logger
}

func NewConfigurationHandler(uc *UseCases, logger *slog.Logger) *ConfigurationHandler {
	return &ConfigurationHandler{uc: uc, logger: handlerLogger(logger, "configuration")}
}

func (h *ConfigurationHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	response, err := h.uc.GetAccountingConfiguration.Execute(r.Context())
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *ConfigurationHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	var request UpdateAccountingConfigurationRequest
	if err := decodeJSON(r, &request); err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	if err := request.Validate(); err != nil {
		writeValidationError(w, r, h.logger, err)
		return
	}
	response, err := h.uc.UpdateAccountingConfiguration.Execute(r.Context(), request)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

