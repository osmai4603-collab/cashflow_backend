-- migrations/000066_marketing_automation_schema.up.sql

CREATE TABLE marketing_campaigns (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    utm_source      VARCHAR(64) NOT NULL DEFAULT 'marketing',
    utm_medium      VARCHAR(64) NOT NULL DEFAULT 'email',
    utm_campaign    VARCHAR(128) NOT NULL,
    total_sent      INT NOT NULL DEFAULT 0,
    total_delivered INT NOT NULL DEFAULT 0,
    total_opened    INT NOT NULL DEFAULT 0,
    total_clicked   INT NOT NULL DEFAULT 0,
    total_bounced   INT NOT NULL DEFAULT 0,
    total_revenue   NUMERIC(15,4) NOT NULL DEFAULT 0,
    state           VARCHAR(32) NOT NULL DEFAULT 'draft',
    company_id      BIGINT NOT NULL REFERENCES companies(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE mailing_lists (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    is_public   BOOLEAN NOT NULL DEFAULT FALSE,
    company_id  BIGINT NOT NULL REFERENCES companies(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE mailing_contacts (
    id           BIGSERIAL PRIMARY KEY,
    partner_id   BIGINT REFERENCES partners(id),
    email        VARCHAR(128) NOT NULL,
    mobile       VARCHAR(32),
    name         VARCHAR(128) NOT NULL,
    is_opt_out   BOOLEAN NOT NULL DEFAULT FALSE,
    is_blacklist BOOLEAN NOT NULL DEFAULT FALSE,
    company_id   BIGINT NOT NULL REFERENCES companies(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(email, company_id)
);

CREATE TABLE mailing_list_contact_rel (
    list_id    BIGINT NOT NULL REFERENCES mailing_lists(id) ON DELETE CASCADE,
    contact_id BIGINT NOT NULL REFERENCES mailing_contacts(id) ON DELETE CASCADE,
    PRIMARY KEY(list_id, contact_id)
);

CREATE TABLE mass_mailings (
    id             BIGSERIAL PRIMARY KEY,
    campaign_id    BIGINT REFERENCES marketing_campaigns(id),
    subject        VARCHAR(256) NOT NULL,
    sender_name    VARCHAR(128) NOT NULL,
    sender_email   VARCHAR(128) NOT NULL,
    reply_to       VARCHAR(128),
    body_html      TEXT NOT NULL,
    scheduled_date TIMESTAMPTZ,
    sent_date      TIMESTAMPTZ,
    state          VARCHAR(32) NOT NULL DEFAULT 'draft',
    company_id     BIGINT NOT NULL REFERENCES companies(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE mailing_traces (
    id            BIGSERIAL PRIMARY KEY,
    mailing_id    BIGINT NOT NULL REFERENCES mass_mailings(id) ON DELETE CASCADE,
    contact_id    BIGINT NOT NULL REFERENCES mailing_contacts(id),
    email         VARCHAR(128) NOT NULL,
    sent_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at  TIMESTAMPTZ,
    opened_at     TIMESTAMPTZ,
    clicked_at    TIMESTAMPTZ,
    bounced_at    TIMESTAMPTZ,
    bounce_reason TEXT,
    tracking_code VARCHAR(64) NOT NULL UNIQUE
);
CREATE INDEX idx_mailing_traces_code ON mailing_traces(tracking_code);

CREATE TABLE marketing_automations (
    id           BIGSERIAL PRIMARY KEY,
    name         VARCHAR(128) NOT NULL,
    trigger_type VARCHAR(64) NOT NULL,
    target_model VARCHAR(64) NOT NULL,
    filter_json  JSONB NOT NULL DEFAULT '{}',
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    company_id   BIGINT NOT NULL REFERENCES companies(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE marketing_automation_activities (
    id             BIGSERIAL PRIMARY KEY,
    automation_id  BIGINT NOT NULL REFERENCES marketing_automations(id) ON DELETE CASCADE,
    parent_id      BIGINT REFERENCES marketing_automation_activities(id) ON DELETE SET NULL,
    action_type    VARCHAR(32) NOT NULL,
    delay_hours    INT NOT NULL DEFAULT 0,
    condition_type VARCHAR(32) NOT NULL DEFAULT 'none',
    template_id    BIGINT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
