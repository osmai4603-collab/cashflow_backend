-- 000040_create_loyalty_schema.up.sql
-- Phase 23: Loyalty & Rewards (loyalty.* / sale_loyalty in Odoo 19.0)

-- 1. Loyalty Programs (loyalty.program)
CREATE TABLE IF NOT EXISTS loyalty_programs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    sequence INT NOT NULL DEFAULT 10,
    company_id BIGINT,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    program_type VARCHAR(20) NOT NULL DEFAULT 'promotion',
    date_from TIMESTAMPTZ,
    date_to TIMESTAMPTZ,
    limit_usage BOOLEAN NOT NULL DEFAULT false,
    max_usage INT NOT NULL DEFAULT 0,
    applies_on VARCHAR(10) NOT NULL DEFAULT 'current',
    trigger VARCHAR(10) NOT NULL DEFAULT 'auto',
    portal_visible BOOLEAN NOT NULL DEFAULT false,
    portal_point_name VARCHAR(64) NOT NULL DEFAULT 'Points',
    is_nominative BOOLEAN NOT NULL DEFAULT false,
    is_payment_program BOOLEAN NOT NULL DEFAULT false,
    sale_ok BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_loyalty_programs_active ON loyalty_programs(active);
CREATE INDEX IF NOT EXISTS idx_loyalty_programs_type ON loyalty_programs(program_type);
CREATE INDEX IF NOT EXISTS idx_loyalty_programs_company ON loyalty_programs(company_id);

CREATE TRIGGER trg_loyalty_programs_updated_at
    BEFORE UPDATE ON loyalty_programs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Program Pricelists junction (many2many)
CREATE TABLE IF NOT EXISTS loyalty_program_pricelists (
    program_id BIGINT NOT NULL REFERENCES loyalty_programs(id) ON DELETE CASCADE,
    pricelist_id BIGINT NOT NULL REFERENCES product_pricelists(id) ON DELETE CASCADE,
    PRIMARY KEY (program_id, pricelist_id)
);

