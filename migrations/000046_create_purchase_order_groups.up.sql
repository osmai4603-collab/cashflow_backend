CREATE TABLE IF NOT EXISTS purchase_order_groups (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT REFERENCES res_companies(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS purchase_order_group_members (
    group_id BIGINT NOT NULL REFERENCES purchase_order_groups(id) ON DELETE CASCADE,
    order_id BIGINT NOT NULL UNIQUE REFERENCES purchase_orders(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, order_id)
);

CREATE INDEX IF NOT EXISTS idx_purchase_order_group_members_group
    ON purchase_order_group_members(group_id);

CREATE TRIGGER trg_purchase_order_groups_updated_at
    BEFORE UPDATE ON purchase_order_groups
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();