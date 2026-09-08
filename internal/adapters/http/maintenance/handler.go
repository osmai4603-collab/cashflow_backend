package maintenancehttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"cashflow_backend/internal/platform/auth"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/i18n"
	"cashflow_backend/internal/platform/pagination"
	"cashflow_backend/internal/platform/response"
	maintenanceusecase "cashflow_backend/internal/usecase/maintenance"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	useCase *maintenanceusecase.Service
	logger  *slog.Logger
}

func NewHandler(useCase *maintenanceusecase.Service, logger *slog.Logger) *Handler {
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

// --- Equipment categories ---

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateCategoryRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain(companyID)
	if err := h.useCase.CreateCategory(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListCategories(r.Context(), &companyID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetCategory(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetCategory(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateCategoryRequest
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
	if err := h.useCase.UpdateCategory(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
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
	if err := h.useCase.DeleteCategory(r.Context(), companyID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Stages ---

func (h *Handler) CreateStage(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	var request CreateStageRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain()
	if err := h.useCase.CreateStage(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListStages(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListStages(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) GetStage(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetStage(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetStage(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateStageRequest
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
	if request.Done != nil {
		value.Done = *request.Done
	}
	if err := h.useCase.UpdateStage(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteStage(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteStage(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Teams ---

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateTeamRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value, memberIDs := request.ToDomain(companyID)
	if err := h.useCase.CreateTeam(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	if len(memberIDs) > 0 {
		if err := h.useCase.SetTeamMembers(r.Context(), value.ID, memberIDs); err != nil {
			response.Error(w, err)
			return
		}
	}
	response.Created(w, value)
}

func (h *Handler) ListTeams(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListTeams(r.Context(), &companyID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetTeam(r.Context(), &companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateTeam(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetTeam(r.Context(), &companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateTeamRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	if request.Name != nil {
		value.Name = i18n.NewTranslation(*request.Name)
	}
	if err := h.useCase.UpdateTeam(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	if request.MemberIDs != nil {
		if err := h.useCase.SetTeamMembers(r.Context(), value.ID, request.MemberIDs); err != nil {
			response.Error(w, err)
			return
		}
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteTeam(w http.ResponseWriter, r *http.Request) {
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
	if err := h.useCase.DeleteTeam(r.Context(), &companyID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Equipment ---

func (h *Handler) CreateEquipment(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateEquipmentRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain(companyID)
	if err := h.useCase.CreateEquipment(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListEquipments(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	page := pagination.Parse(r)
	f := filter.NewFilter()
	if raw := r.URL.Query().Get("name"); raw != "" {
		f.Criteria = append(f.Criteria, filter.Criterion{Field: "name", Operator: filter.OpILike, Value: raw})
	}
	if raw := r.URL.Query().Get("category_id"); raw != "" {
		value, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || value <= 0 {
			response.Error(w, platformerrors.BadRequest("invalid category_id"))
			return
		}
		f.Criteria = append(f.Criteria, filter.Criterion{Field: "category_id", Operator: filter.OpEqual, Value: value})
	}
	result, err := h.useCase.ListEquipments(r.Context(), companyID, f, page)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Paginated(w, http.StatusOK, result.Items, result)
}

func (h *Handler) GetEquipment(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetEquipment(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateEquipment(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetEquipment(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateEquipmentRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	request.Apply(value)
	if err := h.useCase.UpdateEquipment(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteEquipment(w http.ResponseWriter, r *http.Request) {
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
	if err := h.useCase.DeleteEquipment(r.Context(), companyID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// --- Requests ---

func (h *Handler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateRequestRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := request.ToDomain(companyID)
	if err := h.useCase.CreateRequest(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListRequests(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	page := pagination.Parse(r)
	f := filter.NewFilter()
	if raw := r.URL.Query().Get("maintenance_type"); raw != "" {
		f.Criteria = append(f.Criteria, filter.Criterion{Field: "maintenance_type", Operator: filter.OpEqual, Value: raw})
	}
	if raw := r.URL.Query().Get("equipment_id"); raw != "" {
		value, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || value <= 0 {
			response.Error(w, platformerrors.BadRequest("invalid equipment_id"))
			return
		}
		f.Criteria = append(f.Criteria, filter.Criterion{Field: "equipment_id", Operator: filter.OpEqual, Value: value})
	}
	if raw := r.URL.Query().Get("recurring"); raw != "" {
		f.Criteria = append(f.Criteria, filter.Criterion{Field: "recurring_maintenance", Operator: filter.OpEqual, Value: raw == "true"})
	}
	result, err := h.useCase.ListRequests(r.Context(), companyID, f, page)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Paginated(w, http.StatusOK, result.Items, result)
}

func (h *Handler) GetRequest(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetRequest(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateRequest(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetRequest(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateRequestRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	request.Apply(value)
	if err := h.useCase.UpdateRequest(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteRequest(w http.ResponseWriter, r *http.Request) {
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
	if err := h.useCase.DeleteRequest(r.Context(), companyID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) CloseRequest(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.CloseRequest(r.Context(), companyID, id, timeNow())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) ArchiveRequest(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.ArchiveRequest(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.Dashboard(r.Context(), companyID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}

func timeNow() time.Time {
	return time.Now().UTC()
}