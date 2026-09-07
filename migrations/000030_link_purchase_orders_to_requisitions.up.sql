ALTER TABLE purchase_orders
    ADD COLUMN IF NOT EXISTS requisition_id BIGINT REFERENCES purchase_requisitions(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS requisition_type VARCHAR(32);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_requisition_id
    ON purchase_orders(requisition_id);
