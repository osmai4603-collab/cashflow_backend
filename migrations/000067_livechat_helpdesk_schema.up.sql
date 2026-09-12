-- migrations/000067_livechat_helpdesk_schema.up.sql

CREATE TABLE livechat_channels (
    id           BIGSERIAL PRIMARY KEY,
    name         VARCHAR(128) NOT NULL,
    welcome_msg  TEXT NOT NULL DEFAULT 'مرحباً بك! كيف يمكننا مساعدتك اليوم؟',
    button_text  VARCHAR(64) NOT NULL DEFAULT 'تحدث معنا',
    header_color VARCHAR(16) NOT NULL DEFAULT '#1E3A8A',
    company_id   BIGINT NOT NULL REFERENCES res_companies(id),
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE livechat_channel_users (
    channel_id BIGINT NOT NULL REFERENCES livechat_channels(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES res_users(id) ON DELETE CASCADE,
    PRIMARY KEY(channel_id, user_id)
);

CREATE TABLE livechat_sessions (
    id                   BIGSERIAL PRIMARY KEY,
    channel_id           BIGINT NOT NULL REFERENCES livechat_channels(id),
    operator_id          BIGINT REFERENCES res_users(id),
    visitor_uuid         VARCHAR(64) NOT NULL,
    visitor_name         VARCHAR(128) NOT NULL DEFAULT 'زائر',
    visitor_email        VARCHAR(128),
    partner_id           BIGINT REFERENCES res_partners(id),
    status               VARCHAR(32) NOT NULL DEFAULT 'active',
    rating_score         INT CHECK (rating_score BETWEEN 1 AND 5),
    rating_comment       TEXT,
    converted_ticket_id  BIGINT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at            TIMESTAMPTZ
);
CREATE INDEX idx_livechat_sessions_visitor ON livechat_sessions(visitor_uuid);

CREATE TABLE livechat_messages (
    id          BIGSERIAL PRIMARY KEY,
    session_id  BIGINT NOT NULL REFERENCES livechat_sessions(id) ON DELETE CASCADE,
    sender_type VARCHAR(16) NOT NULL, -- visitor, operator, system
    sender_id   BIGINT,
    body        TEXT NOT NULL,
    file_url    TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE helpdesk_teams (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(128) NOT NULL,
    email      VARCHAR(128),
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    active     BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE helpdesk_stages (
    id         BIGSERIAL PRIMARY KEY,
    team_id    BIGINT NOT NULL REFERENCES helpdesk_teams(id) ON DELETE CASCADE,
    name       VARCHAR(64) NOT NULL,
    sequence   INT NOT NULL DEFAULT 10,
    is_closed  BOOLEAN NOT NULL DEFAULT FALSE,
    company_id BIGINT NOT NULL REFERENCES res_companies(id)
);

CREATE TABLE helpdesk_sla_policies (
    id                   BIGSERIAL PRIMARY KEY,
    name                 VARCHAR(128) NOT NULL,
    team_id              BIGINT NOT NULL REFERENCES helpdesk_teams(id) ON DELETE CASCADE,
    priority             VARCHAR(8) NOT NULL DEFAULT '1',
    max_hours_first_resp NUMERIC(6,2) NOT NULL DEFAULT 4.0,
    max_hours_resolution NUMERIC(6,2) NOT NULL DEFAULT 24.0,
    working_calendar_id  BIGINT,
    active               BOOLEAN NOT NULL DEFAULT TRUE,
    company_id           BIGINT NOT NULL REFERENCES res_companies(id)
);

CREATE TABLE helpdesk_tickets (
    id                 BIGSERIAL PRIMARY KEY,
    number             VARCHAR(64) NOT NULL UNIQUE,
    name               VARCHAR(256) NOT NULL,
    description        TEXT NOT NULL,
    team_id            BIGINT NOT NULL REFERENCES helpdesk_teams(id),
    stage_id           BIGINT NOT NULL REFERENCES helpdesk_stages(id),
    priority           VARCHAR(8) NOT NULL DEFAULT '1',
    partner_id         BIGINT REFERENCES res_partners(id),
    partner_email      VARCHAR(128) NOT NULL,
    partner_phone      VARCHAR(32),
    assigned_user_id   BIGINT REFERENCES res_users(id),
    sale_order_id      BIGINT REFERENCES sale_orders(id),
    stock_picking_id   BIGINT REFERENCES stock_pickings(id),
    repair_order_id    BIGINT,
    first_response_at  TIMESTAMPTZ,
    closed_at          TIMESTAMPTZ,
    sla_breach         BOOLEAN NOT NULL DEFAULT FALSE,
    company_id         BIGINT NOT NULL REFERENCES res_companies(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_helpdesk_tickets_team ON helpdesk_tickets(team_id);
CREATE INDEX idx_helpdesk_tickets_partner ON helpdesk_tickets(partner_id);

CREATE TABLE knowledge_categories (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(128) NOT NULL,
    sequence   INT NOT NULL DEFAULT 10,
    company_id BIGINT NOT NULL REFERENCES res_companies(id)
);

CREATE TABLE knowledge_articles (
    id            BIGSERIAL PRIMARY KEY,
    category_id   BIGINT NOT NULL REFERENCES knowledge_categories(id) ON DELETE CASCADE,
    title         VARCHAR(256) NOT NULL,
    slug          VARCHAR(256) NOT NULL UNIQUE,
    content_html  TEXT NOT NULL,
    is_internal   BOOLEAN NOT NULL DEFAULT FALSE,
    view_count    INT NOT NULL DEFAULT 0,
    helpful_count INT NOT NULL DEFAULT 0,
    company_id    BIGINT NOT NULL REFERENCES res_companies(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
