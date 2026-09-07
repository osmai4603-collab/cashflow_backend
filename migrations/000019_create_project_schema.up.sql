-- Phase 17: Project management schema.
-- Project stages and task stages intentionally remain separate, matching Odoo 19.

CREATE TABLE IF NOT EXISTS project_project_stages (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    sequence INT NOT NULL DEFAULT 10,
    fold BOOLEAN NOT NULL DEFAULT false,
    color INT NOT NULL DEFAULT 0,
    company_id BIGINT REFERENCES res_companies(id) ON DELETE CASCADE,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    CONSTRAINT uq_project_project_stages_name_company UNIQUE (name, company_id)
);
CREATE INDEX IF NOT EXISTS idx_project_project_stages_company ON project_project_stages(company_id);
CREATE INDEX IF NOT EXISTS idx_project_project_stages_sequence ON project_project_stages(sequence, id);

CREATE TABLE IF NOT EXISTS project_projects (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    manager_id BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    stage_id BIGINT REFERENCES project_project_stages(id) ON DELETE RESTRICT,
    date_start DATE,
    date_end DATE,
    active BOOLEAN NOT NULL DEFAULT true,
    allow_milestones BOOLEAN NOT NULL DEFAULT false,
    allow_subtasks BOOLEAN NOT NULL DEFAULT true,
    allow_dependencies BOOLEAN NOT NULL DEFAULT false,
    analytic_account_id BIGINT REFERENCES account_analytic_account(id) ON DELETE SET NULL,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    CONSTRAINT chk_project_projects_dates CHECK (date_end IS NULL OR date_start IS NULL OR date_end >= date_start)
);
CREATE INDEX IF NOT EXISTS idx_project_projects_company ON project_projects(company_id);
CREATE INDEX IF NOT EXISTS idx_project_projects_manager ON project_projects(manager_id);
CREATE INDEX IF NOT EXISTS idx_project_projects_stage ON project_projects(stage_id);
CREATE INDEX IF NOT EXISTS idx_project_projects_active ON project_projects(active);

CREATE TABLE IF NOT EXISTS project_task_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    sequence INT NOT NULL DEFAULT 10,
    fold BOOLEAN NOT NULL DEFAULT false,
    color INT NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT true,
    company_id BIGINT REFERENCES res_companies(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    CONSTRAINT uq_project_task_types_name_company UNIQUE (name, company_id)
);
CREATE INDEX IF NOT EXISTS idx_project_task_types_company ON project_task_types(company_id);
CREATE INDEX IF NOT EXISTS idx_project_task_types_sequence ON project_task_types(sequence, id);

CREATE TABLE IF NOT EXISTS project_task_type_projects (
    task_type_id BIGINT NOT NULL REFERENCES project_task_types(id) ON DELETE CASCADE,
    project_id BIGINT NOT NULL REFERENCES project_projects(id) ON DELETE CASCADE,
    PRIMARY KEY (task_type_id, project_id)
);
CREATE INDEX IF NOT EXISTS idx_project_task_type_projects_project ON project_task_type_projects(project_id);

CREATE TABLE IF NOT EXISTS project_tasks (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    project_id BIGINT NOT NULL REFERENCES project_projects(id) ON DELETE CASCADE,
    stage_id BIGINT NOT NULL REFERENCES project_task_types(id) ON DELETE RESTRICT,
    parent_id BIGINT REFERENCES project_tasks(id) ON DELETE CASCADE,
    priority VARCHAR(1) NOT NULL DEFAULT '1',
    date_deadline TIMESTAMPTZ,
    date_assign TIMESTAMPTZ,
    state VARCHAR(32) NOT NULL DEFAULT 'in_progress',
    description TEXT,
    milestone_id BIGINT,
    sequence INT NOT NULL DEFAULT 10,
    allocated_hours DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (allocated_hours >= 0),
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    CONSTRAINT chk_project_tasks_priority CHECK (priority IN ('0', '1', '2', '3')),
    CONSTRAINT chk_project_tasks_state CHECK (state IN ('in_progress', 'changes_requested', 'approved', 'waiting', 'done', 'cancelled')),
    CONSTRAINT chk_project_tasks_not_self_parent CHECK (parent_id IS NULL OR parent_id <> id)
);
CREATE INDEX IF NOT EXISTS idx_project_tasks_project ON project_tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_project_tasks_stage ON project_tasks(stage_id);
CREATE INDEX IF NOT EXISTS idx_project_tasks_parent ON project_tasks(parent_id);
CREATE INDEX IF NOT EXISTS idx_project_tasks_company ON project_tasks(company_id);
CREATE INDEX IF NOT EXISTS idx_project_tasks_state ON project_tasks(state);
CREATE INDEX IF NOT EXISTS idx_project_tasks_deadline ON project_tasks(date_deadline);

