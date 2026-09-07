package projectstorage

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"cashflow_backend/internal/domain/project"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// MemoryRepo is a thread-safe repository used by unit tests and local development.
type MemoryRepo struct {
	mu sync.RWMutex
	projects map[int64]*project.Project
	projectStages map[int64]*project.ProjectStage
	taskStages map[int64]*project.TaskStage
	tasks map[int64]*project.Task
	milestones map[int64]*project.Milestone
	tags map[int64]*project.TaskTag
	assignees map[int64][]int64
	dependencies map[int64][]int64
	taskTags map[int64][]int64
	nextProjectID, nextProjectStageID, nextTaskStageID, nextTaskID, nextMilestoneID, nextTagID int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		projects: make(map[int64]*project.Project), projectStages: make(map[int64]*project.ProjectStage),
		taskStages: make(map[int64]*project.TaskStage), tasks: make(map[int64]*project.Task),
		milestones: make(map[int64]*project.Milestone), tags: make(map[int64]*project.TaskTag),
		assignees: make(map[int64][]int64), dependencies: make(map[int64][]int64), taskTags: make(map[int64][]int64),
	}
}

func now() time.Time { return time.Now().UTC() }
func notFound(kind string, id int64) error { return platformerrors.NotFound(fmt.Sprintf("%s with ID %d not found", kind, id)) }
func companyAllowed(scope int64, companyID int64) bool { return scope <= 0 || companyID == scope }
func optionalCompanyAllowed(scope int64, companyID *int64) bool { return scope <= 0 || companyID == nil || *companyID == scope }

func (r *MemoryRepo) CreateProject(_ context.Context, value *project.Project) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if err := value.Validate(); err != nil { return err }
	r.nextProjectID++; value.ID = r.nextProjectID; value.Active = true; value.CreatedAt = now(); value.UpdatedAt = value.CreatedAt
	clone := *value; r.projects[value.ID] = &clone; return nil
}
func (r *MemoryRepo) GetProjectByID(_ context.Context, companyID, id int64) (*project.Project, error) {
	r.mu.RLock(); defer r.mu.RUnlock()
	value, ok := r.projects[id]; if !ok || !companyAllowed(companyID, value.CompanyID) { return nil, notFound("project", id) }
	clone := *value; return &clone, nil
}
func (r *MemoryRepo) UpdateProject(_ context.Context, value *project.Project) error {
	r.mu.Lock(); defer r.mu.Unlock()
	old, ok := r.projects[value.ID]; if !ok { return notFound("project", value.ID) }
	if err := value.Validate(); err != nil { return err }; if value.CompanyID != old.CompanyID { return platformerrors.Conflict("project company cannot be changed") }
	value.CreatedAt = old.CreatedAt; value.UpdatedAt = now(); clone := *value; r.projects[value.ID] = &clone; return nil
}
func (r *MemoryRepo) DeleteProject(_ context.Context, companyID, id int64) error {
	r.mu.Lock(); defer r.mu.Unlock(); value, ok := r.projects[id]; if !ok || !companyAllowed(companyID, value.CompanyID) { return notFound("project", id) }; delete(r.projects, id); return nil
}
func (r *MemoryRepo) ListProjects(_ context.Context, companyID int64) ([]project.Project, error) {
	r.mu.RLock(); defer r.mu.RUnlock(); result := make([]project.Project, 0)
	for _, value := range r.projects { if companyAllowed(companyID, value.CompanyID) { result = append(result, *value) } }
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil
}

