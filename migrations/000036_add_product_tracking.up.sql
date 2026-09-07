ALTER TABLE product_templates
ADD COLUMN IF NOT EXISTS tracking VARCHAR(16) NOT NULL DEFAULT 'none';

ALTER TABLE product_templates
ADD CONSTRAINT product_templates_tracking_check
CHECK (tracking IN ('none', 'lot', 'serial'));