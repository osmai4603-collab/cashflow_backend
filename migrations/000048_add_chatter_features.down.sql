DROP TABLE IF EXISTS mail_tracking_values;
DROP TABLE IF EXISTS mail_followers_subtypes_rel;
DROP TABLE IF EXISTS mail_followers;
ALTER TABLE mail_messages DROP COLUMN IF EXISTS subtype_id;
ALTER TABLE mail_messages DROP COLUMN IF EXISTS parent_id;
DROP TABLE IF EXISTS mail_message_subtypes;
