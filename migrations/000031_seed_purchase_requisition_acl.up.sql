INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'purchase.requisition', true, true, true, false),
    (2::BIGINT, 'purchase.requisition', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;