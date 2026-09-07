ALTER TABLE stock_move_lines
DROP CONSTRAINT IF EXISTS stock_move_lines_tracking_check;

ALTER TABLE stock_move_lines
DROP COLUMN IF EXISTS tracking_mode;