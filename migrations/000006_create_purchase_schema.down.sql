-- 000006_create_purchase_schema.down.sql
-- Revert Purchase Order schema

DROP TABLE IF EXISTS purchase_order_invoices CASCADE;
DROP TABLE IF EXISTS purchase_order_lines CASCADE;
DROP TABLE IF EXISTS purchase_orders CASCADE;
DROP SEQUENCE IF EXISTS purchase_order_seq CASCADE;