CREATE TABLE IF NOT EXISTS project_task_assignees (
    task_id BIGINT NOT NULL REFERENCES project_tasks(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES res_users(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_project_task_assignees_user ON project_task_assignees(user_id);

CREATE TABLE IF NOT EXISTS project_task_dependencies (
    task_id BIGINT NOT NULL REFERENCES project_tasks(id) ON DELETE CASCADE,
    depends_on_task_id BIGINT NOT NULL REFERENCES project_tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, depends_on_task_id),
    CONSTRAINT chk_project_task_dependencies_not_self CHECK (task_id <> depends_on_task_id)
);
CREATE INDEX IF NOT EXISTS idx_project_task_dependencies_dependency ON project_task_dependencies(depends_on_task_id);

CREATE TABLE IF NOT EXISTS project_task_tags (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    color INT NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    CONSTRAINT uq_project_task_tags_name UNIQUE (name)
);
CREATE TABLE IF NOT EXISTS project_task_tag_rel (
    task_id BIGINT NOT NULL REFERENCES project_tasks(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES project_task_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, tag_id)
);
CREATE INDEX IF NOT EXISTS idx_project_task_tag_rel_tag ON project_task_tag_rel(tag_id);

CREATE TABLE IF NOT EXISTS project_milestones (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    project_id BIGINT NOT NULL REFERENCES project_projects(id) ON DELETE CASCADE,
    date_deadline DATE,
    is_reached BOOLEAN NOT NULL DEFAULT false,
    reached_date DATE,
    sequence INT NOT NULL DEFAULT 10,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_project_milestones_project ON project_milestones(project_id);
CREATE INDEX IF NOT EXISTS idx_project_milestones_deadline ON project_milestones(date_deadline);
ALTER TABLE project_tasks ADD CONSTRAINT fk_project_tasks_milestone FOREIGN KEY (milestone_id) REFERENCES project_milestones(id) ON DELETE SET NULL;

CREATE TRIGGER trg_project_project_stages_updated_at BEFORE UPDATE ON project_project_stages FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_project_projects_updated_at BEFORE UPDATE ON project_projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_project_task_types_updated_at BEFORE UPDATE ON project_task_types FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_project_tasks_updated_at BEFORE UPDATE ON project_tasks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_project_task_tags_updated_at BEFORE UPDATE ON project_task_tags FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_project_milestones_updated_at BEFORE UPDATE ON project_milestones FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'project.project', true, false, false, false), (2::BIGINT, 'project.project', true, true, true, true),
    (1::BIGINT, 'project.task', true, true, true, false), (2::BIGINT, 'project.task', true, true, true, true),
    (1::BIGINT, 'project.project.stage', true, false, false, false), (2::BIGINT, 'project.project.stage', true, true, true, true),
    (1::BIGINT, 'project.task.type', true, false, false, false), (2::BIGINT, 'project.task.type', true, true, true, true),
    (1::BIGINT, 'project.milestone', true, true, true, false), (2::BIGINT, 'project.milestone', true, true, true, true),
    (1::BIGINT, 'project.tags', true, true, true, false), (2::BIGINT, 'project.tags', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;
