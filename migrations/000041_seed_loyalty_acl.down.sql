-- 000041_seed_loyalty_acl.down.sql
-- Phase 23: Remove Loyalty & Rewards ACL permissions.

DELETE FROM res_group_permissions
WHERE model IN (
    'loyalty.program',
    'loyalty.rule',
    'loyalty.reward',
    'loyalty.card',
    'loyalty.card.history',
    'loyalty.mail',
    'sale.order.coupon.points'
);