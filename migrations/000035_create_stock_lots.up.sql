CREATE TABLE IF NOT EXISTS stock_lots (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES product_templates(id) ON DELETE RESTRICT,
    name VARCHAR(128) NOT NULL,
    tracking_mode VARCHAR(16) NOT NULL DEFAULT 'lot',
    company_id BIGINT,
    expiration_at TIMESTAMPTZ,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT stock_lots_tracking_mode CHECK (tracking_mode IN ('lot', 'serial')),
    CONSTRAINT stock_lots_product_name_unique UNIQUE (product_id, name)
);

CREATE INDEX IF NOT EXISTS idx_stock_lots_product ON stock_lots(product_id);