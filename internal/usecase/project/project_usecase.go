package projectusecase

import (
	"context"
	"time"

	"cashflow_backend/internal/domain/project"
	platformerrors "cashflow_backend/internal/platform/errors"
)

type CreateProjectInput struct {
	Name              string
	Description       string
	PartnerID         *int64
	ManagerID         *int64
	StageID           *int64
	DateStart         *time.Time
	DateEnd           *time.Time
	AllowMilestones   bool
	AllowSubtasks     bool
	AllowDependencies bool
	AnalyticAccountID *int64
	CompanyID         int64
}

type CreateTaskInput struct {
	Name           string
	ProjectID      int64
	StageID        int64
	AssigneeIDs    []int64
	ParentID       *int64
	Priority       project.TaskPriority
	DateDeadline   *time.Time
	Description    string
	MilestoneID    *int64
	TagIDs         []int64
	Sequence       int
	AllocatedHours float64
	CompanyID      int64
}

type Service struct {
	repo project.Repository
}

func New(repo project.Repository) *Service { return &Service{repo: repo} }

func (s *Service) CreateProject(ctx context.Context, input CreateProjectInput) (*project.Project, error) {
	value := &project.Project{
		Name: input.Name, Description: input.Description, PartnerID: input.PartnerID,
		ManagerID: input.ManagerID, StageID: input.StageID, DateStart: input.DateStart,
		DateEnd: input.DateEnd, AllowMilestones: input.AllowMilestones, AllowSubtasks: input.AllowSubtasks,
		AllowDependencies: input.AllowDependencies, AnalyticAccountID: input.AnalyticAccountID, CompanyID: input.CompanyID,
	}
	if err := value.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.CreateProject(ctx, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *Service) GetProject(ctx context.Context, companyID, id int64) (*project.Project, error) {
	return s.repo.GetProjectByID(ctx, companyID, id)
}

func (s *Service) ListProjects(ctx context.Context, companyID int64) ([]project.Project, error) {
	return s.repo.ListProjects(ctx, companyID)
}

func (s *Service) UpdateProject(ctx context.Context, value *project.Project) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateProject(ctx, value)
}

func (s *Service) DeleteProject(ctx context.Context, companyID, id int64) error {
	return s.repo.DeleteProject(ctx, companyID, id)
}

func (s *Service) ListTasks(ctx context.Context, companyID, projectID int64) ([]project.Task, error) {
	if _, err := s.repo.GetProjectByID(ctx, companyID, projectID); err != nil {
		return nil, err
	}
	return s.repo.ListTasks(ctx, companyID, project.TaskFilter{ProjectID: &projectID, IncludeDone: true})
}

func (s *Service) GetTask(ctx context.Context, companyID, id int64) (*project.Task, error) {
	return s.repo.GetTaskByID(ctx, companyID, id)
}

func (s *Service) CreateProjectStage(ctx context.Context, value *project.ProjectStage) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return s.repo.CreateProjectStage(ctx, value)
}

func (s *Service) ListProjectStages(ctx context.Context, companyID *int64) ([]project.ProjectStage, error) {
	return s.repo.ListProjectStages(ctx, companyID)
}

func (s *Service) GetProjectStage(ctx context.Context, companyID, id int64) (*project.ProjectStage, error) {
	return s.repo.GetProjectStageByID(ctx, &companyID, id)
}

func (s *Service) UpdateProjectStage(ctx context.Context, value *project.ProjectStage) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateProjectStage(ctx, value)
}
func (s *Service) DeleteProjectStage(ctx context.Context, companyID, id int64) error {
	return s.repo.DeleteProjectStage(ctx, &companyID, id)
}

func (s *Service) CreateTaskStage(ctx context.Context, value *project.TaskStage) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return s.repo.CreateTaskStage(ctx, value)
}

func (s *Service) ListTaskStages(ctx context.Context, companyID *int64, projectID *int64) ([]project.TaskStage, error) {
	return s.repo.ListTaskStages(ctx, companyID, projectID)
}

func (s *Service) GetTaskStage(ctx context.Context, companyID, id int64) (*project.TaskStage, error) {
	return s.repo.GetTaskStageByID(ctx, &companyID, id)
}

func (s *Service) UpdateTaskStage(ctx context.Context, value *project.TaskStage) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateTaskStage(ctx, value)
}
func (s *Service) DeleteTaskStage(ctx context.Context, companyID, id int64) error {
	return s.repo.DeleteTaskStage(ctx, &companyID, id)
}

func (s *Service) CreateMilestone(ctx context.Context, companyID int64, value *project.Milestone) error {
	if err := value.Validate(); err != nil {
		return err
	}
	projectValue, err := s.repo.GetProjectByID(ctx, companyID, value.ProjectID)
	if err != nil {
		return err
	}
	if !projectValue.AllowMilestones {
		return platformerrors.Conflict("milestones are disabled for this project")
	}
	return s.repo.CreateMilestone(ctx, value)
}

func (s *Service) ListMilestones(ctx context.Context, companyID, projectID int64) ([]project.Milestone, error) {
	if _, err := s.repo.GetProjectByID(ctx, companyID, projectID); err != nil {
		return nil, err
	}
	return s.repo.ListMilestones(ctx, companyID, projectID)
}

