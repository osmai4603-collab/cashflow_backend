package accounting

import (
	"log/slog"
	"net/http"
)

type AccountHandler struct {
	uc     *UseCases
	logger *slog.Logger
}

func NewAccountHandler(uc *UseCases, logger *slog.Logger) *AccountHandler {
	return &AccountHandler{uc: uc, logger: handlerLogger(logger, "account")}
}

func (h *AccountHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var request CreateAccountRequest
	if err := decodeJSON(r, &request); err != nil {
		writeBadRequest(w, r, h.logger, err)
		return
	}
	if err := request.Validate(); err != nil {
		writeValidationError(w, r, h.logger, err)
		return
	}

	response, err := h.uc.CreateAccount.Execute(r.Context(), request)
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (h *AccountHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	response, err := h.uc.ListAccounts.Execute(r.Context())
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AccountHandler) HandleTree(w http.ResponseWriter, r *http.Request) {
	response, err := h.uc.ListAccountsTree.Execute(r.Context())
	if err != nil {
		writeAccountingError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}
