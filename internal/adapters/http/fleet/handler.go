package fleethttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/fleet"
	"cashflow_backend/internal/platform/auth"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/i18n"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	fleetusecase "cashflow_backend/internal/usecase/fleet"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *fleetusecase.Service
	logger  *slog.Logger
}

func NewHandler(useCase *fleetusecase.Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{useCase: useCase, logger: logger}
}

func requestCompanyID(r *http.Request) (int64, error) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil || claims.CompanyID <= 0 {
		return 0, platformerrors.Unauthorized("authenticated company is required")
	}
	return claims.CompanyID, nil
}

func pathID(r *http.Request, name string) (int64, error) {
	value, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil || value <= 0 {
		return 0, platformerrors.BadRequest("invalid ID in path")
	}
	return value, nil
}

func decodeBody(r *http.Request, target any) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return platformerrors.BadRequest("invalid JSON request body", err)
	}
	return nil
}

// --- Brands ---

func (h *Handler) CreateBrand(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	var request CreateBrandRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain()
	if err := h.useCase.CreateBrand(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListBrands(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListBrands(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) GetBrand(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetBrand(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateBrand(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetBrand(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateBrandRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	if request.Name != nil {
		value.Name = i18n.NewTranslation(*request.Name)
	}
	if err := h.useCase.UpdateBrand(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteBrand(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteBrand(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Model categories ---

func (h *Handler) CreateModelCategory(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	var request CreateModelCategoryRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain()
	if err := h.useCase.CreateModelCategory(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListModelCategories(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListModelCategories(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) GetModelCategory(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetModelCategory(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateModelCategory(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetModelCategory(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateModelCategoryRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	if request.Name != nil {
		value.Name = i18n.NewTranslation(*request.Name)
	}
	if err := h.useCase.UpdateModelCategory(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteModelCategory(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteModelCategory(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Models ---

func (h *Handler) CreateModel(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	var request CreateModelRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain()
	if err := h.useCase.CreateModel(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListModels(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	var brandID *int64
	if raw := r.URL.Query().Get("brand_id"); raw != "" {
		value, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || value <= 0 {
			response.Error(w, platformerrors.BadRequest("invalid brand_id"))
			return
		}
		brandID = &value
	}
	values, err := h.useCase.ListModels(r.Context(), brandID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) GetModel(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetModel(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateModel(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetModel(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateModelRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	if request.Name != nil {
		value.Name = i18n.NewTranslation(*request.Name)
	}
	if request.BrandID != nil {
		value.BrandID = *request.BrandID
	}
	if request.CategoryID != nil {
		value.CategoryID = request.CategoryID
	}
	if err := h.useCase.UpdateModel(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteModel(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteModel(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Tags ---

func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	var request CreateTagRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain()
	if err := h.useCase.CreateTag(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListTags(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) GetTag(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetTag(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateTag(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetTag(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateTagRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	if request.Name != nil {
		value.Name = i18n.NewTranslation(*request.Name)
	}
	if request.Color != nil {
		value.Color = *request.Color
	}
	if err := h.useCase.UpdateTag(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteTag(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- States ---

func (h *Handler) CreateState(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	var request CreateStateRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain()
	if err := h.useCase.CreateState(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListStates(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListStates(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) GetState(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetState(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateState(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetState(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateStateRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	if request.Name != nil {
		value.Name = i18n.NewTranslation(*request.Name)
	}
	if request.Sequence != nil {
		value.Sequence = *request.Sequence
	}
	if request.Fold != nil {
		value.Fold = *request.Fold
	}
	if err := h.useCase.UpdateState(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteState(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteState(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Service types ---

func (h *Handler) CreateServiceType(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	var request CreateServiceTypeRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain()
	if err := h.useCase.CreateServiceType(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListServiceTypes(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListServiceTypes(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) GetServiceType(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetServiceType(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateServiceType(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetServiceType(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateServiceTypeRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	if request.Name != nil {
		value.Name = i18n.NewTranslation(*request.Name)
	}
	if request.Category != nil {
		value.Category = fleet.ServiceTypeCategory(*request.Category)
	}
	if err := h.useCase.UpdateServiceType(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteServiceType(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteServiceType(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Vehicles ---

func (h *Handler) CreateVehicle(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateVehicleRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain(companyID)
	if err := h.useCase.CreateVehicle(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListVehicles(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	page := pagination.Parse(r)
	f := filter.NewFilter()
	if raw := r.URL.Query().Get("license_plate"); raw != "" {
		f.Criteria = append(f.Criteria, filter.Criterion{Field: "license_plate", Operator: filter.OpILike, Value: raw})
	}
	result, err := h.useCase.ListVehicles(r.Context(), companyID, f, page)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Paginated(w, http.StatusOK, result.Items, result)
}

func (h *Handler) GetVehicle(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetVehicle(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateVehicle(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetVehicle(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateVehicleRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	request.Apply(value)
	if err := h.useCase.UpdateVehicle(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteVehicle(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteVehicle(r.Context(), companyID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Assignation logs ---

func (h *Handler) CreateAssignation(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	vehicleID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateAssignationRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	if request.DriverID <= 0 {
		response.Error(w, platformerrors.Validation("driver_id is required", map[string]string{"driver_id": "must be positive"}))
		return
	}
	value := request.ToDomain(vehicleID)
	if err := h.useCase.CreateAssignationLog(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListAssignations(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	vehicleID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListAssignationLogs(r.Context(), companyID, vehicleID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) UpdateAssignation(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "log_id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetAssignationLog(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateAssignationRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	if request.DateFrom != nil {
		value.DateStart = request.DateFrom
	}
	if request.DateTo != nil {
		value.DateEnd = request.DateTo
	}
	if err := h.useCase.UpdateAssignationLog(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteAssignation(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "log_id")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteAssignationLog(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Odometers ---

func (h *Handler) CreateOdometer(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	vehicleID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateOdometerRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain(vehicleID)
	if err := h.useCase.CreateOdometer(r.Context(), companyID, value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListOdometers(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	vehicleID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListOdometers(r.Context(), companyID, vehicleID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) UpdateOdometer(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "odometer_id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetOdometer(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateOdometerRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	if request.Date != nil {
		value.Date = *request.Date
	}
	if request.Value != nil {
		value.Value = *request.Value
	}
	if request.Unit != nil {
		value.Unit = *request.Unit
	}
	if err := h.useCase.UpdateOdometer(r.Context(), companyID, value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteOdometer(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "odometer_id")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteOdometer(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Service logs ---

func (h *Handler) CreateService(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	vehicleID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateServiceRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain(vehicleID, companyID)
	if err := h.useCase.CreateLogService(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListServices(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	vehicleID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	page := pagination.Parse(r)
	f := filter.NewFilter(
		filter.Criterion{Field: "vehicle_id", Operator: filter.OpEqual, Value: vehicleID},
	)
	result, err := h.useCase.ListLogServices(r.Context(), companyID, f, page)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Paginated(w, http.StatusOK, result.Items, result)
}

func (h *Handler) UpdateService(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "service_id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetLogService(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateServiceRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	request.Apply(value)
	if err := h.useCase.UpdateLogService(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteService(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "service_id")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteLogService(r.Context(), companyID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Contracts ---

func (h *Handler) CreateContract(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	vehicleID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateContractRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain(vehicleID, companyID)
	if err := h.useCase.CreateLogContract(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListContracts(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	vehicleID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	page := pagination.Parse(r)
	f := filter.NewFilter(
		filter.Criterion{Field: "vehicle_id", Operator: filter.OpEqual, Value: vehicleID},
	)
	result, err := h.useCase.ListLogContracts(r.Context(), companyID, f, page)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Paginated(w, http.StatusOK, result.Items, result)
}

func (h *Handler) UpdateContract(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "contract_id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetLogContract(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateContractRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	request.Apply(value)
	if err := h.useCase.UpdateLogContract(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteContract(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "contract_id")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteLogContract(r.Context(), companyID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Reports ---

func (h *Handler) CostByVehicle(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.CostByVehicle(r.Context(), companyID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}