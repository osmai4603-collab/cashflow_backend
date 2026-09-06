package partnerhttp

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
	partnerusecase "cashflow_backend/internal/usecase/partner"

	"github.com/go-chi/chi/v5"
)

// Handler serves HTTP requests for the Partner domain.
type Handler struct {
	useCase partnerusecase.UseCase
	logger  *slog.Logger
}

// NewHandler constructs a new Handler.
func NewHandler(useCase partnerusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

// Create handles POST /api/v1/partners
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreatePartnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	created, err := h.useCase.CreatePartner(r.Context(), req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ToPartnerResponse(created))
}

// GetByID handles GET /api/v1/partners/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid partner ID in path", err))
		return
	}

	partner, err := h.useCase.GetPartner(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPartnerResponse(partner))
}

// Update handles PUT /api/v1/partners/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid partner ID in path", err))
		return
	}

	var req UpdatePartnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, platformerrors.BadRequest("invalid JSON request body", err))
		return
	}

	updated, err := h.useCase.UpdatePartner(r.Context(), id, req.ToInput())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, ToPartnerResponse(updated))
}

// Delete handles DELETE /api/v1/partners/{id} (soft-delete)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, platformerrors.BadRequest("invalid partner ID in path", err))
		return
	}

	if err := h.useCase.DeletePartner(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// List handles GET /api/v1/partners with filtering and pagination
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pageReq := pagination.Parse(r)
	f := parseFilterFromQuery(r)

	result, err := h.useCase.ListPartners(r.Context(), f, pageReq)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Paginated(w, http.StatusOK, ToPartnerResponseList(result.Items), result)
}

// ListCustomers handles GET /api/v1/partners/customers
func (h *Handler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	pageReq := pagination.Parse(r)

	result, err := h.useCase.ListCustomers(r.Context(), pageReq)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Paginated(w, http.StatusOK, ToPartnerResponseList(result.Items), result)
}

// ListSuppliers handles GET /api/v1/partners/suppliers
func (h *Handler) ListSuppliers(w http.ResponseWriter, r *http.Request) {
	pageReq := pagination.Parse(r)

	result, err := h.useCase.ListSuppliers(r.Context(), pageReq)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Paginated(w, http.StatusOK, ToPartnerResponseList(result.Items), result)
}

func parseID(param string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(param), 10, 64)
}

func parseFilterFromQuery(r *http.Request) *filter.Filter {
	q := r.URL.Query()
	f := filter.NewFilter()

	if name := q.Get("name"); name != "" {
		f.Add("name", filter.OpILike, name)
	}

	if pType := q.Get("type"); pType != "" {
		f.Add("type", filter.OpEqual, pType)
	}

	if city := q.Get("city"); city != "" {
		f.Add("city", filter.OpEqual, city)
	}

	if country := q.Get("country"); country != "" {
		f.Add("country", filter.OpEqual, country)
	}

	if cust := q.Get("is_customer"); cust != "" {
		if b, err := strconv.ParseBool(cust); err == nil {
			f.Add("is_customer", filter.OpEqual, b)
		}
	}

	if supp := q.Get("is_supplier"); supp != "" {
		if b, err := strconv.ParseBool(supp); err == nil {
			f.Add("is_supplier", filter.OpEqual, b)
		}
	}

	if len(f.Criteria) == 0 {
		return nil
	}
	return f
}
