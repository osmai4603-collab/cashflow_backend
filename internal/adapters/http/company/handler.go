package companyhttp

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
	companyusecase "cashflow_backend/internal/usecase/company"

	"github.com/go-chi/chi/v5"
)

// Handler serves HTTP requests for the Company domain.
type Handler struct {
	useCase companyusecase.UseCase
	logger  *slog.Logger
}

// NewHandler constructs a new Handler.
func NewHandler(useCase companyusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

// Create handles POST /api/v1/companies
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	created, err := h.useCase.CreateCompany(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Created(w, ToCompanyResponse(created))
}

// GetByID handles GET /api/v1/companies/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseCompanyID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid company ID in path", err))
		return
	}

	c, err := h.useCase.GetCompany(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToCompanyResponse(c))
}

// Update handles PUT /api/v1/companies/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseCompanyID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid company ID in path", err))
		return
	}

	var req UpdateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdateCompany(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToCompanyResponse(updated))
}

// Delete handles DELETE /api/v1/companies/{id} (soft-delete)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseCompanyID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, r, platformerrors.BadRequest("invalid company ID in path", err))
		return
	}

	if err := h.useCase.DeleteCompany(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}

	response.NoContent(w)
}

// List handles GET /api/v1/companies with filtering and pagination
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pageReq := pagination.Parse(r)
	f := parseCompanyFilterFromQuery(r)

	result, err := h.useCase.ListCompanies(r.Context(), f, pageReq)
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.Paginated(w, http.StatusOK, ToCompanyResponseList(result.Items), result)
}

// GetDefault handles GET /api/v1/companies/default
func (h *Handler) GetDefault(w http.ResponseWriter, r *http.Request) {
	c, err := h.useCase.GetDefaultCompany(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, ToCompanyResponse(c))
}

func parseCompanyID(param string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(param), 10, 64)
}

func parseCompanyFilterFromQuery(r *http.Request) *filter.Filter {
	q := r.URL.Query()
	f := filter.NewFilter()

	if name := q.Get("name"); name != "" {
		f.Add("name", filter.OpILike, name)
	}

	if currencyID := q.Get("currency_id"); currencyID != "" {
		if id, err := strconv.ParseInt(currencyID, 10, 64); err == nil {
			f.Add("currency_id", filter.OpEqual, id)
		}
	}

	if country := q.Get("country"); country != "" {
		f.Add("country", filter.OpEqual, country)
	}

	if len(f.Criteria) == 0 {
		return nil
	}
	return f
}
