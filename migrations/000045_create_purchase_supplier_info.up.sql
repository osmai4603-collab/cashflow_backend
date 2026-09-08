CREATE TABLE IF NOT EXISTS purchase_supplier_infos (
    id BIGSERIAL PRIMARY KEY,
    requisition_id BIGINT NOT NULL REFERENCES purchase_requisitions(id) ON DELETE CASCADE,
    requisition_line_id BIGINT NOT NULL UNIQUE REFERENCES purchase_requisition_lines(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    vendor_id BIGINT NOT NULL REFERENCES res_partners(id) ON DELETE RESTRICT,
    product_uom BIGINT REFERENCES uom_uoms(id) ON DELETE SET NULL,
    price NUMERIC(15, 4) NOT NULL CHECK (price > 0),
    currency_id BIGINT REFERENCES res_currencies(id) ON DELETE RESTRICT,
    company_id BIGINT REFERENCES res_companies(id) ON DELETE RESTRICT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_supplier_infos_requisition
    ON purchase_supplier_infos(requisition_id);
CREATE INDEX IF NOT EXISTS idx_purchase_supplier_infos_product_vendor
    ON purchase_supplier_infos(product_id, vendor_id);

CREATE TRIGGER trg_purchase_supplier_infos_updated_at
    BEFORE UPDATE ON purchase_supplier_infos
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();