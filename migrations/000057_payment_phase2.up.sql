ALTER TABLE payment_providers ADD COLUMN IF NOT EXISTS module_state VARCHAR(32) DEFAULT 'installed';
ALTER TABLE payment_providers ADD COLUMN IF NOT EXISTS inline_form BOOLEAN DEFAULT FALSE;
ALTER TABLE payment_providers ADD COLUMN IF NOT EXISTS support_refund VARCHAR(16) DEFAULT 'none';
ALTER TABLE payment_providers ADD COLUMN IF NOT EXISTS support_tokenize BOOLEAN DEFAULT FALSE;
ALTER TABLE payment_providers ADD COLUMN IF NOT EXISTS support_authorize BOOLEAN DEFAULT FALSE;
ALTER TABLE payment_providers ADD COLUMN IF NOT EXISTS webhook_secret TEXT;
ALTER TABLE payment_providers ADD COLUMN IF NOT EXISTS allow_tokenize BOOLEAN DEFAULT FALSE;
ALTER TABLE payment_providers ADD COLUMN IF NOT EXISTS capture_manually BOOLEAN DEFAULT FALSE;
ALTER TABLE payment_providers ADD COLUMN IF NOT EXISTS journal_id BIGINT;

CREATE TABLE payment_provider_configs (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES payment_providers(id) ON DELETE CASCADE,
    key VARCHAR(128) NOT NULL,
    value TEXT NOT NULL,
    is_secret BOOLEAN NOT NULL DEFAULT FALSE,
    environment VARCHAR(16) NOT NULL DEFAULT 'test',
    company_id BIGINT NOT NULL,
    UNIQUE(provider_id, key, environment)
);
CREATE TABLE payment_tokens (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES payment_providers(id),
    partner_id BIGINT NOT NULL,
    provider_ref VARCHAR(256) NOT NULL,
    display_name VARCHAR(128) NOT NULL,
    payment_details VARCHAR(64),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    company_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_payment_tokens_partner ON payment_tokens(partner_id, active);
CREATE TABLE payment_refunds (
    id BIGSERIAL PRIMARY KEY,
    original_transaction_id BIGINT NOT NULL REFERENCES payment_transactions(id),
    refund_transaction_id BIGINT REFERENCES payment_transactions(id),
    amount NUMERIC(15,4) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    reason VARCHAR(64) NOT NULL,
    provider_reference VARCHAR(256),
    state VARCHAR(32) NOT NULL DEFAULT 'pending',
    company_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE payment_webhook_logs (
    id BIGSERIAL PRIMARY KEY,
    provider_code VARCHAR(64) NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    payload JSONB NOT NULL,
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    process_error TEXT,
    idempotency_key VARCHAR(256) NOT NULL UNIQUE,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);