func (r *MemoryRepo) CreateProjectStage(_ context.Context, value *project.ProjectStage) error { r.mu.Lock(); defer r.mu.Unlock(); if err := value.Validate(); err != nil { return err }; r.nextProjectStageID++; value.ID = r.nextProjectStageID; value.Active = true; value.CreatedAt = now(); value.UpdatedAt = value.CreatedAt; clone := *value; r.projectStages[value.ID] = &clone; return nil }
func (r *MemoryRepo) GetProjectStageByID(_ context.Context, companyID *int64, id int64) (*project.ProjectStage, error) { r.mu.RLock(); defer r.mu.RUnlock(); value, ok := r.projectStages[id]; if !ok || !optionalCompanyAllowedPtr(companyID, value.CompanyID) { return nil, notFound("project stage", id) }; clone := *value; return &clone, nil }
func (r *MemoryRepo) UpdateProjectStage(_ context.Context, value *project.ProjectStage) error { r.mu.Lock(); defer r.mu.Unlock(); old, ok := r.projectStages[value.ID]; if !ok { return notFound("project stage", value.ID) }; if err := value.Validate(); err != nil { return err }; value.CreatedAt = old.CreatedAt; value.UpdatedAt = now(); clone := *value; r.projectStages[value.ID] = &clone; return nil }
func (r *MemoryRepo) DeleteProjectStage(_ context.Context, companyID *int64, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); value, ok := r.projectStages[id]; if !ok || !optionalCompanyAllowedPtr(companyID, value.CompanyID) { return notFound("project stage", id) }; for _, p := range r.projects { if p.StageID != nil && *p.StageID == id { return platformerrors.Conflict("project stage is in use") } }; delete(r.projectStages, id); return nil }
func (r *MemoryRepo) ListProjectStages(_ context.Context, companyID *int64) ([]project.ProjectStage, error) { r.mu.RLock(); defer r.mu.RUnlock(); result := make([]project.ProjectStage, 0); for _, value := range r.projectStages { if optionalCompanyAllowedPtr(companyID, value.CompanyID) { result = append(result, *value) } }; sort.Slice(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence }); return result, nil }

func (r *MemoryRepo) CreateTaskStage(_ context.Context, value *project.TaskStage) error { r.mu.Lock(); defer r.mu.Unlock(); if err := value.Validate(); err != nil { return err }; r.nextTaskStageID++; value.ID = r.nextTaskStageID; value.Active = true; value.CreatedAt = now(); value.UpdatedAt = value.CreatedAt; clone := *value; clone.ProjectIDs = append([]int64(nil), value.ProjectIDs...); r.taskStages[value.ID] = &clone; return nil }
func (r *MemoryRepo) GetTaskStageByID(_ context.Context, companyID *int64, id int64) (*project.TaskStage, error) { r.mu.RLock(); defer r.mu.RUnlock(); value, ok := r.taskStages[id]; if !ok || !optionalCompanyAllowedPtr(companyID, value.CompanyID) { return nil, notFound("task stage", id) }; clone := *value; clone.ProjectIDs = append([]int64(nil), value.ProjectIDs...); return &clone, nil }
func (r *MemoryRepo) UpdateTaskStage(_ context.Context, value *project.TaskStage) error { r.mu.Lock(); defer r.mu.Unlock(); old, ok := r.taskStages[value.ID]; if !ok { return notFound("task stage", value.ID) }; if err := value.Validate(); err != nil { return err }; value.CreatedAt = old.CreatedAt; value.UpdatedAt = now(); clone := *value; clone.ProjectIDs = append([]int64(nil), value.ProjectIDs...); r.taskStages[value.ID] = &clone; return nil }
func (r *MemoryRepo) DeleteTaskStage(_ context.Context, companyID *int64, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); value, ok := r.taskStages[id]; if !ok || !optionalCompanyAllowedPtr(companyID, value.CompanyID) { return notFound("task stage", id) }; for _, task := range r.tasks { if task.StageID == id { return platformerrors.Conflict("task stage is in use") } }; delete(r.taskStages, id); return nil }
func (r *MemoryRepo) ListTaskStages(_ context.Context, companyID *int64, projectID *int64) ([]project.TaskStage, error) { r.mu.RLock(); defer r.mu.RUnlock(); result := make([]project.TaskStage, 0); for _, value := range r.taskStages { if !optionalCompanyAllowedPtr(companyID, value.CompanyID) || (projectID != nil && !contains(value.ProjectIDs, *projectID)) { continue }; clone := *value; clone.ProjectIDs = append([]int64(nil), value.ProjectIDs...); result = append(result, clone) }; sort.Slice(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence }); return result, nil }

