-- 000053_create_payment_transactions.up.sql
-- Phase 1 Glue: Separate Payment from PaymentTransaction (payment.transaction in Odoo 19)

CREATE TABLE IF NOT EXISTS payment_providers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'disabled',
    active BOOLEAN NOT NULL DEFAULT true,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_payment_provider_code_company UNIQUE (code, company_id)
);

CREATE TABLE IF NOT EXISTS payment_transactions (
    id BIGSERIAL PRIMARY KEY,
    reference VARCHAR(255) NOT NULL,
    amount NUMERIC(15, 4) NOT NULL,
    currency VARCHAR(10) NOT NULL,
    provider_id BIGINT NOT NULL REFERENCES payment_providers(id) ON DELETE RESTRICT,
    partner_id BIGINT NOT NULL REFERENCES res_partners(id) ON DELETE RESTRICT,
    state VARCHAR(20) NOT NULL DEFAULT 'draft',
    provider_reference VARCHAR(255),
    sale_order_id BIGINT REFERENCES sale_orders(id) ON DELETE SET NULL,
    invoice_id BIGINT REFERENCES account_moves(id) ON DELETE SET NULL,
    payment_id BIGINT REFERENCES account_payments(id) ON DELETE SET NULL,
    idempotency_key VARCHAR(255),
    return_url TEXT,
    webhook_received BOOLEAN NOT NULL DEFAULT false,
    last_error TEXT,
    metadata JSONB,
    company_id BIGINT NOT NULL REFERENCES res_companies(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_payment_tx_reference UNIQUE (reference)
);

CREATE INDEX IF NOT EXISTS idx_payment_transactions_provider_ref ON payment_transactions(provider_id, provider_reference);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_sale_order ON payment_transactions(sale_order_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_invoice ON payment_transactions(invoice_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_payment ON payment_transactions(payment_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_idempotency ON payment_transactions(provider_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
