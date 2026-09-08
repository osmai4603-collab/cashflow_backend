DROP INDEX IF EXISTS idx_purchase_requisition_lines_supplier;
DROP INDEX IF EXISTS idx_purchase_requisitions_active;
DROP INDEX IF EXISTS idx_purchase_requisitions_type;

ALTER TABLE purchase_requisition_lines
    DROP CONSTRAINT IF EXISTS purchase_requisition_lines_ordered_quantity_check,
    DROP CONSTRAINT IF EXISTS purchase_requisition_lines_price_check,
    DROP CONSTRAINT IF EXISTS purchase_requisition_lines_quantity_check,
    DROP COLUMN IF EXISTS product_description_variants,
    DROP COLUMN IF EXISTS qty_ordered;

ALTER TABLE purchase_requisitions
    DROP CONSTRAINT IF EXISTS purchase_requisitions_order_count_check,
    DROP CONSTRAINT IF EXISTS purchase_requisitions_dates_check,
    DROP CONSTRAINT IF EXISTS purchase_requisitions_state_check,
    DROP CONSTRAINT IF EXISTS purchase_requisitions_type_check,
    DROP COLUMN IF EXISTS order_count,
    DROP COLUMN IF EXISTS reference,
    DROP COLUMN IF EXISTS active;

DELETE FROM ir_sequences
WHERE code IN ('purchase.requisition.blanket.order', 'purchase.requisition.purchase.template');