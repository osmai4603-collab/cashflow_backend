package project

import "context"

// Repository defines project-management persistence operations.
type Repository interface {
	CreateProject(ctx context.Context, project *Project) error
	GetProjectByID(ctx context.Context, companyID, id int64) (*Project, error)
	UpdateProject(ctx context.Context, project *Project) error
	DeleteProject(ctx context.Context, companyID, id int64) error
	ListProjects(ctx context.Context, companyID int64) ([]Project, error)

	CreateProjectStage(ctx context.Context, stage *ProjectStage) error
	GetProjectStageByID(ctx context.Context, companyID *int64, id int64) (*ProjectStage, error)
	UpdateProjectStage(ctx context.Context, stage *ProjectStage) error
	DeleteProjectStage(ctx context.Context, companyID *int64, id int64) error
	ListProjectStages(ctx context.Context, companyID *int64) ([]ProjectStage, error)

	CreateTaskStage(ctx context.Context, stage *TaskStage) error
	GetTaskStageByID(ctx context.Context, companyID *int64, id int64) (*TaskStage, error)
	UpdateTaskStage(ctx context.Context, stage *TaskStage) error
	DeleteTaskStage(ctx context.Context, companyID *int64, id int64) error
	ListTaskStages(ctx context.Context, companyID *int64, projectID *int64) ([]TaskStage, error)

	CreateTask(ctx context.Context, task *Task) error
	GetTaskByID(ctx context.Context, companyID, id int64) (*Task, error)
	UpdateTask(ctx context.Context, task *Task) error
	DeleteTask(ctx context.Context, companyID, id int64) error
	ListTasks(ctx context.Context, companyID int64, filter TaskFilter) ([]Task, error)
	ListKanban(ctx context.Context, companyID, projectID int64) ([]TaskStageBucket, error)

	SetTaskAssignees(ctx context.Context, companyID, taskID int64, userIDs []int64) error
	SetTaskTags(ctx context.Context, companyID, taskID int64, tagIDs []int64) error
	AddTaskDependency(ctx context.Context, companyID, taskID, dependsOnTaskID int64) error
	RemoveTaskDependency(ctx context.Context, companyID, taskID, dependsOnTaskID int64) error

	CreateMilestone(ctx context.Context, milestone *Milestone) error
	GetMilestoneByID(ctx context.Context, companyID, id int64) (*Milestone, error)
	UpdateMilestone(ctx context.Context, milestone *Milestone) error
	DeleteMilestone(ctx context.Context, companyID, id int64) error
	ListMilestones(ctx context.Context, companyID, projectID int64) ([]Milestone, error)

	CreateTaskTag(ctx context.Context, tag *TaskTag) error
	GetTaskTagByID(ctx context.Context, id int64) (*TaskTag, error)
	UpdateTaskTag(ctx context.Context, tag *TaskTag) error
	DeleteTaskTag(ctx context.Context, id int64) error
	ListTaskTags(ctx context.Context) ([]TaskTag, error)
}

type TaskFilter struct {
	ProjectID   *int64
	StageID     *int64
	State       *TaskState
	AssigneeID  *int64
	ParentID    *int64
	IncludeDone bool
}

type TaskStageBucket struct {
	Stage StageSummary `json:"stage"`
	Tasks []Task        `json:"tasks"`
}

type StageSummary struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Sequence int    `json:"sequence"`
	Fold     bool   `json:"fold"`
	Color    int    `json:"color"`
}
