package projecthttp

import (
	"time"

	projectdomain "cashflow_backend/internal/domain/project"
	projectusecase "cashflow_backend/internal/usecase/project"
)

type CreateProjectRequest struct {
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	PartnerID         *int64     `json:"partner_id"`
	ManagerID         *int64     `json:"manager_id"`
	StageID           *int64     `json:"stage_id"`
	DateStart         *time.Time `json:"date_start"`
	DateEnd           *time.Time `json:"date_end"`
	AllowMilestones   bool       `json:"allow_milestones"`
	AllowSubtasks     bool       `json:"allow_subtasks"`
	AllowDependencies bool       `json:"allow_dependencies"`
	AnalyticAccountID *int64     `json:"analytic_account_id"`
}

func (r CreateProjectRequest) ToInput(companyID int64) projectusecase.CreateProjectInput {
	return projectusecase.CreateProjectInput{
		Name: r.Name, Description: r.Description, PartnerID: r.PartnerID, ManagerID: r.ManagerID,
		StageID: r.StageID, DateStart: r.DateStart, DateEnd: r.DateEnd,
		AllowMilestones: r.AllowMilestones, AllowSubtasks: r.AllowSubtasks,
		AllowDependencies: r.AllowDependencies, AnalyticAccountID: r.AnalyticAccountID, CompanyID: companyID,
	}
}

type CreateTaskRequest struct {
	Name           string                     `json:"name"`
	ProjectID      int64                      `json:"project_id"`
	StageID        int64                      `json:"stage_id"`
	AssigneeIDs    []int64                    `json:"assignee_ids"`
	ParentID       *int64                     `json:"parent_id"`
	Priority       projectdomain.TaskPriority `json:"priority"`
	DateDeadline   *time.Time                 `json:"date_deadline"`
	Description    string                     `json:"description"`
	MilestoneID    *int64                     `json:"milestone_id"`
	TagIDs         []int64                    `json:"tag_ids"`
	Sequence       int                        `json:"sequence"`
	AllocatedHours float64                    `json:"allocated_hours"`
}

func (r CreateTaskRequest) ToInput(companyID int64) projectusecase.CreateTaskInput {
	return projectusecase.CreateTaskInput{
		Name: r.Name, ProjectID: r.ProjectID, StageID: r.StageID, AssigneeIDs: r.AssigneeIDs,
		ParentID: r.ParentID, Priority: r.Priority, DateDeadline: r.DateDeadline, Description: r.Description,
		MilestoneID: r.MilestoneID, TagIDs: r.TagIDs, Sequence: r.Sequence,
		AllocatedHours: r.AllocatedHours, CompanyID: companyID,
	}
}

type DependencyRequest struct {
	DependsOnTaskID int64 `json:"depends_on_task_id"`
}

type CreateStageRequest struct {
	Name      string `json:"name"`
	Sequence  int    `json:"sequence"`
	Fold      bool   `json:"fold"`
	Color     int    `json:"color"`
	CompanyID *int64 `json:"company_id"`
}

type CreateTaskStageRequest struct {
	Name       string  `json:"name"`
	Sequence   int     `json:"sequence"`
	Fold       bool    `json:"fold"`
	Color      int     `json:"color"`
	ProjectIDs []int64 `json:"project_ids"`
}

type CreateMilestoneRequest struct {
	Name         string     `json:"name"`
	DateDeadline *time.Time `json:"date_deadline"`
	Sequence     int        `json:"sequence"`
}

type CreateTaskTagRequest struct {
	Name  string `json:"name"`
	Color int    `json:"color"`
}

type UpdateProjectRequest = CreateProjectRequest
type UpdateTaskRequest = CreateTaskRequest
type UpdateStageRequest = CreateStageRequest
type UpdateTaskStageRequest = CreateTaskStageRequest
type UpdateMilestoneRequest = CreateMilestoneRequest
type UpdateTaskTagRequest = CreateTaskTagRequest

type ProjectResponse struct{ *projectdomain.Project }
type TaskResponse struct{ *projectdomain.Task }
