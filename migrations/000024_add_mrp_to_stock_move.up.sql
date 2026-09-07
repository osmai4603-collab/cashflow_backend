ALTER TABLE stock_moves ADD COLUMN production_id BIGINT REFERENCES mrp_productions(id) ON DELETE CASCADE;
ALTER TABLE stock_moves ADD COLUMN production_finished_id BIGINT REFERENCES mrp_productions(id) ON DELETE CASCADE;

CREATE INDEX idx_stock_move_production ON stock_moves(production_id);
CREATE INDEX idx_stock_move_production_fin ON stock_moves(production_finished_id);
