-- 000041_seed_loyalty_acl.up.sql
-- Phase 23: Seed ACL permissions for Loyalty & Rewards models.
-- Group 1 (internal user): CRUD except delete. Group 2 (manager/admin): full CRUD.

INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'loyalty.program', true, true, true, false),
    (2::BIGINT, 'loyalty.program', true, true, true, true),
    (1::BIGINT, 'loyalty.rule', true, true, true, false),
    (2::BIGINT, 'loyalty.rule', true, true, true, true),
    (1::BIGINT, 'loyalty.reward', true, true, true, false),
    (2::BIGINT, 'loyalty.reward', true, true, true, true),
    (1::BIGINT, 'loyalty.card', true, true, true, false),
    (2::BIGINT, 'loyalty.card', true, true, true, true),
    (1::BIGINT, 'loyalty.card.history', true, true, true, false),
    (2::BIGINT, 'loyalty.card.history', true, true, true, true),
    (1::BIGINT, 'loyalty.mail', true, true, true, false),
    (2::BIGINT, 'loyalty.mail', true, true, true, true),
    (1::BIGINT, 'sale.order.coupon.points', true, true, true, false),
    (2::BIGINT, 'sale.order.coupon.points', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;