-- 000021_create_reorder_landed_cost_schema.down.sql

DROP TABLE IF EXISTS stock_valuation_adjustment_lines;
DROP TABLE IF EXISTS stock_landed_cost_lines;
DROP TABLE IF EXISTS stock_landed_costs;
DROP TABLE IF EXISTS stock_orderpoints;

ALTER TABLE product_templates
    DROP COLUMN IF EXISTS landed_cost_ok,
    DROP COLUMN IF EXISTS split_method_landed_cost;

ALTER TABLE companies
    DROP COLUMN IF EXISTS lc_journal_id;

DROP SEQUENCE IF EXISTS stock_orderpoint_sequence;
DROP SEQUENCE IF EXISTS stock_landed_cost_sequence;

DELETE FROM res_group_permissions WHERE model IN ('stock.orderpoint', 'stock.landed.cost');