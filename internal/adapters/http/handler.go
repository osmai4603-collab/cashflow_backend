package httpadapter

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"cashflow_backend/internal/domain"
	"cashflow_backend/internal/usecase"
)

type TransactionHandler struct {
	uc     *usecase.TransactionUseCase
	logger *slog.Logger
}

func NewTransactionHandler(uc *usecase.TransactionUseCase, logger *slog.Logger) *TransactionHandler {
	return &TransactionHandler{
		uc:     uc,
		logger: logger,
	}
}

func (h *TransactionHandler) Root(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"service": "cashflow_backend",
		"version": "1.0.0",
		"status":  "healthy",
	})
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	tx, err := h.uc.CreateTransaction(r.Context(), usecase.CreateTransactionInput{
		Amount:      req.Amount,
		Type:        req.Type,
		Description: req.Description,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAmount) ||
			errors.Is(err, domain.ErrEmptyDescription) ||
			errors.Is(err, domain.ErrInvalidType) {
			respondError(w, http.StatusBadRequest, "validation failed", err.Error())
			return
		}
		h.logger.Error("failed to create transaction", "error", err)
		respondError(w, http.StatusInternalServerError, "internal server error", "")
		return
	}

	respondJSON(w, http.StatusCreated, toTransactionResponse(tx))
}

func (h *TransactionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "id parameter is required", "")
		return
	}

	tx, err := h.uc.GetTransaction(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTransactionNotFound) {
			respondError(w, http.StatusNotFound, "transaction not found", "")
			return
		}
		h.logger.Error("failed to get transaction", "id", id, "error", err)
		respondError(w, http.StatusInternalServerError, "internal server error", "")
		return
	}

	respondJSON(w, http.StatusOK, toTransactionResponse(tx))
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	txs, err := h.uc.ListTransactions(r.Context())
	if err != nil {
		h.logger.Error("failed to list transactions", "error", err)
		respondError(w, http.StatusInternalServerError, "internal server error", "")
		return
	}

	res := make([]TransactionResponse, len(txs))
	for i, tx := range txs {
		res[i] = toTransactionResponse(tx)
	}

	respondJSON(w, http.StatusOK, res)
}

func (h *TransactionHandler) Summary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.uc.GetSummary(r.Context())
	if err != nil {
		h.logger.Error("failed to get summary", "error", err)
		respondError(w, http.StatusInternalServerError, "internal server error", "")
		return
	}

	respondJSON(w, http.StatusOK, toSummaryResponse(summary))
}

func respondJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, statusCode int, message string, details string) {
	respondJSON(w, statusCode, ErrorResponse{
		Error:   message,
		Details: details,
	})
}