func (s *Service) GetMilestone(ctx context.Context, companyID, id int64) (*project.Milestone, error) {
	return s.repo.GetMilestoneByID(ctx, companyID, id)
}

func (s *Service) UpdateMilestone(ctx context.Context, value *project.Milestone) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateMilestone(ctx, value)
}
func (s *Service) DeleteMilestone(ctx context.Context, companyID, id int64) error {
	return s.repo.DeleteMilestone(ctx, companyID, id)
}

func (s *Service) CreateTaskTag(ctx context.Context, value *project.TaskTag) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return s.repo.CreateTaskTag(ctx, value)
}

func (s *Service) ListTaskTags(ctx context.Context) ([]project.TaskTag, error) {
	return s.repo.ListTaskTags(ctx)
}

func (s *Service) GetTaskTag(ctx context.Context, id int64) (*project.TaskTag, error) {
	return s.repo.GetTaskTagByID(ctx, id)
}

func (s *Service) UpdateTaskTag(ctx context.Context, value *project.TaskTag) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateTaskTag(ctx, value)
}
func (s *Service) DeleteTaskTag(ctx context.Context, id int64) error {
	return s.repo.DeleteTaskTag(ctx, id)
}

func (s *Service) CreateTask(ctx context.Context, input CreateTaskInput) (*project.Task, error) {
	parent, err := s.repo.GetProjectByID(ctx, input.CompanyID, input.ProjectID)
	if err != nil {
		return nil, err
	}
	if input.ParentID != nil && !parent.AllowSubtasks {
		return nil, platformerrors.Conflict("subtasks are disabled for this project")
	}
	if input.MilestoneID != nil && !parent.AllowMilestones {
		return nil, platformerrors.Conflict("milestones are disabled for this project")
	}
	value := &project.Task{
		Name: input.Name, ProjectID: input.ProjectID, StageID: input.StageID,
		AssigneeIDs: append([]int64(nil), input.AssigneeIDs...), ParentID: input.ParentID,
		Priority: input.Priority, DateDeadline: input.DateDeadline, Description: input.Description,
		MilestoneID: input.MilestoneID, TagIDs: append([]int64(nil), input.TagIDs...), Sequence: input.Sequence,
		AllocatedHours: input.AllocatedHours, CompanyID: input.CompanyID,
	}
	if err := value.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.CreateTask(ctx, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *Service) UpdateTask(ctx context.Context, value *project.Task) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateTask(ctx, value)
}

func (s *Service) DeleteTask(ctx context.Context, companyID, id int64) error {
	return s.repo.DeleteTask(ctx, companyID, id)
}

func (s *Service) AddDependency(ctx context.Context, companyID, taskID, dependsOnTaskID int64) error {
	task, err := s.repo.GetTaskByID(ctx, companyID, taskID)
	if err != nil {
		return err
	}
	projectValue, err := s.repo.GetProjectByID(ctx, companyID, task.ProjectID)
	if err != nil {
		return err
	}
	if !projectValue.AllowDependencies {
		return platformerrors.Conflict("task dependencies are disabled for this project")
	}
	dependency, err := s.repo.GetTaskByID(ctx, companyID, dependsOnTaskID)
	if err != nil {
		return err
	}
	if dependency.IsClosed() {
		return s.repo.AddTaskDependency(ctx, companyID, taskID, dependsOnTaskID)
	}
	if err := s.repo.AddTaskDependency(ctx, companyID, taskID, dependsOnTaskID); err != nil {
		return err
	}
	if !task.IsClosed() && task.State != project.TaskWaiting {
		task.State = project.TaskWaiting
		return s.repo.UpdateTask(ctx, task)
	}
	return nil
}

func (s *Service) RemoveDependency(ctx context.Context, companyID, taskID, dependsOnTaskID int64) error {
	if err := s.repo.RemoveTaskDependency(ctx, companyID, taskID, dependsOnTaskID); err != nil {
		return err
	}
	task, err := s.repo.GetTaskByID(ctx, companyID, taskID)
	if err != nil {
		return err
	}
	if task.IsClosed() || len(task.DependOnIDs) > 0 || task.State != project.TaskWaiting {
		return nil
	}
	task.State = project.TaskInProgress
	return s.repo.UpdateTask(ctx, task)
}

func (s *Service) ReachMilestone(ctx context.Context, companyID, milestoneID int64) error {
	milestone, err := s.repo.GetMilestoneByID(ctx, companyID, milestoneID)
	if err != nil {
		return err
	}
	projectValue, err := s.repo.GetProjectByID(ctx, companyID, milestone.ProjectID)
	if err != nil {
		return err
	}
	if !projectValue.AllowMilestones {
		return platformerrors.Conflict("milestones are disabled for this project")
	}
	tasks, err := s.repo.ListTasks(ctx, companyID, project.TaskFilter{ProjectID: &milestone.ProjectID, IncludeDone: true})
	if err != nil {
		return err
	}
	hasOpenTasks := false
	for _, task := range tasks {
		if task.MilestoneID != nil && *task.MilestoneID == milestoneID && !task.IsClosed() {
			hasOpenTasks = true
			break
		}
	}
	if err := milestone.Reach(time.Now().UTC(), hasOpenTasks); err != nil {
		return err
	}
	return s.repo.UpdateMilestone(ctx, milestone)
}
