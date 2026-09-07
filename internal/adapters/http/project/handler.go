package projecthttp

import (
	"encoding/json"
	"net/http"
	"strconv"

	projectdomain "cashflow_backend/internal/domain/project"
	"cashflow_backend/internal/platform/auth"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/response"
	projectusecase "cashflow_backend/internal/usecase/project"

	"github.com/go-chi/chi/v5"
)

type Handler struct{ useCase *projectusecase.Service }

func NewHandler(useCase *projectusecase.Service) *Handler { return &Handler{useCase: useCase} }

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

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateProjectRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.CreateProject(r.Context(), request.ToInput(companyID))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, ProjectResponse{Project: value})
}

func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListProjects(r.Context(), companyID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetProject(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ProjectResponse{Project: value})
}

func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetProject(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateProjectRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	input := request.ToInput(companyID)
	value.Name, value.Description, value.PartnerID, value.ManagerID = input.Name, input.Description, input.PartnerID, input.ManagerID
	value.StageID, value.DateStart, value.DateEnd = input.StageID, input.DateStart, input.DateEnd
	value.AllowMilestones, value.AllowSubtasks, value.AllowDependencies = input.AllowMilestones, input.AllowSubtasks, input.AllowDependencies
	value.AnalyticAccountID = input.AnalyticAccountID
	if err := h.useCase.UpdateProject(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ProjectResponse{Project: value})
}

func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
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
	if err := h.useCase.DeleteProject(r.Context(), companyID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateTaskRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.CreateTask(r.Context(), request.ToInput(companyID))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, TaskResponse{Task: value})
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetTask(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, TaskResponse{Task: value})
}

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	projectID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListTasks(r.Context(), companyID, projectID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetTask(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateTaskRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	input := request.ToInput(companyID)
	value.Name, value.ProjectID, value.StageID, value.AssigneeIDs, value.ParentID = input.Name, input.ProjectID, input.StageID, input.AssigneeIDs, input.ParentID
	value.Priority, value.DateDeadline, value.Description, value.MilestoneID, value.TagIDs = input.Priority, input.DateDeadline, input.Description, input.MilestoneID, input.TagIDs
	value.Sequence, value.AllocatedHours = input.Sequence, input.AllocatedHours
	if err := h.useCase.UpdateTask(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, TaskResponse{Task: value})
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
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
	if err := h.useCase.DeleteTask(r.Context(), companyID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) ReachMilestone(w http.ResponseWriter, r *http.Request) {
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
	if err := h.useCase.ReachMilestone(r.Context(), companyID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) CreateProjectStage(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateStageRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	if request.CompanyID == nil {
		request.CompanyID = &companyID
	}
	if *request.CompanyID != companyID {
		response.Error(w, platformerrors.Forbidden("stage belongs to another company"))
		return
	}
	value := &projectdomain.ProjectStage{Name: request.Name, Sequence: request.Sequence, Fold: request.Fold, Color: request.Color, CompanyID: request.CompanyID}
	if err := h.useCase.CreateProjectStage(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListProjectStages(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListProjectStages(r.Context(), &companyID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) UpdateProjectStage(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetProjectStage(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateStageRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value.Name, value.Sequence, value.Fold, value.Color = request.Name, request.Sequence, request.Fold, request.Color
	if err := h.useCase.UpdateProjectStage(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}
func (h *Handler) DeleteProjectStage(w http.ResponseWriter, r *http.Request) {
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
	if err := h.useCase.DeleteProjectStage(r.Context(), companyID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) CreateTaskStage(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateTaskStageRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := &projectdomain.TaskStage{Name: request.Name, Sequence: request.Sequence, Fold: request.Fold, Color: request.Color, CompanyID: &companyID, ProjectIDs: request.ProjectIDs}
	if err := h.useCase.CreateTaskStage(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListTaskStages(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	var projectID *int64
	if raw := r.URL.Query().Get("project_id"); raw != "" {
		value, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || value <= 0 {
			response.Error(w, platformerrors.BadRequest("invalid project_id"))
			return
		}
		projectID = &value
	}
	values, err := h.useCase.ListTaskStages(r.Context(), &companyID, projectID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) UpdateTaskStage(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetTaskStage(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateTaskStageRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value.Name, value.Sequence, value.Fold, value.Color, value.ProjectIDs = request.Name, request.Sequence, request.Fold, request.Color, request.ProjectIDs
	if err := h.useCase.UpdateTaskStage(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}
func (h *Handler) DeleteTaskStage(w http.ResponseWriter, r *http.Request) {
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
	if err := h.useCase.DeleteTaskStage(r.Context(), companyID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) CreateMilestone(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	projectID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	var request CreateMilestoneRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := &projectdomain.Milestone{Name: request.Name, ProjectID: projectID, DateDeadline: request.DateDeadline, Sequence: request.Sequence}
	if err := h.useCase.CreateMilestone(r.Context(), companyID, value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListMilestones(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	projectID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListMilestones(r.Context(), companyID, projectID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) UpdateMilestone(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.useCase.GetMilestone(r.Context(), companyID, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateMilestoneRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value.Name, value.DateDeadline, value.Sequence = request.Name, request.DateDeadline, request.Sequence
	if err := h.useCase.UpdateMilestone(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}
func (h *Handler) DeleteMilestone(w http.ResponseWriter, r *http.Request) {
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
	if err := h.useCase.DeleteMilestone(r.Context(), companyID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) CreateTaskTag(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	var request CreateTaskTagRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value := &projectdomain.TaskTag{Name: request.Name, Color: request.Color}
	if err := h.useCase.CreateTaskTag(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, value)
}

func (h *Handler) ListTaskTags(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	values, err := h.useCase.ListTaskTags(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, values)
}

func (h *Handler) UpdateTaskTag(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	value, err := h.useCase.GetTaskTag(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	var request UpdateTaskTagRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	value.Name, value.Color = request.Name, request.Color
	if err := h.useCase.UpdateTaskTag(r.Context(), value); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, value)
}
func (h *Handler) DeleteTaskTag(w http.ResponseWriter, r *http.Request) {
	if _, err := requestCompanyID(r); err != nil {
		response.Error(w, err)
		return
	}
	id, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.DeleteTaskTag(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) AddDependency(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	taskID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	var request DependencyRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.AddDependency(r.Context(), companyID, taskID, request.DependsOnTaskID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) RemoveDependency(w http.ResponseWriter, r *http.Request) {
	companyID, err := requestCompanyID(r)
	if err != nil {
		response.Error(w, err)
		return
	}
	taskID, err := pathID(r, "id")
	if err != nil {
		response.Error(w, err)
		return
	}
	var request DependencyRequest
	if err := decodeBody(r, &request); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.RemoveDependency(r.Context(), companyID, taskID, request.DependsOnTaskID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}
