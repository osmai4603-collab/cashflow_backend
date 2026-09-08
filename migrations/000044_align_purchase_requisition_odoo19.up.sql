-- Align the existing requisition schema with the Odoo 19 domain contract.

ALTER TABLE purchase_requisitions
    ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS reference VARCHAR(255),
    ADD COLUMN IF NOT EXISTS order_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE purchase_requisition_lines
    ADD COLUMN IF NOT EXISTS qty_ordered NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    ADD COLUMN IF NOT EXISTS product_description_variants VARCHAR(255);

ALTER TABLE purchase_requisitions
    DROP CONSTRAINT IF EXISTS purchase_requisitions_type_check,
    DROP CONSTRAINT IF EXISTS purchase_requisitions_state_check,
    DROP CONSTRAINT IF EXISTS purchase_requisitions_dates_check,
    DROP CONSTRAINT IF EXISTS purchase_requisitions_order_count_check;

ALTER TABLE purchase_requisition_lines
    DROP CONSTRAINT IF EXISTS purchase_requisition_lines_quantity_check,
    DROP CONSTRAINT IF EXISTS purchase_requisition_lines_price_check,
    DROP CONSTRAINT IF EXISTS purchase_requisition_lines_ordered_quantity_check;

ALTER TABLE purchase_requisitions
    ADD CONSTRAINT purchase_requisitions_type_check
        CHECK (requisition_type IN ('blanket_order', 'purchase_template')),
    ADD CONSTRAINT purchase_requisitions_state_check
        CHECK (state IN ('draft', 'confirmed', 'done', 'cancel')),
    ADD CONSTRAINT purchase_requisitions_dates_check
        CHECK (date_end IS NULL OR date_start IS NULL OR date_end >= date_start),
    ADD CONSTRAINT purchase_requisitions_order_count_check
        CHECK (order_count >= 0);

ALTER TABLE purchase_requisition_lines
    ADD CONSTRAINT purchase_requisition_lines_quantity_check
        CHECK (product_qty > 0),
    ADD CONSTRAINT purchase_requisition_lines_price_check
        CHECK (price_unit >= 0),
    ADD CONSTRAINT purchase_requisition_lines_ordered_quantity_check
        CHECK (qty_ordered >= 0);

CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_type
    ON purchase_requisitions(requisition_type);
CREATE INDEX IF NOT EXISTS idx_purchase_requisitions_active
    ON purchase_requisitions(active);
CREATE INDEX IF NOT EXISTS idx_purchase_requisition_lines_supplier
    ON purchase_requisition_lines(supplier_id);

INSERT INTO ir_sequences (name, code, prefix, suffix, padding, start_number, current_number, sequence_type, company_id)
VALUES
    ('Blanket Order', 'purchase.requisition.blanket.order', 'BO', '', 5, 1, 0, 'normal', NULL),
    ('Purchase Template', 'purchase.requisition.purchase.template', 'PT', '', 5, 1, 0, 'normal', NULL)
ON CONFLICT (code) DO NOTHING;