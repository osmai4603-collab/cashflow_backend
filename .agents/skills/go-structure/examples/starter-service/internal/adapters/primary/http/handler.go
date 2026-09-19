package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/example/starter-service/internal/core/domain"
	"github.com/example/starter-service/internal/usecases"
)

// WalletHandler المحول الأولي لمعالجة طلبات HTTP المتعلقة بالمحافظ
type WalletHandler struct {
	transferUC *usecases.TransferFundsUseCase
}

// NewWalletHandler منشئ معالج HTTP
func NewWalletHandler(transferUC *usecases.TransferFundsUseCase) *WalletHandler {
	return &WalletHandler{transferUC: transferUC}
}

// TransferRequestDTO هيكل دخل الطلب الخارجي
type TransferRequestDTO struct {
	FromWalletID string `json:"from_wallet_id"`
	ToWalletID   string `json:"to_wallet_id"`
	Amount       int64  `json:"amount"`
}

// HandleTransfer معالجة استدعاء POST /api/v1/wallets/transfer
func (h *WalletHandler) HandleTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var dto TransferRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}

	req := usecases.TransferRequest{
		FromWalletID: dto.FromWalletID,
		ToWalletID:   dto.ToWalletID,
		Amount:       dto.Amount,
	}

	result, err := h.transferUC.Execute(r.Context(), req)
	if err != nil {
		// مطابقة الأخطاء الدلالية لرموز حالة HTTP المناسبة
		switch {
		case errors.Is(err, domain.ErrWalletNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		case errors.Is(err, domain.ErrInsufficientBalance),
			errors.Is(err, domain.ErrInvalidAmount),
			errors.Is(err, domain.ErrSameAccountTransfer):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal processing error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "transfer completed successfully",
		"data":    result,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
