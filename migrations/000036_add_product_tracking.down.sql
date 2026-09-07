ALTER TABLE product_templates
DROP CONSTRAINT IF EXISTS product_templates_tracking_check;

ALTER TABLE product_templates
DROP COLUMN IF EXISTS tracking;