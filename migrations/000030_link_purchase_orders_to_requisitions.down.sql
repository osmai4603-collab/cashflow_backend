DROP INDEX IF EXISTS idx_purchase_orders_requisition_id;

ALTER TABLE purchase_orders
    DROP COLUMN IF EXISTS requisition_type,
    DROP COLUMN IF EXISTS requisition_id;
