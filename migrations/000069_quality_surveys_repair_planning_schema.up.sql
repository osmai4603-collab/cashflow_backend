-- migrations/000069_quality_surveys_repair_planning_schema.up.sql

CREATE TABLE quality_control_points (
    id           BIGSERIAL PRIMARY KEY,
    name         VARCHAR(128) NOT NULL,
    product_id   BIGINT REFERENCES products(id),
    category_id  BIGINT REFERENCES product_categories(id),
    trigger      VARCHAR(32) NOT NULL,
    test_type    VARCHAR(32) NOT NULL DEFAULT 'pass_fail',
    norm_min     NUMERIC(10,4),
    norm_max     NUMERIC(10,4),
    instructions TEXT,
    company_id   BIGINT NOT NULL REFERENCES companies(id),
    active       BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE quality_alerts (
    id           BIGSERIAL PRIMARY KEY,
    name         VARCHAR(128) NOT NULL,
    product_id   BIGINT NOT NULL REFERENCES products(id),
    lot_id       BIGINT REFERENCES stock_lots(id),
    picking_id   BIGINT REFERENCES stock_pickings(id),
    description  TEXT NOT NULL,
    action_taken TEXT,
    stage        VARCHAR(32) NOT NULL DEFAULT 'new',
    company_id   BIGINT NOT NULL REFERENCES companies(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE survey_surveys (
    id          BIGSERIAL PRIMARY KEY,
    title       VARCHAR(256) NOT NULL,
    description TEXT,
    is_scoring  BOOLEAN NOT NULL DEFAULT FALSE,
    passing_score NUMERIC(5,2) DEFAULT 70.0,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    company_id  BIGINT NOT NULL REFERENCES companies(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE survey_questions (
    id         BIGSERIAL PRIMARY KEY,
    survey_id  BIGINT NOT NULL REFERENCES survey_surveys(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    type       VARCHAR(32) NOT NULL, -- single_choice, multiple_choice, text, rating
    sequence   INT NOT NULL DEFAULT 10
);

CREATE TABLE repair_orders (
    id               BIGSERIAL PRIMARY KEY,
    name             VARCHAR(64) NOT NULL UNIQUE,
    partner_id       BIGINT NOT NULL REFERENCES partners(id),
    product_id       BIGINT NOT NULL REFERENCES products(id),
    product_lot_id   BIGINT REFERENCES stock_lots(id),
    warranty_check   BOOLEAN NOT NULL DEFAULT FALSE,
    state            VARCHAR(32) NOT NULL DEFAULT 'draft',
    location_id      BIGINT NOT NULL REFERENCES stock_locations(id),
    location_dest_id BIGINT NOT NULL REFERENCES stock_locations(id),
    amount_total     NUMERIC(15,4) NOT NULL DEFAULT 0,
    account_move_id  BIGINT REFERENCES account_moves(id),
    company_id       BIGINT NOT NULL REFERENCES companies(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE repair_order_lines (
    id          BIGSERIAL PRIMARY KEY,
    repair_id   BIGINT NOT NULL REFERENCES repair_orders(id) ON DELETE CASCADE,
    product_id  BIGINT NOT NULL REFERENCES products(id),
    quantity    NUMERIC(15,4) NOT NULL DEFAULT 1,
    price_unit  NUMERIC(15,4) NOT NULL DEFAULT 0,
    price_total NUMERIC(15,4) NOT NULL DEFAULT 0
);

CREATE TABLE planning_roles (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(64) NOT NULL,
    color      VARCHAR(16) DEFAULT '#3B82F6',
    company_id BIGINT NOT NULL REFERENCES companies(id)
);

CREATE TABLE planning_shifts (
    id              BIGSERIAL PRIMARY KEY,
    employee_id     BIGINT REFERENCES hr_employees(id),
    role_id         BIGINT NOT NULL REFERENCES planning_roles(id),
    start_at        TIMESTAMPTZ NOT NULL,
    end_at          TIMESTAMPTZ NOT NULL,
    allocated_hours NUMERIC(6,2) NOT NULL,
    is_published    BOOLEAN NOT NULL DEFAULT FALSE,
    company_id      BIGINT NOT NULL REFERENCES companies(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_planning_shifts_time ON planning_shifts(employee_id, start_at, end_at);
