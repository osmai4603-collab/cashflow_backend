-- Purchase requisition schema: blanket orders and templates in Odoo-style workflow

CREATE TABLE IF NOT EXISTS purchase_requisitions (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    requisition_type VARCHAR(32) NOT NULL DEFAULT 'blanket_order',
    vendor_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    user_id BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    date_start TIMESTAMPTZ,
    date_end TIMESTAMPTZ,
    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    currency_id BIGINT REFERENCES res_currencies(id) ON DELETE RESTRICT,
    company_id BIGINT REFERENCES res_companies(id) ON DELETE CASCADE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES res_users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_vendor_id ON purchase_requisitions(vendor_id);
CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_user_id ON purchase_requisitions(user_id);
CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_state ON purchase_requisitions(state);
CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_company_id ON purchase_requisitions(company_id);
CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_date_start ON purchase_requisitions(date_start);

CREATE TRIGGER trg_purchase_requisitions_updated_at
    BEFORE UPDATE ON purchase_requisitions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS purchase_requisition_lines (
    id BIGSERIAL PRIMARY KEY,
    requisition_id BIGINT NOT NULL REFERENCES purchase_requisitions(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    product_qty NUMERIC(15, 4) NOT NULL DEFAULT 1.0000,
    product_uom BIGINT REFERENCES uom_uoms(id) ON DELETE SET NULL,
    price_unit NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    schedule_date TIMESTAMPTZ,
    supplier_id BIGINT REFERENCES res_partners(id) ON DELETE SET NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_requisition_lines_requisition_id ON purchase_requisition_lines(requisition_id);
CREATE INDEX IF NOT EXISTS idx_purchase_requisition_lines_product_id ON purchase_requisition_lines(product_id);

CREATE TRIGGER trg_purchase_requisition_lines_updated_at
    BEFORE UPDATE ON purchase_requisition_lines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
