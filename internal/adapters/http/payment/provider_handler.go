package paymenthttp

import (
	"encoding/json"
	"net/http"

	"cashflow_backend/internal/domain/payment"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
	"github.com/go-chi/chi/v5"
)

type initiateTransactionRequest struct {
	ProviderCode string  `json:"provider_code"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	PartnerID    int64   `json:"partner_id"`
	CompanyID    int64   `json:"company_id"`
}

func (h *Handler) InitiateTransaction(w http.ResponseWriter, r *http.Request) {
	var request initiateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	if request.ProviderCode == "" || request.Amount <= 0 || request.PartnerID <= 0 || request.CompanyID <= 0 {
		response.Error(w, platformerrors.Validation("provider, amount, partner and company are required", nil))
		return
	}
	transaction, err := h.useCase.CreateTransaction(r.Context(), request.ProviderCode, request.Amount, request.Currency, request.PartnerID, request.CompanyID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, transaction)
}

type transactionWebhookRequest struct {
	Reference         string                   `json:"transaction_ref"`
	NewState          payment.TransactionState `json:"new_state"`
	ProviderReference string                   `json:"provider_ref"`
}

func (h *Handler) ProcessTransactionWebhook(w http.ResponseWriter, r *http.Request) {
	var request transactionWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	if request.Reference == "" || request.NewState == "" {
		response.Error(w, platformerrors.Validation("transaction_ref and new_state are required", nil))
		return
	}
	if err := h.useCase.ProcessTransactionWebhook(r.Context(), request.Reference, request.NewState, request.ProviderReference); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "processed"})
}

func (h *Handler) CaptureTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid transaction ID", err))
		return
	}
	transaction, err := h.useCase.CaptureTransaction(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, transaction)
}

func (h *Handler) VoidTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid transaction ID", err))
		return
	}
	transaction, err := h.useCase.VoidTransaction(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, transaction)
}
