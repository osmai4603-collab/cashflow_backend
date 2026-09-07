-- 000022_sale_purchase_stock_integration.up.sql

-- 1. Create Stock Procurement Groups
CREATE TABLE IF NOT EXISTS stock_procurement_groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    company_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_stock_procurement_groups_updated_at
    BEFORE UPDATE ON stock_procurement_groups
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 2. Add columns to sale_orders
ALTER TABLE sale_orders
ADD COLUMN IF NOT EXISTS delivery_status VARCHAR(20) NOT NULL DEFAULT 'nothing',
ADD COLUMN IF NOT EXISTS procurement_group_id BIGINT REFERENCES stock_procurement_groups(id) ON DELETE SET NULL;

-- 3. Add columns to sale_order_lines
ALTER TABLE sale_order_lines
ADD COLUMN IF NOT EXISTS route_id BIGINT;

-- 4. Add columns to purchase_orders
ALTER TABLE purchase_orders
ADD COLUMN IF NOT EXISTS receipt_status VARCHAR(20) NOT NULL DEFAULT 'nothing',
ADD COLUMN IF NOT EXISTS procurement_group_id BIGINT REFERENCES stock_procurement_groups(id) ON DELETE SET NULL;

-- 5. Add columns to stock_pickings
ALTER TABLE stock_pickings
ADD COLUMN IF NOT EXISTS procurement_group_id BIGINT REFERENCES stock_procurement_groups(id) ON DELETE SET NULL;

-- 6. Add columns to stock_moves
ALTER TABLE stock_moves
ADD COLUMN IF NOT EXISTS procurement_group_id BIGINT REFERENCES stock_procurement_groups(id) ON DELETE SET NULL;
