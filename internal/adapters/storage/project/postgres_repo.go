package projectstorage

import (
	"context"
	"errors"
	"fmt"

	"cashflow_backend/internal/domain/project"
	platformerrors "cashflow_backend/internal/platform/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepo persists the project-management core in PostgreSQL.
type PostgresRepo struct{ pool *pgxpool.Pool }

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo { return &PostgresRepo{pool: pool} }

func (r *PostgresRepo) CreateProject(ctx context.Context, value *project.Project) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return r.pool.QueryRow(ctx, `INSERT INTO project_projects (name, description, partner_id, manager_id, stage_id, date_start, date_end, active, allow_milestones, allow_subtasks, allow_dependencies, analytic_account_id, company_id) VALUES ($1,$2,$3,$4,$5,$6,$7,true,$8,$9,$10,$11,$12) RETURNING id, created_at, updated_at`, value.Name, value.Description, value.PartnerID, value.ManagerID, value.StageID, value.DateStart, value.DateEnd, value.AllowMilestones, value.AllowSubtasks, value.AllowDependencies, value.AnalyticAccountID, value.CompanyID).Scan(&value.ID, &value.CreatedAt, &value.UpdatedAt)
}

func (r *PostgresRepo) GetProjectByID(ctx context.Context, companyID, id int64) (*project.Project, error) {
	value := &project.Project{}
	err := r.pool.QueryRow(ctx, `SELECT id,name,description,partner_id,manager_id,stage_id,date_start,date_end,active,allow_milestones,allow_subtasks,allow_dependencies,analytic_account_id,company_id,created_at,updated_at FROM project_projects WHERE id=$1 AND company_id=$2`, id, companyID).Scan(&value.ID, &value.Name, &value.Description, &value.PartnerID, &value.ManagerID, &value.StageID, &value.DateStart, &value.DateEnd, &value.Active, &value.AllowMilestones, &value.AllowSubtasks, &value.AllowDependencies, &value.AnalyticAccountID, &value.CompanyID, &value.CreatedAt, &value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("project", id)
	}
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch project", err)
	}
	return value, nil
}

