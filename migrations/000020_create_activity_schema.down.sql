DROP TRIGGER IF EXISTS trg_mail_email_queue_updated_at ON mail_email_queue;
DROP TRIGGER IF EXISTS trg_mail_notifications_updated_at ON mail_notifications;
DROP TRIGGER IF EXISTS trg_mail_activities_updated_at ON mail_activities;
DROP TRIGGER IF EXISTS trg_mail_activity_types_updated_at ON mail_activity_types;

DELETE FROM res_group_permissions
WHERE model IN ('mail.activity', 'mail.activity.type', 'mail.message', 'mail.notification');

DROP TABLE IF EXISTS mail_email_queue;
DROP TABLE IF EXISTS mail_notifications;
DROP TABLE IF EXISTS mail_messages;
DROP TABLE IF EXISTS mail_activities;
DROP TABLE IF EXISTS mail_activity_types;
