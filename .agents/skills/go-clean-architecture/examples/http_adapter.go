package cleanarch

import (
	"encoding/json"
	"net/http"
	"time"
)

// =============================================================================
// LAYER 3: ADAPTERS LAYER (Primary / Driving Adapter: HTTP Handler)
// =============================================================================
// Rules:
// - Translates HTTP requests (JSON DTOs) to Use Case Commands.
// - Validates HTTP input boundaries.
// - Never executes raw SQL or business validation logic.
// =============================================================================

type PayInvoiceRequestDTO struct {
	InvoiceID string `json:"invoice_id"`
}

type PayInvoiceResponseDTO struct {
	Status string `json:"status"`
	Message string `json:"message"`
}

type InvoiceHTTPHandler struct {
	payUseCase *PayInvoiceUseCase
}

func NewInvoiceHTTPHandler(payUC *PayInvoiceUseCase) *InvoiceHTTPHandler {
	return &InvoiceHTTPHandler{
		payUseCase: payUC,
	}
}

func (h *InvoiceHTTPHandler) HandlePay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqDTO PayInvoiceRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&reqDTO); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	if reqDTO.InvoiceID == "" {
		http.Error(w, "invoice_id is required", http.StatusBadRequest)
		return
	}

	cmd := PayInvoiceCommand{
		InvoiceID: reqDTO.InvoiceID,
		PaidAt:    time.Now(),
	}

	if err := h.payUseCase.Execute(r.Context(), cmd); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(PayInvoiceResponseDTO{
		Status:  "success",
		Message: "invoice marked as paid",
	})
}
