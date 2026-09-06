-- 000007_create_stock_schema.down.sql
-- Rollback for stock schema

DROP TABLE IF EXISTS stock_quants CASCADE;
DROP TABLE IF EXISTS stock_moves CASCADE;
DROP TABLE IF EXISTS stock_pickings CASCADE;
DROP TABLE IF EXISTS stock_warehouses CASCADE;
DROP TABLE IF EXISTS stock_locations CASCADE;

DROP SEQUENCE IF EXISTS stock_picking_int_seq;
DROP SEQUENCE IF EXISTS stock_picking_out_seq;
DROP SEQUENCE IF EXISTS stock_picking_in_seq;
