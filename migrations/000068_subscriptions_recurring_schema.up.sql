-- migrations/000068_subscriptions_recurring_schema.up.sql

CREATE TABLE subscription_plans (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    period          VARCHAR(16) NOT NULL,
    period_interval INT NOT NULL DEFAULT 1,
    price           NUMERIC(15,4) NOT NULL DEFAULT 0,
    currency        VARCHAR(3) NOT NULL DEFAULT 'SAR',
    product_id      BIGINT NOT NULL REFERENCES product_templates(id),
    company_id      BIGINT NOT NULL REFERENCES res_companies(id),
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sale_subscriptions (
    id                  BIGSERIAL PRIMARY KEY,
    code                VARCHAR(64) NOT NULL UNIQUE,
    partner_id          BIGINT NOT NULL REFERENCES res_partners(id),
    plan_id             BIGINT NOT NULL REFERENCES subscription_plans(id),
    state               VARCHAR(32) NOT NULL DEFAULT 'draft',
    start_date          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    next_billing_date   TIMESTAMPTZ NOT NULL,
    end_date            TIMESTAMPTZ,
    recurring_amount    NUMERIC(15,4) NOT NULL DEFAULT 0,
    payment_token_id    BIGINT REFERENCES payment_tokens(id),
    failed_charge_count INT NOT NULL DEFAULT 0,
    company_id          BIGINT NOT NULL REFERENCES res_companies(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_subscriptions_next_bill ON sale_subscriptions(state, next_billing_date);
