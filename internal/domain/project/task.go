package project

import (
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type TaskState string

const (
	TaskInProgress       TaskState = "in_progress"
	TaskChangesRequested TaskState = "changes_requested"
	TaskApproved         TaskState = "approved"
	TaskWaiting          TaskState = "waiting"
	TaskDone             TaskState = "done"
	TaskCancelled        TaskState = "cancelled"
)

type TaskPriority string

const (
	PriorityLow    TaskPriority = "0"
	PriorityNormal TaskPriority = "1"
	PriorityHigh   TaskPriority = "2"
	PriorityUrgent TaskPriority = "3"
)

type Task struct {
	ID             int64        `json:"id"`
	Name           string       `json:"name"`
	ProjectID      int64        `json:"project_id"`
	StageID        int64        `json:"stage_id"`
	AssigneeIDs    []int64      `json:"assignee_ids,omitempty"`
	ParentID       *int64       `json:"parent_id,omitempty"`
	DependOnIDs    []int64      `json:"depend_on_ids,omitempty"`
	Priority       TaskPriority `json:"priority"`
	DateDeadline   *time.Time   `json:"date_deadline,omitempty"`
	DateAssign     *time.Time   `json:"date_assign,omitempty"`
	State          TaskState    `json:"state"`
	Description    string       `json:"description,omitempty"`
	MilestoneID    *int64       `json:"milestone_id,omitempty"`
	TagIDs         []int64      `json:"tag_ids,omitempty"`
	Sequence       int          `json:"sequence"`
	AllocatedHours float64      `json:"allocated_hours"`
	CompanyID      int64        `json:"company_id"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

func (t *Task) Validate() error {
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return platformerrors.Validation("task name is required", nil)
	}
	if len(t.Name) > 255 {
		return platformerrors.Validation("task name cannot exceed 255 characters", nil)
	}
	if t.ProjectID <= 0 {
		return platformerrors.Validation("project_id is required", nil)
	}
	if t.StageID <= 0 {
		return platformerrors.Validation("stage_id is required", nil)
	}
	if t.CompanyID <= 0 {
		return platformerrors.Validation("company_id is required", nil)
	}
	if t.Priority == "" {
		t.Priority = PriorityNormal
	}
	if !validPriority(t.Priority) {
		return platformerrors.Validation("invalid task priority", nil)
	}
	if t.State == "" {
		t.State = TaskInProgress
	}
	if !validState(t.State) {
		return platformerrors.Validation("invalid task state", nil)
	}
	if t.AllocatedHours < 0 {
		return platformerrors.Validation("allocated_hours cannot be negative", nil)
	}
	return nil
}

func validPriority(priority TaskPriority) bool {
	return priority == PriorityLow || priority == PriorityNormal || priority == PriorityHigh || priority == PriorityUrgent
}

func validState(state TaskState) bool {
	return state == TaskInProgress || state == TaskChangesRequested || state == TaskApproved || state == TaskWaiting || state == TaskDone || state == TaskCancelled
}

// DependencyCreatesCycle reports whether adding dependencyID -> taskID closes a cycle.
func DependencyCreatesCycle(taskID, dependencyID int64, dependencies map[int64][]int64) bool {
	if taskID <= 0 || dependencyID <= 0 || taskID == dependencyID {
		return true
	}
	visited := make(map[int64]bool)
	var visit func(int64) bool
	visit = func(current int64) bool {
		if current == taskID {
			return true
		}
		if visited[current] {
			return false
		}
		visited[current] = true
		for _, parent := range dependencies[current] {
			if visit(parent) {
				return true
			}
		}
		return false
	}
	return visit(dependencyID)
}

func (t *Task) IsClosed() bool { return t.State == TaskDone || t.State == TaskCancelled }
