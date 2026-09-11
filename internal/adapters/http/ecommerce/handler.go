package ecommercehttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/ecommerce"
	"cashflow_backend/internal/platform/response"
	ecommerceusecase "cashflow_backend/internal/usecase/ecommerce"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *ecommerceusecase.UseCase
	logger  *slog.Logger
}

func NewHandler(useCase *ecommerceusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.useCase.GetCategories(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cats)
}

func (h *Handler) GetProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.useCase.GetProducts(r.Context(), nil) // simplified
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, products)
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	product, err := h.useCase.GetProductByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, product)
}

func (h *Handler) AddToCart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WebsiteID int64   `json:"website_id"`
		ProductID int64   `json:"product_id"`
		Quantity  float64 `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	sessionUUID := r.Header.Get("X-Session-ID")
	cart, err := h.useCase.AddToCart(r.Context(), req.WebsiteID, sessionUUID, req.ProductID, req.Quantity)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, cart)
}

func (h *Handler) UpdateCartLine(w http.ResponseWriter, r *http.Request) {
	lineID, _ := strconv.ParseInt(chi.URLParam(r, "line_id"), 10, 64)
	var req struct {
		CartID   int64   `json:"cart_id"`
		Quantity float64 `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	cart, err := h.useCase.UpdateCartLine(r.Context(), req.CartID, lineID, req.Quantity)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cart)
}

func (h *Handler) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	lineID, _ := strconv.ParseInt(chi.URLParam(r, "line_id"), 10, 64)
	cartID, _ := strconv.ParseInt(r.URL.Query().Get("cart_id"), 10, 64)
	cart, err := h.useCase.RemoveFromCart(r.Context(), cartID, lineID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cart)
}

func (h *Handler) GetCart(w http.ResponseWriter, r *http.Request) {
	sessionUUID := r.Header.Get("X-Session-ID")
	cart, err := h.useCase.GetCart(r.Context(), sessionUUID, 0) // partnerID=0 for guest
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cart)
}

func (h *Handler) ApplyCoupon(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CartID int64  `json:"cart_id"`
		Code   string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.ApplyCoupon(r.Context(), req.CartID, req.Code); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) CheckoutAddresses(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CartID            int64 `json:"cart_id"`
		ShippingAddressID int64 `json:"shipping_address_id"`
		InvoiceAddressID  int64 `json:"invoice_address_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	cart, err := h.useCase.SetCheckoutAddresses(r.Context(), req.CartID, req.ShippingAddressID, req.InvoiceAddressID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cart)
}

func (h *Handler) CheckoutShipping(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CartID           int64 `json:"cart_id"`
		DeliveryMethodID int64 `json:"delivery_method_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	cart, err := h.useCase.SetShippingMethod(r.Context(), req.CartID, req.DeliveryMethodID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cart)
}

func (h *Handler) Pay(w http.ResponseWriter, r *http.Request) {
	var req ecommerce.CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	result, err := h.useCase.ProcessCheckout(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
