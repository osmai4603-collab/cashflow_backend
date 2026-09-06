-- 000005_create_sales_schema.down.sql
-- Revert sales schema

DROP TABLE IF EXISTS sale_order_invoices CASCADE;
DROP TABLE IF EXISTS sale_order_lines CASCADE;
DROP TABLE IF EXISTS sale_orders CASCADE;
DROP SEQUENCE IF EXISTS sale_order_seq;