func (r *MemoryRepo) CreateTask(_ context.Context, value *project.Task) error { r.mu.Lock(); defer r.mu.Unlock(); if err := value.Validate(); err != nil { return err }; p, ok := r.projects[value.ProjectID]; if !ok || p.CompanyID != value.CompanyID { return platformerrors.Validation("task project must belong to the same company", nil) }; stage, ok := r.taskStages[value.StageID]; if !ok || !optionalCompanyAllowedPtr(&value.CompanyID, stage.CompanyID) || (len(stage.ProjectIDs) > 0 && !contains(stage.ProjectIDs, value.ProjectID)) { return platformerrors.Validation("task stage must belong to the project and company", nil) }; if value.ParentID != nil && !r.validParentLocked(value.ID, *value.ParentID, value.ProjectID) { return platformerrors.Validation("parent task must belong to the same project", nil) }; r.nextTaskID++; value.ID = r.nextTaskID; value.CreatedAt = now(); value.UpdatedAt = value.CreatedAt; clone := *value; r.tasks[value.ID] = &clone; r.assignees[value.ID] = append([]int64(nil), value.AssigneeIDs...); r.dependencies[value.ID] = append([]int64(nil), value.DependOnIDs...); r.taskTags[value.ID] = append([]int64(nil), value.TagIDs...); return nil }
func (r *MemoryRepo) GetTaskByID(_ context.Context, companyID, id int64) (*project.Task, error) { r.mu.RLock(); defer r.mu.RUnlock(); value, ok := r.tasks[id]; if !ok || !companyAllowed(companyID, value.CompanyID) { return nil, notFound("task", id) }; return r.cloneTaskLocked(value), nil }
func (r *MemoryRepo) UpdateTask(_ context.Context, value *project.Task) error { r.mu.Lock(); defer r.mu.Unlock(); old, ok := r.tasks[value.ID]; if !ok || old.CompanyID != value.CompanyID { return notFound("task", value.ID) }; if err := value.Validate(); err != nil { return err }; if value.ParentID != nil && !r.validParentLocked(value.ID, *value.ParentID, value.ProjectID) { return platformerrors.Validation("parent task must belong to the same project", nil) }; value.CreatedAt = old.CreatedAt; value.UpdatedAt = now(); clone := *value; r.tasks[value.ID] = &clone; r.assignees[value.ID] = append([]int64(nil), value.AssigneeIDs...); r.taskTags[value.ID] = append([]int64(nil), value.TagIDs...); return nil }
func (r *MemoryRepo) DeleteTask(_ context.Context, companyID, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); value, ok := r.tasks[id]; if !ok || !companyAllowed(companyID, value.CompanyID) { return notFound("task", id) }; delete(r.tasks, id); delete(r.assignees, id); delete(r.dependencies, id); delete(r.taskTags, id); return nil }
func (r *MemoryRepo) ListTasks(_ context.Context, companyID int64, filter project.TaskFilter) ([]project.Task, error) { r.mu.RLock(); defer r.mu.RUnlock(); result := make([]project.Task, 0); for _, value := range r.tasks { if !companyAllowed(companyID, value.CompanyID) || (filter.ProjectID != nil && value.ProjectID != *filter.ProjectID) || (filter.StageID != nil && value.StageID != *filter.StageID) || (filter.State != nil && value.State != *filter.State) || (filter.AssigneeID != nil && !contains(r.assignees[value.ID], *filter.AssigneeID)) || (filter.ParentID != nil && (value.ParentID == nil || *value.ParentID != *filter.ParentID)) || (!filter.IncludeDone && value.IsClosed()) { continue }; result = append(result, *r.cloneTaskLocked(value)) }; sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil }
func (r *MemoryRepo) ListKanban(_ context.Context, companyID, projectID int64) ([]project.TaskStageBucket, error) { r.mu.RLock(); defer r.mu.RUnlock(); targetProject, ok := r.projects[projectID]; if !ok || targetProject.CompanyID != companyID { return nil, notFound("project", projectID) }; buckets := make(map[int64]*project.TaskStageBucket); for _, stage := range r.taskStages { if !optionalCompanyAllowedPtr(&targetProject.CompanyID, stage.CompanyID) || (len(stage.ProjectIDs) > 0 && !contains(stage.ProjectIDs, projectID)) { continue }; buckets[stage.ID] = &project.TaskStageBucket{Stage: project.StageSummary{ID: stage.ID, Name: stage.Name, Sequence: stage.Sequence, Fold: stage.Fold, Color: stage.Color}} }; for _, task := range r.tasks { if task.CompanyID != companyID || task.ProjectID != projectID { continue }; bucket, ok := buckets[task.StageID]; if !ok { continue }; bucket.Tasks = append(bucket.Tasks, *r.cloneTaskLocked(task)) }; result := make([]project.TaskStageBucket, 0, len(buckets)); for _, bucket := range buckets { sort.Slice(bucket.Tasks, func(i, j int) bool { return bucket.Tasks[i].Sequence < bucket.Tasks[j].Sequence }); result = append(result, *bucket) }; sort.Slice(result, func(i, j int) bool { return result[i].Stage.Sequence < result[j].Stage.Sequence }); return result, nil }

