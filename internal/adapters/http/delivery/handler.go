package deliveryhttp

import (
	"encoding/json"
	"net/http"
	"strconv"

	"cashflow_backend/internal/usecase/delivery"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	uc *deliveryusecase.UseCase
}

func NewHandler(uc *deliveryusecase.UseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) ListCarriers(w http.ResponseWriter, r *http.Request) {
	carriers, err := h.uc.ListCarriers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(carriers)
}

func (h *Handler) CalculateRate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID   int64 `json:"order_id"`
		CarrierID int64 `json:"carrier_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := h.uc.CalculateRate(r.Context(), req.OrderID, req.CarrierID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) CalculateOrderRate(w http.ResponseWriter, r *http.Request) {
	orderID, _ := strconv.ParseInt(chi.URLParam(r, "orderID"), 10, 64)

	var req struct {
		CarrierID int64 `json:"carrier_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := h.uc.CalculateRate(r.Context(), orderID, req.CarrierID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) SetOrderDelivery(w http.ResponseWriter, r *http.Request) {
	orderID, _ := strconv.ParseInt(chi.URLParam(r, "orderID"), 10, 64)

	var req struct {
		CarrierID int64 `json:"carrier_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.uc.AddShippingToOrder(r.Context(), orderID, req.CarrierID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
