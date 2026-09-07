-- 000012_create_analytic_schema.up.sql
-- Analytic Accounting schema (Phase 10) built against Odoo 19.0 addons/analytic:
--   plans + applicability rules (G2), JSONB distribution (G3), accounts, lines,
--   automatic distribution models (G7), and the project-plan config (G8).

-- 1. Analytic Plans (account.analytic.plan)
CREATE TABLE IF NOT EXISTS account_analytic_plan (
    id                      BIGSERIAL PRIMARY KEY,
    name                    TEXT NOT NULL,
    description             TEXT,
    parent_id               BIGINT REFERENCES account_analytic_plan(id) ON DELETE CASCADE,
    parent_path             TEXT,
    root_id                 BIGINT,
    sequence                INT DEFAULT 10,
    color                   INT,
    default_applicability   TEXT NOT NULL DEFAULT 'optional', -- optional|mandatory|unavailable
    active                  BOOLEAN DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              BIGINT,
    updated_by              BIGINT
);

CREATE INDEX IF NOT EXISTS idx_analytic_plan_parent ON account_analytic_plan(parent_id);
CREATE INDEX IF NOT EXISTS idx_analytic_plan_root ON account_analytic_plan(root_id);
CREATE INDEX IF NOT EXISTS idx_analytic_plan_active ON account_analytic_plan(active);

CREATE TRIGGER trg_account_analytic_plan_updated_at
    BEFORE UPDATE ON account_analytic_plan
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 2. Applicability Rules (account.analytic.applicability - G2)
CREATE TABLE IF NOT EXISTS account_analytic_applicability (
    id                BIGSERIAL PRIMARY KEY,
    analytic_plan_id  BIGINT NOT NULL REFERENCES account_analytic_plan(id) ON DELETE CASCADE,
    business_domain   TEXT NOT NULL DEFAULT 'general', -- general|sale_order|purchase_order|expense
    applicability     TEXT NOT NULL,                   -- optional|mandatory|unavailable
    company_id        BIGINT,                          -- NULL = applies to all companies
    sequence          INT DEFAULT 10,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by        BIGINT,
    updated_by        BIGINT
);

CREATE INDEX IF NOT EXISTS idx_analytic_applicability_plan ON account_analytic_applicability(analytic_plan_id);
CREATE INDEX IF NOT EXISTS idx_analytic_applicability_domain ON account_analytic_applicability(business_domain);
CREATE INDEX IF NOT EXISTS idx_analytic_applicability_company ON account_analytic_applicability(company_id);

CREATE TRIGGER trg_account_analytic_applicability_updated_at
    BEFORE UPDATE ON account_analytic_applicability
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 3. Analytic Accounts (account.analytic.account)
CREATE TABLE IF NOT EXISTS account_analytic_account (
    id             BIGSERIAL PRIMARY KEY,
    name           TEXT NOT NULL,
    code           TEXT,
    plan_id        BIGINT NOT NULL REFERENCES account_analytic_plan(id) ON DELETE RESTRICT,
    root_plan_id   BIGINT,
    partner_id     BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    color          INT,
    company_id     BIGINT,             -- NULL = shared across companies
    active         BOOLEAN DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by     BIGINT,
    updated_by     BIGINT
);

CREATE INDEX IF NOT EXISTS idx_analytic_account_plan ON account_analytic_account(plan_id);
CREATE INDEX IF NOT EXISTS idx_analytic_account_partner ON account_analytic_account(partner_id);
CREATE INDEX IF NOT EXISTS idx_analytic_account_active ON account_analytic_account(active);
CREATE UNIQUE INDEX IF NOT EXISTS idx_analytic_account_code ON account_analytic_account(code) WHERE code IS NOT NULL;