func (r *MemoryRepo) SetTaskAssignees(_ context.Context, companyID, taskID int64, userIDs []int64) error { r.mu.Lock(); defer r.mu.Unlock(); if err := r.ensureTaskCompanyLocked(companyID, taskID); err != nil { return err }; r.assignees[taskID] = append([]int64(nil), userIDs...); r.tasks[taskID].AssigneeIDs = append([]int64(nil), userIDs...); return nil }
func (r *MemoryRepo) SetTaskTags(_ context.Context, companyID, taskID int64, tagIDs []int64) error { r.mu.Lock(); defer r.mu.Unlock(); if err := r.ensureTaskCompanyLocked(companyID, taskID); err != nil { return err }; for _, id := range tagIDs { if _, ok := r.tags[id]; !ok { return notFound("task tag", id) } }; r.taskTags[taskID] = append([]int64(nil), tagIDs...); r.tasks[taskID].TagIDs = append([]int64(nil), tagIDs...); return nil }
func (r *MemoryRepo) AddTaskDependency(_ context.Context, companyID, taskID, dependsOnTaskID int64) error { r.mu.Lock(); defer r.mu.Unlock(); if err := r.ensureTaskCompanyLocked(companyID, taskID); err != nil { return err }; dependency, err := r.taskForCompanyLocked(companyID, dependsOnTaskID); if err != nil { return err }; if dependency.ProjectID != r.tasks[taskID].ProjectID { return platformerrors.Conflict("task dependencies must belong to the same project") }; if project.DependencyCreatesCycle(taskID, dependsOnTaskID, r.dependencies) { return platformerrors.Conflict("task dependency would create a cycle") }; if !contains(r.dependencies[taskID], dependsOnTaskID) { r.dependencies[taskID] = append(r.dependencies[taskID], dependsOnTaskID); r.tasks[taskID].DependOnIDs = append(r.tasks[taskID].DependOnIDs, dependsOnTaskID) }; return nil }
func (r *MemoryRepo) RemoveTaskDependency(_ context.Context, companyID, taskID, dependsOnTaskID int64) error { r.mu.Lock(); defer r.mu.Unlock(); if err := r.ensureTaskCompanyLocked(companyID, taskID); err != nil { return err }; values := r.dependencies[taskID]; next := values[:0]; for _, value := range values { if value != dependsOnTaskID { next = append(next, value) } }; r.dependencies[taskID] = next; r.tasks[taskID].DependOnIDs = append([]int64(nil), next...); return nil }

func (r *MemoryRepo) CreateMilestone(_ context.Context, value *project.Milestone) error { r.mu.Lock(); defer r.mu.Unlock(); if err := value.Validate(); err != nil { return err }; if _, ok := r.projects[value.ProjectID]; !ok { return notFound("project", value.ProjectID) }; r.nextMilestoneID++; value.ID = r.nextMilestoneID; value.CreatedAt = now(); value.UpdatedAt = value.CreatedAt; clone := *value; r.milestones[value.ID] = &clone; return nil }
func (r *MemoryRepo) GetMilestoneByID(_ context.Context, companyID, id int64) (*project.Milestone, error) { r.mu.RLock(); defer r.mu.RUnlock(); value, ok := r.milestones[id]; if !ok { return nil, notFound("milestone", id) }; p := r.projects[value.ProjectID]; if p == nil || p.CompanyID != companyID { return nil, notFound("milestone", id) }; clone := *value; return &clone, nil }
func (r *MemoryRepo) UpdateMilestone(_ context.Context, value *project.Milestone) error { r.mu.Lock(); defer r.mu.Unlock(); old, ok := r.milestones[value.ID]; if !ok { return notFound("milestone", value.ID) }; if err := value.Validate(); err != nil { return err }; value.CreatedAt = old.CreatedAt; value.UpdatedAt = now(); clone := *value; r.milestones[value.ID] = &clone; return nil }
func (r *MemoryRepo) DeleteMilestone(_ context.Context, companyID, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); if _, err := r.milestoneForCompanyLocked(companyID, id); err != nil { return err }; delete(r.milestones, id); for _, task := range r.tasks { if task.MilestoneID != nil && *task.MilestoneID == id { task.MilestoneID = nil } }; return nil }
func (r *MemoryRepo) ListMilestones(_ context.Context, companyID, projectID int64) ([]project.Milestone, error) { r.mu.RLock(); defer r.mu.RUnlock(); result := make([]project.Milestone, 0); p := r.projects[projectID]; if p == nil || p.CompanyID != companyID { return result, nil }; for _, value := range r.milestones { if value.ProjectID == projectID { result = append(result, *value) } }; sort.Slice(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence }); return result, nil }