-- 2. Loyalty Rules (loyalty.rule)
CREATE TABLE IF NOT EXISTS loyalty_rules (
    id BIGSERIAL PRIMARY KEY,
    active BOOLEAN NOT NULL DEFAULT true,
    program_id BIGINT NOT NULL REFERENCES loyalty_programs(id) ON DELETE CASCADE,
    company_id BIGINT,
    product_ids BIGINT[] DEFAULT '{}',
    product_category_id BIGINT,
    product_tag_id BIGINT,
    product_domain TEXT,
    reward_point_amount NUMERIC(15,4) NOT NULL DEFAULT 1.0000,
    reward_point_split BOOLEAN NOT NULL DEFAULT false,
    reward_point_mode VARCHAR(8) NOT NULL DEFAULT 'order',
    minimum_qty INT NOT NULL DEFAULT 1,
    minimum_amount NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    minimum_amount_tax_mode VARCHAR(4) NOT NULL DEFAULT 'incl',
    mode VARCHAR(10) NOT NULL DEFAULT 'auto',
    code VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_loyalty_rules_program ON loyalty_rules(program_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_loyalty_rules_code ON loyalty_rules(code) WHERE code IS NOT NULL;

CREATE TRIGGER trg_loyalty_rules_updated_at
    BEFORE UPDATE ON loyalty_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 3. Loyalty Rewards (loyalty.reward)
CREATE TABLE IF NOT EXISTS loyalty_rewards (
    id BIGSERIAL PRIMARY KEY,
    active BOOLEAN NOT NULL DEFAULT true,
    program_id BIGINT NOT NULL REFERENCES loyalty_programs(id) ON DELETE CASCADE,
    description VARCHAR(255),
    reward_type VARCHAR(10) NOT NULL DEFAULT 'discount',
    discount NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    discount_mode VARCHAR(10) NOT NULL DEFAULT 'percent',
    discount_applicability VARCHAR(10) NOT NULL DEFAULT 'order',
    discount_product_ids BIGINT[] DEFAULT '{}',
    discount_product_category_id BIGINT,
    discount_product_tag_id BIGINT,
    discount_max_amount NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    discount_line_product_id BIGINT REFERENCES product_templates(id) ON DELETE RESTRICT,
    reward_product_id BIGINT REFERENCES product_templates(id) ON DELETE SET NULL,
    reward_product_qty INT NOT NULL DEFAULT 1,
    reward_product_uom_id BIGINT,
    required_points NUMERIC(15,4) NOT NULL DEFAULT 1.0000,
    clear_wallet BOOLEAN NOT NULL DEFAULT false,
    product_domain TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_loyalty_rewards_program ON loyalty_rewards(program_id);

CREATE TRIGGER trg_loyalty_rewards_updated_at
    BEFORE UPDATE ON loyalty_rewards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 4. Loyalty Cards (loyalty.card)
CREATE TABLE IF NOT EXISTS loyalty_cards (
    id BIGSERIAL PRIMARY KEY,
    program_id BIGINT NOT NULL REFERENCES loyalty_programs(id) ON DELETE RESTRICT,
    company_id BIGINT,
    partner_id BIGINT,
    points NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    code VARCHAR(64) NOT NULL UNIQUE,
    expiration_date TIMESTAMPTZ,
    use_count INT NOT NULL DEFAULT 0,
    order_id BIGINT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT,
    updated_by BIGINT
);

CREATE INDEX IF NOT EXISTS idx_loyalty_cards_program ON loyalty_cards(program_id);
CREATE INDEX IF NOT EXISTS idx_loyalty_cards_partner ON loyalty_cards(partner_id);
CREATE INDEX IF NOT EXISTS idx_loyalty_cards_active ON loyalty_cards(active);

CREATE TRIGGER trg_loyalty_cards_updated_at
    BEFORE UPDATE ON loyalty_cards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 5. Loyalty History (loyalty.history)
CREATE TABLE IF NOT EXISTS loyalty_card_history (
    id BIGSERIAL PRIMARY KEY,
    card_id BIGINT NOT NULL REFERENCES loyalty_cards(id) ON DELETE CASCADE,
    company_id BIGINT,
    description TEXT NOT NULL,
    issued NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    used NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    order_model VARCHAR(32),
    order_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_loyalty_card_history_card ON loyalty_card_history(card_id);

-- 6. Loyalty Mails (loyalty.mail — config only)
CREATE TABLE IF NOT EXISTS loyalty_mails (
    id BIGSERIAL PRIMARY KEY,
    active BOOLEAN NOT NULL DEFAULT true,
    program_id BIGINT NOT NULL REFERENCES loyalty_programs(id) ON DELETE CASCADE,
    trigger VARCHAR(16) NOT NULL DEFAULT 'create',
    points NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_loyalty_mails_program ON loyalty_mails(program_id);

CREATE TRIGGER trg_loyalty_mails_updated_at
    BEFORE UPDATE ON loyalty_mails
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 7. Sale Order Coupon Points (sale.order.coupon.points)
CREATE TABLE IF NOT EXISTS sale_order_coupon_points (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES sale_orders(id) ON DELETE CASCADE,
    coupon_id BIGINT NOT NULL REFERENCES loyalty_cards(id) ON DELETE CASCADE,
    points NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (order_id, coupon_id)
);

CREATE INDEX IF NOT EXISTS idx_sale_order_coupon_points_coupon ON sale_order_coupon_points(coupon_id);

-- 8. Sale order integration columns
ALTER TABLE sale_orders
    ADD COLUMN IF NOT EXISTS applied_coupon_ids BIGINT[] DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS code_enabled_rule_ids BIGINT[] DEFAULT '{}';

ALTER TABLE sale_order_lines
    ADD COLUMN IF NOT EXISTS reward_id BIGINT,
    ADD COLUMN IF NOT EXISTS coupon_id BIGINT,
    ADD COLUMN IF NOT EXISTS reward_identifier_code VARCHAR(64),
    ADD COLUMN IF NOT EXISTS points_cost NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    ADD COLUMN IF NOT EXISTS is_reward_line BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE sale_order_lines
    ADD CONSTRAINT fk_sale_order_lines_reward FOREIGN KEY (reward_id) REFERENCES loyalty_rewards(id) ON DELETE RESTRICT,
    ADD CONSTRAINT fk_sale_order_lines_coupon FOREIGN KEY (coupon_id) REFERENCES loyalty_cards(id) ON DELETE RESTRICT;