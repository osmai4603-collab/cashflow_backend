-- Phase 15: activities, notifications, messages, and email delivery.

CREATE TABLE IF NOT EXISTS mail_activity_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    summary VARCHAR(255),
    res_model VARCHAR(128),
    category VARCHAR(32) NOT NULL DEFAULT 'default',
    delay_count INT NOT NULL DEFAULT 0 CHECK (delay_count >= 0),
    delay_unit VARCHAR(16) NOT NULL DEFAULT 'days',
    icon VARCHAR(128),
    sequence INT NOT NULL DEFAULT 10,
    default_note TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    system_type BOOLEAN NOT NULL DEFAULT false,
    company_id BIGINT REFERENCES res_companies(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    CONSTRAINT chk_mail_activity_types_delay_unit CHECK (delay_unit IN ('days', 'weeks', 'months')),
    CONSTRAINT uq_mail_activity_types_name_company UNIQUE (name, company_id)
);
CREATE INDEX IF NOT EXISTS idx_mail_activity_types_company ON mail_activity_types(company_id);
CREATE INDEX IF NOT EXISTS idx_mail_activity_types_active ON mail_activity_types(active);

CREATE TABLE IF NOT EXISTS mail_activities (
    id BIGSERIAL PRIMARY KEY,
    activity_type_id BIGINT NOT NULL REFERENCES mail_activity_types(id) ON DELETE RESTRICT,
    summary VARCHAR(255) NOT NULL,
    note TEXT,
    date_deadline DATE NOT NULL,
    assigned_user_id BIGINT NOT NULL REFERENCES res_users(id) ON DELETE RESTRICT,
    res_model VARCHAR(128),
    res_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    date_done TIMESTAMPTZ,
    feedback TEXT,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    CONSTRAINT chk_mail_activities_reference_pair CHECK ((res_model IS NULL) = (res_id IS NULL)),
    CONSTRAINT chk_mail_activities_done_data CHECK (active OR date_done IS NOT NULL),
    CONSTRAINT chk_mail_activities_done_date CHECK (date_done IS NULL OR date_done >= created_at)
);
CREATE INDEX IF NOT EXISTS idx_mail_activities_user_deadline ON mail_activities(company_id, assigned_user_id, active, date_deadline, id);
CREATE INDEX IF NOT EXISTS idx_mail_activities_resource ON mail_activities(company_id, res_model, res_id, active, date_deadline, id);
CREATE INDEX IF NOT EXISTS idx_mail_activities_type_active ON mail_activities(activity_type_id, active);
CREATE INDEX IF NOT EXISTS idx_mail_activities_deadline ON mail_activities(company_id, active, date_deadline);

CREATE TABLE IF NOT EXISTS mail_messages (
    id BIGSERIAL PRIMARY KEY,
    subject VARCHAR(255),
    body TEXT NOT NULL,
    message_type VARCHAR(32) NOT NULL DEFAULT 'notification',
    res_model VARCHAR(128),
    res_id BIGINT,
    author_id BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    activity_id BIGINT REFERENCES mail_activities(id) ON DELETE SET NULL,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_mail_messages_reference_pair CHECK ((res_model IS NULL) = (res_id IS NULL))
);
CREATE INDEX IF NOT EXISTS idx_mail_messages_resource ON mail_messages(company_id, res_model, res_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_mail_messages_activity ON mail_messages(activity_id);

CREATE TABLE IF NOT EXISTS mail_notifications (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT REFERENCES mail_messages(id) ON DELETE CASCADE,
    activity_id BIGINT REFERENCES mail_activities(id) ON DELETE CASCADE,
    recipient_user_id BIGINT NOT NULL REFERENCES res_users(id) ON DELETE CASCADE,
    notification_type VARCHAR(16) NOT NULL DEFAULT 'inbox',
    status VARCHAR(16) NOT NULL DEFAULT 'unread',
    read_at TIMESTAMPTZ,
    email VARCHAR(320),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE RESTRICT,
    CONSTRAINT chk_mail_notifications_type CHECK (notification_type IN ('inbox', 'email')),
    CONSTRAINT chk_mail_notifications_status CHECK (status IN ('unread', 'read', 'queued', 'sent', 'failed')),
    CONSTRAINT chk_mail_notifications_read_at CHECK ((status = 'read') = (read_at IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS idx_mail_notifications_recipient ON mail_notifications(company_id, recipient_user_id, status, created_at, id);
CREATE INDEX IF NOT EXISTS idx_mail_notifications_activity ON mail_notifications(activity_id);

CREATE TABLE IF NOT EXISTS mail_email_queue (
    id BIGSERIAL PRIMARY KEY,
    notification_id BIGINT REFERENCES mail_notifications(id) ON DELETE CASCADE,
    recipient_email VARCHAR(320) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'queued',
    attempts INT NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error TEXT,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE RESTRICT,
    CONSTRAINT chk_mail_email_queue_status CHECK (status IN ('queued', 'processing', 'sent', 'failed', 'dead_letter'))
);
CREATE INDEX IF NOT EXISTS idx_mail_email_queue_delivery ON mail_email_queue(status, next_attempt_at, id);
CREATE INDEX IF NOT EXISTS idx_mail_email_queue_company ON mail_email_queue(company_id, status, next_attempt_at);

CREATE TRIGGER trg_mail_activity_types_updated_at BEFORE UPDATE ON mail_activity_types FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_mail_activities_updated_at BEFORE UPDATE ON mail_activities FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_mail_notifications_updated_at BEFORE UPDATE ON mail_notifications FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_mail_email_queue_updated_at BEFORE UPDATE ON mail_email_queue FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

INSERT INTO mail_activity_types (name, summary, category, icon, delay_count, delay_unit, system_type)
VALUES
    ('Email', 'Send an email', 'default', 'fa-envelope', 0, 'days', true),
    ('Call', 'Make a phone call', 'call', 'fa-phone', 0, 'days', true),
    ('Meeting', 'Schedule a meeting', 'meeting', 'fa-calendar', 0, 'days', true),
    ('To-Do', 'Complete a task', 'default', 'fa-check', 0, 'days', true),
    ('Document', 'Send or review a document', 'upload_file', 'fa-upload', 0, 'days', true),
    ('Exception', 'Handle an exception', 'exception', 'fa-warning', 0, 'days', true)
ON CONFLICT (name, company_id) DO NOTHING;

INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'mail.activity', true, true, true, false),
    (2::BIGINT, 'mail.activity', true, true, true, true),
    (1::BIGINT, 'mail.activity.type', true, false, false, false),
    (2::BIGINT, 'mail.activity.type', true, true, true, true),
    (1::BIGINT, 'mail.message', true, true, false, false),
    (2::BIGINT, 'mail.message', true, true, true, true),
    (1::BIGINT, 'mail.notification', true, true, true, false),
    (2::BIGINT, 'mail.notification', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;
