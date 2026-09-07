-- Model-level ACL permissions.
CREATE TABLE IF NOT EXISTS res_group_permissions (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES res_groups(id) ON DELETE CASCADE,
    model VARCHAR(100) NOT NULL,
    can_read BOOLEAN NOT NULL DEFAULT false,
    can_create BOOLEAN NOT NULL DEFAULT false,
    can_update BOOLEAN NOT NULL DEFAULT false,
    can_delete BOOLEAN NOT NULL DEFAULT false,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_res_group_permissions_group_model UNIQUE (group_id, model)
);

CREATE INDEX IF NOT EXISTS idx_res_group_permissions_model
    ON res_group_permissions(model);
CREATE INDEX IF NOT EXISTS idx_res_group_permissions_group
    ON res_group_permissions(group_id);

CREATE TRIGGER trg_res_group_permissions_updated_at
    BEFORE UPDATE ON res_group_permissions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

INSERT INTO res_group_permissions (group_id, model, can_read)
SELECT 2, 'system.api', true
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = 2)
ON CONFLICT (group_id, model) DO NOTHING;

INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT group_id, model, can_read, can_create, can_update, can_delete
FROM (VALUES
    (1::BIGINT, 'res.company', true, false, false, false),
    (2::BIGINT, 'res.company', true, true, true, true),
    (1::BIGINT, 'product.template', true, true, true, false),
    (2::BIGINT, 'product.template', true, true, true, true),
    (1::BIGINT, 'product.product', true, true, true, false),
    (2::BIGINT, 'product.product', true, true, true, true),
    (1::BIGINT, 'product.category', true, true, true, false),
    (2::BIGINT, 'product.category', true, true, true, true),
    (1::BIGINT, 'uom.uom', true, false, false, false),
    (2::BIGINT, 'uom.uom', true, true, true, true),
    (1::BIGINT, 'product.pricelist', true, true, true, false),
    (2::BIGINT, 'product.pricelist', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;

INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT group_id, 'res.partner', can_read, can_create, can_update, can_delete
FROM (VALUES
    (1::BIGINT, true, true, true, false),
    (2::BIGINT, true, true, true, true)
) AS defaults(group_id, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;
