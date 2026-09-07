CREATE TABLE IF NOT EXISTS stock_move_lines (
    id BIGSERIAL PRIMARY KEY,
    move_id BIGINT NOT NULL REFERENCES stock_moves(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    product_uom BIGINT REFERENCES uom_uoms(id) ON DELETE SET NULL,
    lot_id BIGINT,
    package_id BIGINT,
    owner_id BIGINT,
    location_id BIGINT NOT NULL REFERENCES stock_locations(id) ON DELETE RESTRICT,
    location_dest_id BIGINT NOT NULL REFERENCES stock_locations(id) ON DELETE RESTRICT,
    reserved_quantity NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    quantity_done NUMERIC(15, 4) NOT NULL DEFAULT 0.0000,
    date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT stock_move_lines_nonnegative CHECK (reserved_quantity >= 0 AND quantity_done >= 0)
);

CREATE INDEX IF NOT EXISTS idx_stock_move_lines_move ON stock_move_lines(move_id);
CREATE INDEX IF NOT EXISTS idx_stock_move_lines_product ON stock_move_lines(product_id);