ALTER TABLE stock_move_lines
ADD COLUMN IF NOT EXISTS tracking_mode VARCHAR(16) NOT NULL DEFAULT 'none';

ALTER TABLE stock_move_lines
ADD CONSTRAINT stock_move_lines_tracking_check
CHECK (tracking_mode IN ('none', 'lot', 'serial'));