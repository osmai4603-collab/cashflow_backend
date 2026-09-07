-- 000022_sale_purchase_stock_integration.down.sql

ALTER TABLE stock_moves DROP COLUMN IF EXISTS procurement_group_id;
ALTER TABLE stock_pickings DROP COLUMN IF EXISTS procurement_group_id;
ALTER TABLE purchase_orders DROP COLUMN IF EXISTS procurement_group_id;
ALTER TABLE purchase_orders DROP COLUMN IF EXISTS receipt_status;
ALTER TABLE sale_order_lines DROP COLUMN IF EXISTS route_id;
ALTER TABLE sale_orders DROP COLUMN IF EXISTS procurement_group_id;
ALTER TABLE sale_orders DROP COLUMN IF EXISTS delivery_status;

DROP TABLE IF EXISTS stock_procurement_groups;