func (r *PostgresRepo) UpdateProject(ctx context.Context, value *project.Project) error {
	if err := value.Validate(); err != nil {
		return err
	}
	err := r.pool.QueryRow(ctx, `UPDATE project_projects SET name=$1,description=$2,partner_id=$3,manager_id=$4,stage_id=$5,date_start=$6,date_end=$7,active=$8,allow_milestones=$9,allow_subtasks=$10,allow_dependencies=$11,analytic_account_id=$12,updated_at=NOW() WHERE id=$13 AND company_id=$14 RETURNING updated_at`, value.Name, value.Description, value.PartnerID, value.ManagerID, value.StageID, value.DateStart, value.DateEnd, value.Active, value.AllowMilestones, value.AllowSubtasks, value.AllowDependencies, value.AnalyticAccountID, value.ID, value.CompanyID).Scan(&value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound("project", value.ID)
	}
	if err != nil {
		return platformerrors.Internal("failed to update project", err)
	}
	return nil
}
func (r *PostgresRepo) DeleteProject(ctx context.Context, companyID, id int64) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM project_projects WHERE id=$1 AND company_id=$2`, id, companyID)
	if err != nil {
		return platformerrors.Internal("failed to delete project", err)
	}
	if result.RowsAffected() == 0 {
		return notFound("project", id)
	}
	return nil
}
func (r *PostgresRepo) ListProjects(ctx context.Context, companyID int64) ([]project.Project, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,name,description,partner_id,manager_id,stage_id,date_start,date_end,active,allow_milestones,allow_subtasks,allow_dependencies,analytic_account_id,company_id,created_at,updated_at FROM project_projects WHERE company_id=$1 ORDER BY id`, companyID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list projects", err)
	}
	defer rows.Close()
	result := make([]project.Project, 0)
	for rows.Next() {
		var value project.Project
		if err := rows.Scan(&value.ID, &value.Name, &value.Description, &value.PartnerID, &value.ManagerID, &value.StageID, &value.DateStart, &value.DateEnd, &value.Active, &value.AllowMilestones, &value.AllowSubtasks, &value.AllowDependencies, &value.AnalyticAccountID, &value.CompanyID, &value.CreatedAt, &value.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan project", err)
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *PostgresRepo) CreateTask(ctx context.Context, value *project.Task) error {
	if err := value.Validate(); err != nil {
		return err
	}
	err := r.pool.QueryRow(ctx, `INSERT INTO project_tasks (name,project_id,stage_id,parent_id,priority,date_deadline,state,description,milestone_id,sequence,allocated_hours,company_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id,created_at,updated_at`, value.Name, value.ProjectID, value.StageID, value.ParentID, string(value.Priority), value.DateDeadline, string(value.State), value.Description, value.MilestoneID, value.Sequence, value.AllocatedHours, value.CompanyID).Scan(&value.ID, &value.CreatedAt, &value.UpdatedAt)
	if err != nil {
		return platformerrors.Internal("failed to create task", err)
	}
	return r.replaceTaskRelations(ctx, value)
}
func (r *PostgresRepo) GetTaskByID(ctx context.Context, companyID, id int64) (*project.Task, error) {
	value := &project.Task{}
	var priority, state string
	err := r.pool.QueryRow(ctx, `SELECT id,name,project_id,stage_id,parent_id,priority,date_deadline,date_assign,state,description,milestone_id,sequence,allocated_hours,company_id,created_at,updated_at FROM project_tasks WHERE id=$1 AND company_id=$2`, id, companyID).Scan(&value.ID, &value.Name, &value.ProjectID, &value.StageID, &value.ParentID, &priority, &value.DateDeadline, &value.DateAssign, &state, &value.Description, &value.MilestoneID, &value.Sequence, &value.AllocatedHours, &value.CompanyID, &value.CreatedAt, &value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("task", id)
	}
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch task", err)
	}
	value.Priority = project.TaskPriority(priority)
	value.State = project.TaskState(state)
	if err := r.loadTaskRelations(ctx, value); err != nil {
		return nil, err
	}
	return value, nil
}
func (r *PostgresRepo) UpdateTask(ctx context.Context, value *project.Task) error {
	if err := value.Validate(); err != nil {
		return err
	}
	err := r.pool.QueryRow(ctx, `UPDATE project_tasks SET name=$1,stage_id=$2,parent_id=$3,priority=$4,date_deadline=$5,date_assign=$6,state=$7,description=$8,milestone_id=$9,sequence=$10,allocated_hours=$11,updated_at=NOW() WHERE id=$12 AND company_id=$13 RETURNING updated_at`, value.Name, value.StageID, value.ParentID, string(value.Priority), value.DateDeadline, value.DateAssign, string(value.State), value.Description, value.MilestoneID, value.Sequence, value.AllocatedHours, value.ID, value.CompanyID).Scan(&value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound("task", value.ID)
	}
	if err != nil {
		return platformerrors.Internal("failed to update task", err)
	}
	return r.replaceTaskRelations(ctx, value)
}
func (r *PostgresRepo) DeleteTask(ctx context.Context, companyID, id int64) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM project_tasks WHERE id=$1 AND company_id=$2`, id, companyID)
	if err != nil {
		return platformerrors.Internal("failed to delete task", err)
	}
	if result.RowsAffected() == 0 {
		return notFound("task", id)
	}
	return nil
}
func (r *PostgresRepo) ListTasks(ctx context.Context, companyID int64, filter project.TaskFilter) ([]project.Task, error) {
	query := `SELECT id FROM project_tasks WHERE company_id=$1`
	args := []any{companyID}
	next := 2
	if filter.ProjectID != nil {
		query += fmt.Sprintf(" AND project_id=$%d", next)
		args = append(args, *filter.ProjectID)
		next++
	}
	if filter.StageID != nil {
		query += fmt.Sprintf(" AND stage_id=$%d", next)
		args = append(args, *filter.StageID)
		next++
	}
	if filter.State != nil {
		query += fmt.Sprintf(" AND state=$%d", next)
		args = append(args, string(*filter.State))
		next++
	}
	query += " ORDER BY id"
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, platformerrors.Internal("failed to list tasks", err)
	}
	defer rows.Close()
	result := make([]project.Task, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, platformerrors.Internal("failed to scan task", err)
		}
		value, err := r.GetTaskByID(ctx, companyID, id)
		if err != nil {
			return nil, err
		}
		if !filter.IncludeDone && value.IsClosed() {
			continue
		}
		if filter.AssigneeID != nil && !contains(value.AssigneeIDs, *filter.AssigneeID) {
			continue
		}
		if filter.ParentID != nil && (value.ParentID == nil || *value.ParentID != *filter.ParentID) {
			continue
		}
		result = append(result, *value)
	}
	return result, rows.Err()
}
func (r *PostgresRepo) ListKanban(ctx context.Context, companyID, projectID int64) ([]project.TaskStageBucket, error) {
	if _, err := r.GetProjectByID(ctx, companyID, projectID); err != nil {
		return nil, err
	}
	stages, err := r.ListTaskStages(ctx, &companyID, &projectID)
	if err != nil {
		return nil, err
	}
	tasks, err := r.ListTasks(ctx, companyID, project.TaskFilter{ProjectID: &projectID, IncludeDone: true})
	if err != nil {
		return nil, err
	}
	buckets := make([]project.TaskStageBucket, len(stages))
	byID := make(map[int64]int, len(stages))
	for i, stage := range stages {
		buckets[i].Stage = project.StageSummary{ID: stage.ID, Name: string(stage.Name), Sequence: stage.Sequence, Fold: stage.Fold, Color: stage.Color}
		byID[stage.ID] = i
	}
	for _, task := range tasks {
		if index, ok := byID[task.StageID]; ok {
			buckets[index].Tasks = append(buckets[index].Tasks, task)
		}
	}
	return buckets, nil
}

func (r *PostgresRepo) SetTaskAssignees(ctx context.Context, companyID, taskID int64, ids []int64) error {
	if _, err := r.GetTaskByID(ctx, companyID, taskID); err != nil {
		return err
	}
	if _, err := r.pool.Exec(ctx, `DELETE FROM project_task_assignees WHERE task_id=$1`, taskID); err != nil {
		return platformerrors.Internal("failed to clear task assignees", err)
	}
	for _, id := range ids {
		if _, err := r.pool.Exec(ctx, `INSERT INTO project_task_assignees(task_id,user_id) VALUES($1,$2)`, taskID, id); err != nil {
			return platformerrors.Internal("failed to assign task", err)
		}
	}
	return nil
}
func (r *PostgresRepo) SetTaskTags(ctx context.Context, companyID, taskID int64, ids []int64) error {
	if _, err := r.GetTaskByID(ctx, companyID, taskID); err != nil {
		return err
	}
	if _, err := r.pool.Exec(ctx, `DELETE FROM project_task_tag_rel WHERE task_id=$1`, taskID); err != nil {
		return platformerrors.Internal("failed to clear task tags", err)
	}
	for _, id := range ids {
		if _, err := r.pool.Exec(ctx, `INSERT INTO project_task_tag_rel(task_id,tag_id) VALUES($1,$2)`, taskID, id); err != nil {
			return platformerrors.Internal("failed to tag task", err)
		}
	}
	return nil
}
func (r *PostgresRepo) AddTaskDependency(ctx context.Context, companyID, taskID, dependsOn int64) error {
	task, err := r.GetTaskByID(ctx, companyID, taskID)
	if err != nil {
		return err
	}
	dependency, err := r.GetTaskByID(ctx, companyID, dependsOn)
	if err != nil {
		return err
	}
	if task.ProjectID != dependency.ProjectID {
		return platformerrors.Conflict("task dependencies must belong to the same project")
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO project_task_dependencies(task_id,depends_on_task_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, taskID, dependsOn); err != nil {
		return platformerrors.Internal("failed to add task dependency", err)
	}
	return nil
}
func (r *PostgresRepo) RemoveTaskDependency(ctx context.Context, companyID, taskID, dependsOn int64) error {
	if _, err := r.GetTaskByID(ctx, companyID, taskID); err != nil {
		return err
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM project_task_dependencies WHERE task_id=$1 AND depends_on_task_id=$2`, taskID, dependsOn)
	if err != nil {
		return platformerrors.Internal("failed to remove task dependency", err)
	}
	return nil
}

