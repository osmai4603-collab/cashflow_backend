-- 000008_create_crm_schema.up.sql
-- CRM & Sales Pipeline Schema: Stages, Lost Reasons, Tags, Leads & Opportunities

-- 1. Stages (crm.stage in Odoo)
CREATE TABLE IF NOT EXISTS crm_stages (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    sequence INT NOT NULL DEFAULT 10,
    is_won BOOLEAN NOT NULL DEFAULT false,
    is_closed BOOLEAN NOT NULL DEFAULT false,
    fold BOOLEAN NOT NULL DEFAULT false,
    requirements TEXT,
    company_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_crm_stages_sequence ON crm_stages(sequence);
CREATE INDEX IF NOT EXISTS idx_crm_stages_active ON crm_stages(active);

CREATE TRIGGER trg_crm_stages_updated_at
    BEFORE UPDATE ON crm_stages
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed Default Pipeline Stages
INSERT INTO crm_stages (id, name, sequence, is_won, is_closed, fold, active)
VALUES 
    (1, 'New', 10, false, false, false, true),
    (2, 'Qualified', 20, false, false, false, true),
    (3, 'Proposition', 30, false, false, false, true),
    (4, 'Won', 40, true, true, false, true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('crm_stages_id_seq', (SELECT COALESCE(MAX(id), 1) FROM crm_stages));

-- 2. Lost Reasons (crm.lost.reason in Odoo)
CREATE TABLE IF NOT EXISTS crm_lost_reasons (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_crm_lost_reasons_active ON crm_lost_reasons(active);

CREATE TRIGGER trg_crm_lost_reasons_updated_at
    BEFORE UPDATE ON crm_lost_reasons
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed Default Lost Reasons
INSERT INTO crm_lost_reasons (id, name, active)
VALUES 
    (1, 'Too expensive', true),
    (2, 'We don''t have people/skills', true),
    (3, 'Not enough features', true),
    (4, 'Lost to competitor', true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('crm_lost_reasons_id_seq', (SELECT COALESCE(MAX(id), 1) FROM crm_lost_reasons));

-- 3. CRM Tags (crm.tag in Odoo)
CREATE TABLE IF NOT EXISTS crm_tags (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    color INT DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_crm_tags_updated_at
    BEFORE UPDATE ON crm_tags
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Seed Default Tags
INSERT INTO crm_tags (id, name, color, active)
VALUES 
    (1, 'Software', 1, true),
    (2, 'Consulting', 2, true),
    (3, 'Support', 3, true),
    (4, 'Enterprise', 4, true)
ON CONFLICT (id) DO NOTHING;

SELECT setval('crm_tags_id_seq', (SELECT COALESCE(MAX(id), 1) FROM crm_tags));

-- 4. Leads & Opportunities (crm.lead in Odoo)
CREATE TABLE IF NOT EXISTS crm_leads (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'lead', -- 'lead', 'opportunity'
    partner_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    partner_name VARCHAR(255),
    contact_name VARCHAR(255),
    email_from CITEXT,
    phone VARCHAR(50),
    stage_id BIGINT NOT NULL REFERENCES crm_stages(id) ON DELETE RESTRICT,
    salesperson_id BIGINT,
    expected_revenue NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    prorated_revenue NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    probability NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    source VARCHAR(64),
    priority VARCHAR(20) NOT NULL DEFAULT '1', -- '0'=Low, '1'=Medium, '2'=High, '3'=Very High
    lost_reason_id BIGINT REFERENCES crm_lost_reasons(id) ON DELETE SET NULL,
    lost_feedback TEXT,
    date_deadline TIMESTAMPTZ,
    date_closed TIMESTAMPTZ,
    date_conversion TIMESTAMPTZ,
    notes TEXT,
    company_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_crm_leads_name ON crm_leads(name);
CREATE INDEX IF NOT EXISTS idx_crm_leads_type ON crm_leads(type);
CREATE INDEX IF NOT EXISTS idx_crm_leads_stage_id ON crm_leads(stage_id);
CREATE INDEX IF NOT EXISTS idx_crm_leads_partner_id ON crm_leads(partner_id);
CREATE INDEX IF NOT EXISTS idx_crm_leads_salesperson_id ON crm_leads(salesperson_id);
CREATE INDEX IF NOT EXISTS idx_crm_leads_priority ON crm_leads(priority);
CREATE INDEX IF NOT EXISTS idx_crm_leads_active ON crm_leads(active);
CREATE INDEX IF NOT EXISTS idx_crm_leads_date_deadline ON crm_leads(date_deadline);

CREATE TRIGGER trg_crm_leads_updated_at
    BEFORE UPDATE ON crm_leads
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 5. Lead Tags Junction
CREATE TABLE IF NOT EXISTS crm_lead_tags (
    lead_id BIGINT NOT NULL REFERENCES crm_leads(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES crm_tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (lead_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_crm_lead_tags_tag_id ON crm_lead_tags(tag_id);
