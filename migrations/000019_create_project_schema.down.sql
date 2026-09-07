DROP TRIGGER IF EXISTS trg_project_milestones_updated_at ON project_milestones;
DROP TRIGGER IF EXISTS trg_project_task_tags_updated_at ON project_task_tags;
DROP TRIGGER IF EXISTS trg_project_tasks_updated_at ON project_tasks;
DROP TRIGGER IF EXISTS trg_project_task_types_updated_at ON project_task_types;
DROP TRIGGER IF EXISTS trg_project_projects_updated_at ON project_projects;
DROP TRIGGER IF EXISTS trg_project_project_stages_updated_at ON project_project_stages;

DELETE FROM res_group_permissions
WHERE model IN ('project.project', 'project.task', 'project.project.stage', 'project.task.type', 'project.milestone', 'project.tags');

ALTER TABLE project_tasks DROP CONSTRAINT IF EXISTS fk_project_tasks_milestone;
DROP TABLE IF EXISTS project_milestones;
DROP TABLE IF EXISTS project_task_tag_rel;
DROP TABLE IF EXISTS project_task_tags;
DROP TABLE IF EXISTS project_task_dependencies;
DROP TABLE IF EXISTS project_task_assignees;
DROP TABLE IF EXISTS project_tasks;
DROP TABLE IF EXISTS project_task_type_projects;
DROP TABLE IF EXISTS project_task_types;
DROP TABLE IF EXISTS project_projects;
DROP TABLE IF EXISTS project_project_stages;
