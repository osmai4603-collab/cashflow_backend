package currencyhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	currencyusecase "cashflow_backend/internal/usecase/currency"

	"github.com/go-chi/chi/v5"
)

// Handler serves HTTP requests for the Currency domain.
type Handler struct {
	useCase currencyusecase.UseCase
	logger  *slog.Logger
}

// NewHandler constructs a new Handler.
func NewHandler(useCase currencyusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

// Create handles POST /api/v1/currencies
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCurrencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	created, err := h.useCase.CreateCurrency(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToCurrencyResponse(created))
}

// GetByID handles GET /api/v1/currencies/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseCurrencyID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid currency ID in path", err))
		return
	}

	c, err := h.useCase.GetCurrency(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToCurrencyResponse(c))
}

// Update handles PUT /api/v1/currencies/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseCurrencyID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid currency ID in path", err))
		return
	}

	var req UpdateCurrencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdateCurrency(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToCurrencyResponse(updated))
}

// Delete handles DELETE /api/v1/currencies/{id} (soft-delete)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseCurrencyID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid currency ID in path", err))
		return
	}

	if err := h.useCase.DeleteCurrency(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// List handles GET /api/v1/currencies with filtering and pagination
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pageReq := pagination.Parse(r)
	f := parseCurrencyFilterFromQuery(r)

	result, err := h.useCase.ListCurrencies(r.Context(), f, pageReq)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Paginated(w, http.StatusOK, ToCurrencyResponseList(result.Items), result)
}

// CreateRate handles POST /api/v1/currencies/{id}/rates
func (h *Handler) CreateRate(w http.ResponseWriter, r *http.Request) {
	id, err := parseCurrencyID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid currency ID in path", err))
		return
	}

	var req CreateRateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}
	req.CurrencyID = id

	created, err := h.useCase.CreateRate(r.Context(), req.ToRateInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToCurrencyRateResponse(created))
}

// ListRates handles GET /api/v1/currencies/{id}/rates
func (h *Handler) ListRates(w http.ResponseWriter, r *http.Request) {
	id, err := parseCurrencyID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid currency ID in path", err))
		return
	}

	pageReq := pagination.Parse(r)
	result, err := h.useCase.ListRates(r.Context(), id, pageReq)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Paginated(w, http.StatusOK, ToCurrencyRateResponseList(result.Items), result)
}

// Convert handles POST /api/v1/currencies/convert
func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	var req ConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	result, err := h.useCase.Convert(r.Context(), req.ToConvertInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func parseCurrencyID(param string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(param), 10, 64)
}

func parseCurrencyFilterFromQuery(r *http.Request) *filter.Filter {
	q := r.URL.Query()
	f := filter.NewFilter()

	if name := q.Get("name"); name != "" {
		f.Add("name", filter.OpILike, name)
	}

	if fullName := q.Get("full_name"); fullName != "" {
		f.Add("full_name", filter.OpILike, fullName)
	}

	if len(f.Criteria) == 0 {
		return nil
	}
	return f
}