CREATE TRIGGER trg_account_analytic_account_updated_at
    BEFORE UPDATE ON account_analytic_account
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 4. Analytic Lines (account.analytic.line - G4, G5)
CREATE TABLE IF NOT EXISTS account_analytic_line (
    id                   BIGSERIAL PRIMARY KEY,
    name                 TEXT NOT NULL,                    -- G5: label / description
    date                 DATE NOT NULL,
    amount               DOUBLE PRECISION NOT NULL DEFAULT 0,
    unit_amount          DOUBLE PRECISION NOT NULL DEFAULT 0,
    product_uom_id       BIGINT,
    partner_id           BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    user_id              BIGINT NOT NULL,
    company_id           BIGINT NOT NULL,
    currency_code        TEXT NOT NULL DEFAULT 'USD',
    category             TEXT NOT NULL DEFAULT 'other',
    account_id           BIGINT NOT NULL REFERENCES account_analytic_account(id) ON DELETE RESTRICT, -- "Project plan" column (G4)
    analytic_distribution JSONB,                           -- G3: {account_ids: percentage}
    move_line_id         BIGINT,                           -- link to account_move_lines
    general_account_id   BIGINT,                           -- linked general ledger account
    source               TEXT NOT NULL DEFAULT 'manual',   -- manual|invoice|vendor_bill|sale_order|employee
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by           BIGINT,
    updated_by           BIGINT
);

CREATE INDEX IF NOT EXISTS idx_analytic_line_date ON account_analytic_line(date);
CREATE INDEX IF NOT EXISTS idx_analytic_line_account ON account_analytic_line(account_id);
CREATE INDEX IF NOT EXISTS idx_analytic_line_move ON account_analytic_line(move_line_id);
CREATE INDEX IF NOT EXISTS idx_analytic_line_company ON account_analytic_line(company_id);
CREATE INDEX IF NOT EXISTS idx_analytic_line_source ON account_analytic_line(source);
-- G3: GIN index to search analytic accounts referenced inside the JSONB distribution.
CREATE INDEX IF NOT EXISTS idx_analytic_line_dist_gin ON account_analytic_line
    USING gin(regexp_split_to_array(jsonb_path_query_array(
        analytic_distribution, '$.keyvalue()."key"')::text, '\D+'))
    WHERE analytic_distribution IS NOT NULL;

CREATE TRIGGER trg_account_analytic_line_updated_at
    BEFORE UPDATE ON account_analytic_line
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 5. Automatic Distribution Models (account.analytic.distribution.model - G7)
CREATE TABLE IF NOT EXISTS account_analytic_distribution_model (
    id                    BIGSERIAL PRIMARY KEY,
    sequence              INT DEFAULT 10,
    partner_id            BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    partner_category_id   BIGINT,                          -- FK added when partner categories land (M13)
    company_id            BIGINT,
    analytic_distribution JSONB NOT NULL,
    active                BOOLEAN DEFAULT TRUE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by            BIGINT,
    updated_by            BIGINT
);

CREATE INDEX IF NOT EXISTS idx_analytic_dist_model_partner ON account_analytic_distribution_model(partner_id);
CREATE INDEX IF NOT EXISTS idx_analytic_dist_model_seq ON account_analytic_distribution_model(sequence);
CREATE INDEX IF NOT EXISTS idx_analytic_dist_model_active ON account_analytic_distribution_model(active);

CREATE TRIGGER trg_account_analytic_distribution_model_updated_at
    BEFORE UPDATE ON account_analytic_distribution_model
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 6. Seed Data (analytic_data.xml equivalent - G9)
-- Default "Project" analytic plan (used by G8 project plan config).
INSERT INTO account_analytic_plan (id, name, description, sequence, color, default_applicability, active) VALUES
(1, 'Project', 'Default project plan (analytic.project_plan)', 10, 0, 'optional', true),
(2, 'Departments', 'Department cost center plan (demo)', 20, 1, 'optional', true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('account_analytic_plan_id_seq', (SELECT COALESCE(MAX(id), 1) FROM account_analytic_plan));

-- Default analytic accounts for each seeded plan.
INSERT INTO account_analytic_account (id, name, code, plan_id, root_plan_id, active) VALUES
(1, 'General', 'PRJ', 1, 1, true),
(2, 'Administration', 'ADM', 2, 2, true),
(3, 'Sales & Marketing', 'S&M', 2, 2, true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('account_analytic_account_id_seq', (SELECT COALESCE(MAX(id), 1) FROM account_analytic_account));

-- G8: analytic.project_plan -> id of the "Project" plan.
INSERT INTO ir_config_parameters (key, value, company_id) VALUES
('analytic.project_plan', '1', NULL)
ON CONFLICT (key) DO NOTHING;