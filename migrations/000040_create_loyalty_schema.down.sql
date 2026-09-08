-- 000040_create_loyalty_schema.down.sql
-- Phase 23: Loyalty & Rewards rollback

ALTER TABLE sale_order_lines DROP CONSTRAINT IF EXISTS fk_sale_order_lines_coupon;
ALTER TABLE sale_order_lines DROP CONSTRAINT IF EXISTS fk_sale_order_lines_reward;

ALTER TABLE sale_order_lines
    DROP COLUMN IF EXISTS reward_id,
    DROP COLUMN IF EXISTS coupon_id,
    DROP COLUMN IF EXISTS reward_identifier_code,
    DROP COLUMN IF EXISTS points_cost,
    DROP COLUMN IF EXISTS is_reward_line;

ALTER TABLE sale_orders
    DROP COLUMN IF EXISTS applied_coupon_ids,
    DROP COLUMN IF EXISTS code_enabled_rule_ids;

DROP TABLE IF EXISTS sale_order_coupon_points;
DROP TABLE IF EXISTS loyalty_mails;
DROP TABLE IF EXISTS loyalty_card_history;
DROP TABLE IF EXISTS loyalty_cards;
DROP TABLE IF EXISTS loyalty_rewards;
DROP TABLE IF EXISTS loyalty_rules;
DROP TABLE IF EXISTS loyalty_program_pricelists;
DROP TABLE IF EXISTS loyalty_programs;