func (r *MemoryRepo) CreateTaskTag(_ context.Context, value *project.TaskTag) error { r.mu.Lock(); defer r.mu.Unlock(); if err := value.Validate(); err != nil { return err }; r.nextTagID++; value.ID = r.nextTagID; value.Active = true; value.CreatedAt = now(); value.UpdatedAt = value.CreatedAt; clone := *value; r.tags[value.ID] = &clone; return nil }
func (r *MemoryRepo) GetTaskTagByID(_ context.Context, id int64) (*project.TaskTag, error) { r.mu.RLock(); defer r.mu.RUnlock(); value, ok := r.tags[id]; if !ok { return nil, notFound("task tag", id) }; clone := *value; return &clone, nil }
func (r *MemoryRepo) UpdateTaskTag(_ context.Context, value *project.TaskTag) error { r.mu.Lock(); defer r.mu.Unlock(); old, ok := r.tags[value.ID]; if !ok { return notFound("task tag", value.ID) }; if err := value.Validate(); err != nil { return err }; value.CreatedAt = old.CreatedAt; value.UpdatedAt = now(); clone := *value; r.tags[value.ID] = &clone; return nil }
func (r *MemoryRepo) DeleteTaskTag(_ context.Context, id int64) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.tags[id]; !ok { return notFound("task tag", id) }; delete(r.tags, id); for taskID, ids := range r.taskTags { r.taskTags[taskID] = remove(ids, id) }; return nil }
func (r *MemoryRepo) ListTaskTags(_ context.Context) ([]project.TaskTag, error) { r.mu.RLock(); defer r.mu.RUnlock(); result := make([]project.TaskTag, 0, len(r.tags)); for _, value := range r.tags { result = append(result, *value) }; sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID }); return result, nil }

func (r *MemoryRepo) cloneTaskLocked(value *project.Task) *project.Task { clone := *value; clone.AssigneeIDs = append([]int64(nil), r.assignees[value.ID]...); clone.DependOnIDs = append([]int64(nil), r.dependencies[value.ID]...); clone.TagIDs = append([]int64(nil), r.taskTags[value.ID]...); return &clone }
func (r *MemoryRepo) ensureTaskCompanyLocked(companyID, taskID int64) error { _, err := r.taskForCompanyLocked(companyID, taskID); return err }
func (r *MemoryRepo) taskForCompanyLocked(companyID, taskID int64) (*project.Task, error) { value, ok := r.tasks[taskID]; if !ok || value.CompanyID != companyID { return nil, notFound("task", taskID) }; return value, nil }
func (r *MemoryRepo) validParentLocked(taskID, parentID, projectID int64) bool { parent, ok := r.tasks[parentID]; return ok && parent.ID != taskID && parent.ProjectID == projectID }
func (r *MemoryRepo) milestoneForCompanyLocked(companyID, id int64) (*project.Milestone, error) { value, ok := r.milestones[id]; if !ok { return nil, notFound("milestone", id) }; p := r.projects[value.ProjectID]; if p == nil || p.CompanyID != companyID { return nil, notFound("milestone", id) }; return value, nil }
func optionalCompanyAllowedPtr(scope *int64, value *int64) bool { return scope == nil || value == nil || *scope == *value }
func contains(values []int64, target int64) bool { for _, value := range values { if value == target { return true } }; return false }
func remove(values []int64, target int64) []int64 { result := values[:0]; for _, value := range values { if value != target { result = append(result, value) } }; return result }
