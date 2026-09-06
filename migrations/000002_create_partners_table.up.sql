-- 000002_create_partners_table.up.sql
-- Res Partners table representing contacts, customers, suppliers, and companies

CREATE TABLE IF NOT EXISTS res_partners (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email CITEXT,
    phone VARCHAR(50),
    mobile VARCHAR(50),
    type VARCHAR(20) NOT NULL DEFAULT 'individual',
    is_customer BOOLEAN NOT NULL DEFAULT true,
    is_supplier BOOLEAN NOT NULL DEFAULT false,
    vat_number VARCHAR(50),
    website VARCHAR(255),
    company_id BIGINT,
    parent_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    street VARCHAR(255),
    street2 VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(100),
    country VARCHAR(100),
    zip_code VARCHAR(20),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

-- Indexes for performant searching and filtering
CREATE INDEX IF NOT EXISTS idx_partners_name ON res_partners(name);
CREATE INDEX IF NOT EXISTS idx_partners_email ON res_partners(email);
CREATE INDEX IF NOT EXISTS idx_partners_active ON res_partners(active);
CREATE INDEX IF NOT EXISTS idx_partners_is_customer ON res_partners(is_customer) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_partners_is_supplier ON res_partners(is_supplier) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_partners_company_id ON res_partners(company_id);
CREATE INDEX IF NOT EXISTS idx_partners_parent_id ON res_partners(parent_id);

-- Trigger for auto-updating updated_at timestamp
CREATE TRIGGER trg_partners_updated_at
    BEFORE UPDATE ON res_partners
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
