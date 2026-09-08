CREATE TABLE IF NOT EXISTS mail_message_subtypes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    res_model VARCHAR(128),
    description TEXT,
    internal BOOLEAN NOT NULL DEFAULT false,
    default_subtype BOOLEAN NOT NULL DEFAULT true,
    sequence INT NOT NULL DEFAULT 10,
    CONSTRAINT uq_mail_message_subtypes_name_model UNIQUE (name, res_model)
);

ALTER TABLE mail_messages ADD COLUMN IF NOT EXISTS subtype_id BIGINT REFERENCES mail_message_subtypes(id) ON DELETE SET NULL;
ALTER TABLE mail_messages ADD COLUMN IF NOT EXISTS parent_id BIGINT REFERENCES mail_messages(id) ON DELETE CASCADE;

CREATE TABLE IF NOT EXISTS mail_followers (
    id BIGSERIAL PRIMARY KEY,
    res_model VARCHAR(128) NOT NULL,
    res_id BIGINT NOT NULL,
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES res_users(id) ON DELETE CASCADE,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE CASCADE,
    CONSTRAINT uq_mail_followers_res_partner UNIQUE (res_model, res_id, partner_id),
    CONSTRAINT uq_mail_followers_res_user UNIQUE (res_model, res_id, user_id),
    CONSTRAINT chk_mail_followers_target CHECK (partner_id IS NOT NULL OR user_id IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS mail_followers_subtypes_rel (
    follower_id BIGINT NOT NULL REFERENCES mail_followers(id) ON DELETE CASCADE,
    subtype_id BIGINT NOT NULL REFERENCES mail_message_subtypes(id) ON DELETE CASCADE,
    PRIMARY KEY (follower_id, subtype_id)
);

CREATE TABLE IF NOT EXISTS mail_tracking_values (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES mail_messages(id) ON DELETE CASCADE,
    field_name VARCHAR(64) NOT NULL,
    field_desc VARCHAR(128) NOT NULL,
    old_value_text TEXT,
    new_value_text TEXT
);

CREATE INDEX IF NOT EXISTS idx_mail_followers_resource ON mail_followers(res_model, res_id);
CREATE INDEX IF NOT EXISTS idx_mail_tracking_message ON mail_tracking_values(message_id);

INSERT INTO mail_message_subtypes (name, description, default_subtype, sequence)
VALUES
    ('discussions', 'Discussions', true, 1),
    ('mt_comment', 'Comment', true, 1),
    ('mt_note', 'Note', false, 10),
    ('activities', 'Activities', false, 20)
ON CONFLICT (name, res_model) DO NOTHING;
