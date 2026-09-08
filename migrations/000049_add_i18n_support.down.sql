-- 000049_add_i18n_support.down.sql

DROP TABLE IF EXISTS ir_translation;

ALTER TABLE product_categories ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';

ALTER TABLE accounts ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';

ALTER TABLE sale_order_lines ALTER COLUMN name TYPE TEXT USING name->>'en_US';

ALTER TABLE stock_locations ALTER COLUMN complete_name TYPE VARCHAR(500) USING complete_name->>'en_US';
ALTER TABLE stock_locations ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';

ALTER TABLE warehouses ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';

ALTER TABLE product_templates ALTER COLUMN name TYPE VARCHAR(255) USING name->>'en_US';

DROP TABLE IF EXISTS res_lang;