func (r *PostgresRepo) loadTaskRelations(ctx context.Context, value *project.Task) error {
	rows, err := r.pool.Query(ctx, `SELECT user_id FROM project_task_assignees WHERE task_id=$1`, value.ID)
	if err != nil {
		return platformerrors.Internal("failed to load task assignees", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		value.AssigneeIDs = append(value.AssigneeIDs, id)
	}
	rows, err = r.pool.Query(ctx, `SELECT depends_on_task_id FROM project_task_dependencies WHERE task_id=$1`, value.ID)
	if err != nil {
		return platformerrors.Internal("failed to load task dependencies", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		value.DependOnIDs = append(value.DependOnIDs, id)
	}
	return nil
}
func (r *PostgresRepo) replaceTaskRelations(ctx context.Context, value *project.Task) error {
	if err := r.SetTaskAssignees(ctx, value.CompanyID, value.ID, value.AssigneeIDs); err != nil {
		return err
	}
	return r.SetTaskTags(ctx, value.CompanyID, value.ID, value.TagIDs)
}

func (r *PostgresRepo) CreateProjectStage(ctx context.Context, value *project.ProjectStage) error {
	if err := value.Validate(); err != nil {
		return err
	}
	return r.pool.QueryRow(ctx, `INSERT INTO project_project_stages (name,sequence,fold,color,company_id,active) VALUES ($1,$2,$3,$4,$5,true) RETURNING id,created_at,updated_at`, value.Name, value.Sequence, value.Fold, value.Color, value.CompanyID).Scan(&value.ID, &value.CreatedAt, &value.UpdatedAt)
}
func (r *PostgresRepo) GetProjectStageByID(ctx context.Context, companyID *int64, id int64) (*project.ProjectStage, error) {
	value := &project.ProjectStage{}
	err := r.pool.QueryRow(ctx, `SELECT id,name,sequence,fold,color,company_id,active,created_at,updated_at FROM project_project_stages WHERE id=$1 AND ($2::bigint IS NULL OR company_id IS NULL OR company_id=$2)`, id, companyID).Scan(&value.ID, &value.Name, &value.Sequence, &value.Fold, &value.Color, &value.CompanyID, &value.Active, &value.CreatedAt, &value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("project stage", id)
	}
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch project stage", err)
	}
	return value, nil
}
func (r *PostgresRepo) UpdateProjectStage(ctx context.Context, value *project.ProjectStage) error {
	if err := value.Validate(); err != nil {
		return err
	}
	err := r.pool.QueryRow(ctx, `UPDATE project_project_stages SET name=$1,sequence=$2,fold=$3,color=$4,active=$5,updated_at=NOW() WHERE id=$6 RETURNING updated_at`, value.Name, value.Sequence, value.Fold, value.Color, value.Active, value.ID).Scan(&value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound("project stage", value.ID)
	}
	if err != nil {
		return platformerrors.Internal("failed to update project stage", err)
	}
	return nil
}
func (r *PostgresRepo) DeleteProjectStage(ctx context.Context, companyID *int64, id int64) error {
	if _, err := r.GetProjectStageByID(ctx, companyID, id); err != nil {
		return err
	}
	result, err := r.pool.Exec(ctx, `DELETE FROM project_project_stages WHERE id=$1`, id)
	if err != nil {
		return platformerrors.Internal("failed to delete project stage", err)
	}
	if result.RowsAffected() == 0 {
		return notFound("project stage", id)
	}
	return nil
}
func (r *PostgresRepo) ListProjectStages(ctx context.Context, companyID *int64) ([]project.ProjectStage, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,name,sequence,fold,color,company_id,active,created_at,updated_at FROM project_project_stages WHERE ($1::bigint IS NULL OR company_id IS NULL OR company_id=$1) ORDER BY sequence,id`, companyID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list project stages", err)
	}
	defer rows.Close()
	result := make([]project.ProjectStage, 0)
	for rows.Next() {
		var value project.ProjectStage
		if err := rows.Scan(&value.ID, &value.Name, &value.Sequence, &value.Fold, &value.Color, &value.CompanyID, &value.Active, &value.CreatedAt, &value.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan project stage", err)
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *PostgresRepo) CreateTaskStage(ctx context.Context, value *project.TaskStage) error {
	if err := value.Validate(); err != nil {
		return err
	}
	err := r.pool.QueryRow(ctx, `INSERT INTO project_task_types(name,sequence,fold,color,active,company_id) VALUES($1,$2,$3,$4,true,$5) RETURNING id,created_at,updated_at`, value.Name, value.Sequence, value.Fold, value.Color, value.CompanyID).Scan(&value.ID, &value.CreatedAt, &value.UpdatedAt)
	if err != nil {
		return platformerrors.Internal("failed to create task stage", err)
	}
	return r.replaceTaskStageProjects(ctx, value.ID, value.ProjectIDs)
}
func (r *PostgresRepo) GetTaskStageByID(ctx context.Context, companyID *int64, id int64) (*project.TaskStage, error) {
	value := &project.TaskStage{}
	err := r.pool.QueryRow(ctx, `SELECT id,name,sequence,fold,color,active,company_id,created_at,updated_at FROM project_task_types WHERE id=$1 AND ($2::bigint IS NULL OR company_id IS NULL OR company_id=$2)`, id, companyID).Scan(&value.ID, &value.Name, &value.Sequence, &value.Fold, &value.Color, &value.Active, &value.CompanyID, &value.CreatedAt, &value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("task stage", id)
	}
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch task stage", err)
	}
	if err := r.loadTaskStageProjects(ctx, value); err != nil {
		return nil, err
	}
	return value, nil
}
func (r *PostgresRepo) UpdateTaskStage(ctx context.Context, value *project.TaskStage) error {
	if err := value.Validate(); err != nil {
		return err
	}
	err := r.pool.QueryRow(ctx, `UPDATE project_task_types SET name=$1,sequence=$2,fold=$3,color=$4,active=$5,updated_at=NOW() WHERE id=$6 RETURNING updated_at`, value.Name, value.Sequence, value.Fold, value.Color, value.Active, value.ID).Scan(&value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound("task stage", value.ID)
	}
	if err != nil {
		return platformerrors.Internal("failed to update task stage", err)
	}
	return r.replaceTaskStageProjects(ctx, value.ID, value.ProjectIDs)
}
func (r *PostgresRepo) DeleteTaskStage(ctx context.Context, companyID *int64, id int64) error {
	if _, err := r.GetTaskStageByID(ctx, companyID, id); err != nil {
		return err
	}
	result, err := r.pool.Exec(ctx, `DELETE FROM project_task_types WHERE id=$1`, id)
	if err != nil {
		return platformerrors.Internal("failed to delete task stage", err)
	}
	if result.RowsAffected() == 0 {
		return notFound("task stage", id)
	}
	return nil
}
func (r *PostgresRepo) ListTaskStages(ctx context.Context, companyID *int64, projectID *int64) ([]project.TaskStage, error) {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT t.id,t.name,t.sequence,t.fold,t.color,t.active,t.company_id,t.created_at,t.updated_at FROM project_task_types t LEFT JOIN project_task_type_projects p ON p.task_type_id=t.id WHERE ($1::bigint IS NULL OR t.company_id IS NULL OR t.company_id=$1) AND ($2::bigint IS NULL OR p.project_id=$2) ORDER BY t.sequence,t.id`, companyID, projectID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list task stages", err)
	}
	defer rows.Close()
	result := make([]project.TaskStage, 0)
	for rows.Next() {
		var value project.TaskStage
		if err := rows.Scan(&value.ID, &value.Name, &value.Sequence, &value.Fold, &value.Color, &value.Active, &value.CompanyID, &value.CreatedAt, &value.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan task stage", err)
		}
		if err := r.loadTaskStageProjects(ctx, &value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *PostgresRepo) CreateMilestone(ctx context.Context, value *project.Milestone) error {
	if err := value.Validate(); err != nil {
		return err
	}
	err := r.pool.QueryRow(ctx, `INSERT INTO project_milestones(name,project_id,date_deadline,is_reached,reached_date,sequence) SELECT $1,$2,$3,$4,$5,$6 WHERE EXISTS (SELECT 1 FROM project_projects WHERE id=$2) RETURNING id,created_at,updated_at`, value.Name, value.ProjectID, value.DateDeadline, value.IsReached, value.ReachedDate, value.Sequence).Scan(&value.ID, &value.CreatedAt, &value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound("project", value.ProjectID)
	}
	if err != nil {
		return platformerrors.Internal("failed to create milestone", err)
	}
	return nil
}
func (r *PostgresRepo) GetMilestoneByID(ctx context.Context, companyID, id int64) (*project.Milestone, error) {
	value := &project.Milestone{}
	err := r.pool.QueryRow(ctx, `SELECT m.id,m.name,m.project_id,m.date_deadline,m.is_reached,m.reached_date,m.sequence,m.created_at,m.updated_at FROM project_milestones m JOIN project_projects p ON p.id=m.project_id WHERE m.id=$1 AND p.company_id=$2`, id, companyID).Scan(&value.ID, &value.Name, &value.ProjectID, &value.DateDeadline, &value.IsReached, &value.ReachedDate, &value.Sequence, &value.CreatedAt, &value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("milestone", id)
	}
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch milestone", err)
	}
	return value, nil
}
func (r *PostgresRepo) UpdateMilestone(ctx context.Context, value *project.Milestone) error {
	if err := value.Validate(); err != nil {
		return err
	}
	err := r.pool.QueryRow(ctx, `UPDATE project_milestones SET name=$1,date_deadline=$2,is_reached=$3,reached_date=$4,sequence=$5,updated_at=NOW() WHERE id=$6 RETURNING updated_at`, value.Name, value.DateDeadline, value.IsReached, value.ReachedDate, value.Sequence, value.ID).Scan(&value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound("milestone", value.ID)
	}
	if err != nil {
		return platformerrors.Internal("failed to update milestone", err)
	}
	return nil
}
func (r *PostgresRepo) DeleteMilestone(ctx context.Context, companyID, id int64) error {
	if _, err := r.GetMilestoneByID(ctx, companyID, id); err != nil {
		return err
	}
	result, err := r.pool.Exec(ctx, `DELETE FROM project_milestones WHERE id=$1`, id)
	if err != nil {
		return platformerrors.Internal("failed to delete milestone", err)
	}
	if result.RowsAffected() == 0 {
		return notFound("milestone", id)
	}
	return nil
}
func (r *PostgresRepo) ListMilestones(ctx context.Context, companyID, projectID int64) ([]project.Milestone, error) {
	rows, err := r.pool.Query(ctx, `SELECT m.id,m.name,m.project_id,m.date_deadline,m.is_reached,m.reached_date,m.sequence,m.created_at,m.updated_at FROM project_milestones m JOIN project_projects p ON p.id=m.project_id WHERE p.company_id=$1 AND m.project_id=$2 ORDER BY m.sequence,m.id`, companyID, projectID)
	if err != nil {
		return nil, platformerrors.Internal("failed to list milestones", err)
	}
	defer rows.Close()
	result := make([]project.Milestone, 0)
	for rows.Next() {
		var value project.Milestone
		if err := rows.Scan(&value.ID, &value.Name, &value.ProjectID, &value.DateDeadline, &value.IsReached, &value.ReachedDate, &value.Sequence, &value.CreatedAt, &value.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan milestone", err)
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *PostgresRepo) CreateTaskTag(ctx context.Context, value *project.TaskTag) error {
	if err := value.Validate(); err != nil {
		return err
	}
	err := r.pool.QueryRow(ctx, `INSERT INTO project_task_tags(name,color,active) VALUES($1,$2,true) RETURNING id,created_at,updated_at`, value.Name, value.Color).Scan(&value.ID, &value.CreatedAt, &value.UpdatedAt)
	if err != nil {
		return platformerrors.Internal("failed to create task tag", err)
	}
	return nil
}
func (r *PostgresRepo) GetTaskTagByID(ctx context.Context, id int64) (*project.TaskTag, error) {
	value := &project.TaskTag{}
	err := r.pool.QueryRow(ctx, `SELECT id,name,color,active,created_at,updated_at FROM project_task_tags WHERE id=$1`, id).Scan(&value.ID, &value.Name, &value.Color, &value.Active, &value.CreatedAt, &value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("task tag", id)
	}
	if err != nil {
		return nil, platformerrors.Internal("failed to fetch task tag", err)
	}
	return value, nil
}
func (r *PostgresRepo) UpdateTaskTag(ctx context.Context, value *project.TaskTag) error {
	if err := value.Validate(); err != nil {
		return err
	}
	err := r.pool.QueryRow(ctx, `UPDATE project_task_tags SET name=$1,color=$2,active=$3,updated_at=NOW() WHERE id=$4 RETURNING updated_at`, value.Name, value.Color, value.Active, value.ID).Scan(&value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound("task tag", value.ID)
	}
	if err != nil {
		return platformerrors.Internal("failed to update task tag", err)
	}
	return nil
}
func (r *PostgresRepo) DeleteTaskTag(ctx context.Context, id int64) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM project_task_tags WHERE id=$1`, id)
	if err != nil {
		return platformerrors.Internal("failed to delete task tag", err)
	}
	if result.RowsAffected() == 0 {
		return notFound("task tag", id)
	}
	return nil
}
func (r *PostgresRepo) ListTaskTags(ctx context.Context) ([]project.TaskTag, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,name,color,active,created_at,updated_at FROM project_task_tags ORDER BY id`)
	if err != nil {
		return nil, platformerrors.Internal("failed to list task tags", err)
	}
	defer rows.Close()
	result := make([]project.TaskTag, 0)
	for rows.Next() {
		var value project.TaskTag
		if err := rows.Scan(&value.ID, &value.Name, &value.Color, &value.Active, &value.CreatedAt, &value.UpdatedAt); err != nil {
			return nil, platformerrors.Internal("failed to scan task tag", err)
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *PostgresRepo) replaceTaskStageProjects(ctx context.Context, stageID int64, projectIDs []int64) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM project_task_type_projects WHERE task_type_id=$1`, stageID); err != nil {
		return platformerrors.Internal("failed to clear task stage projects", err)
	}
	for _, projectID := range projectIDs {
		if _, err := r.pool.Exec(ctx, `INSERT INTO project_task_type_projects(task_type_id,project_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, stageID, projectID); err != nil {
			return platformerrors.Internal("failed to assign task stage project", err)
		}
	}
	return nil
}
func (r *PostgresRepo) loadTaskStageProjects(ctx context.Context, value *project.TaskStage) error {
	rows, err := r.pool.Query(ctx, `SELECT project_id FROM project_task_type_projects WHERE task_type_id=$1 ORDER BY project_id`, value.ID)
	if err != nil {
		return platformerrors.Internal("failed to load task stage projects", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		value.ProjectIDs = append(value.ProjectIDs, id)
	}
	return rows.Err()
}
