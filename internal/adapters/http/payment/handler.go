package paymenthttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	paymentusecase "cashflow_backend/internal/usecase/payment"

	"github.com/go-chi/chi/v5"
)

// Handler serves REST API endpoints for the Payment domain.
type Handler struct {
	useCase *paymentusecase.UseCase
	logger  *slog.Logger
}

// NewHandler constructs a new Payment HTTP Handler.
func NewHandler(useCase *paymentusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func parseID(param string) (int64, error) {
	return strconv.ParseInt(param, 10, 64)
}

// CreatePayment handles POST /api/v1/payments
func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	p, err := h.useCase.CreatePayment(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToPaymentResponse(p))
}

// GetPayment handles GET /api/v1/payments/{id}
func (h *Handler) GetPayment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid payment ID", err))
		return
	}

	p, err := h.useCase.GetPayment(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPaymentResponse(p))
}

// UpdatePayment handles PUT /api/v1/payments/{id}
func (h *Handler) UpdatePayment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid payment ID", err))
		return
	}

	var req UpdatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	p, err := h.useCase.UpdatePayment(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPaymentResponse(p))
}

// DeletePayment handles DELETE /api/v1/payments/{id}
func (h *Handler) DeletePayment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid payment ID", err))
		return
	}

	if err := h.useCase.DeletePayment(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// ListPayments handles GET /api/v1/payments
func (h *Handler) ListPayments(w http.ResponseWriter, r *http.Request) {
	page := pagination.Parse(r)
	q := r.URL.Query()

	f := filter.NewFilter()
	if partnerIDStr := q.Get("partner_id"); partnerIDStr != "" {
		if pid, err := strconv.ParseInt(partnerIDStr, 10, 64); err == nil {
			f.Add("partner_id", filter.OpEqual, pid)
		}
	}
	if journalIDStr := q.Get("journal_id"); journalIDStr != "" {
		if jid, err := strconv.ParseInt(journalIDStr, 10, 64); err == nil {
			f.Add("journal_id", filter.OpEqual, jid)
		}
	}
	if pType := q.Get("payment_type"); pType != "" {
		f.Add("payment_type", filter.OpEqual, pType)
	}
	if state := q.Get("state"); state != "" {
		f.Add("state", filter.OpEqual, state)
	}
	if method := q.Get("payment_method"); method != "" {
		f.Add("payment_method", filter.OpEqual, method)
	}

	res, err := h.useCase.ListPayments(r.Context(), f, page)
	if err != nil {
		response.Error(w, err)
		return
	}

	respItems := make([]PaymentResponse, len(res.Items))
	for i := range res.Items {
		respItems[i] = ToPaymentResponse(&res.Items[i])
	}

	response.Paginated(w, http.StatusOK, respItems, res)
}

// PostPayment handles POST /api/v1/payments/{id}/post
func (h *Handler) PostPayment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid payment ID", err))
		return
	}

	var req PostPaymentRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	p, err := h.useCase.PostPayment(r.Context(), id, req.AutoReconcile)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPaymentResponse(p))
}

// CancelPayment handles POST /api/v1/payments/{id}/cancel
func (h *Handler) CancelPayment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid payment ID", err))
		return
	}

	p, err := h.useCase.CancelPayment(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPaymentResponse(p))
}

// ReconcilePayment handles POST /api/v1/payments/{id}/reconcile
func (h *Handler) ReconcilePayment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid payment ID", err))
		return
	}

	var req ReconcilePaymentRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	p, err := h.useCase.ReconcilePayment(r.Context(), id, paymentusecase.ReconcileInput{
		InvoiceIDs: req.InvoiceIDs,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPaymentResponse(p))
}

// GetReceivableAging handles GET /api/v1/payments/receivable
func (h *Handler) GetReceivableAging(w http.ResponseWriter, r *http.Request) {
	asOfDate := time.Now().UTC()
	q := r.URL.Query()
	if dateStr := q.Get("as_of_date"); dateStr != "" {
		if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
			asOfDate = parsed
		}
	}

	var partnerID *int64
	if pidStr := q.Get("partner_id"); pidStr != "" {
		if pid, err := strconv.ParseInt(pidStr, 10, 64); err == nil {
			partnerID = &pid
		}
	}

	report, err := h.useCase.GetReceivableAging(r.Context(), asOfDate, partnerID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToAgingReportResponse(report))
}

// GetPayableAging handles GET /api/v1/payments/payable
func (h *Handler) GetPayableAging(w http.ResponseWriter, r *http.Request) {
	asOfDate := time.Now().UTC()
	q := r.URL.Query()
	if dateStr := q.Get("as_of_date"); dateStr != "" {
		if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
			asOfDate = parsed
		}
	}

	var partnerID *int64
	if pidStr := q.Get("partner_id"); pidStr != "" {
		if pid, err := strconv.ParseInt(pidStr, 10, 64); err == nil {
			partnerID = &pid
		}
	}

	report, err := h.useCase.GetPayableAging(r.Context(), asOfDate, partnerID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToAgingReportResponse(report))
}
