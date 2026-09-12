CREATE TABLE IF NOT EXISTS edi_documents (
    id BIGSERIAL PRIMARY KEY,
    move_id BIGINT NOT NULL,
    format VARCHAR(32) NOT NULL DEFAULT 'ubl_2_1',
    transaction_type VARCHAR(16) NOT NULL DEFAULT 'standard',
    state VARCHAR(32) NOT NULL DEFAULT 'to_send',
    xml_content BYTEA,
    hash VARCHAR(128),
    qr_code VARCHAR(512),
    error_msg TEXT,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS edi_certificates (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    cert_content BYTEA,
    private_key BYTEA,
    public_key BYTEA,
    csr TEXT,
    csid VARCHAR(512),
    secret TEXT,
    company_id BIGINT NOT NULL,
    is_production BOOLEAN NOT NULL DEFAULT FALSE,
    expiration_date TIMESTAMPTZ,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE edi_documents ADD COLUMN IF NOT EXISTS uuid VARCHAR(36);
ALTER TABLE edi_documents ADD COLUMN IF NOT EXISTS previous_hash VARCHAR(128);
ALTER TABLE edi_documents ADD COLUMN IF NOT EXISTS signature TEXT;
ALTER TABLE edi_documents ADD COLUMN IF NOT EXISTS zatca_status VARCHAR(32);
ALTER TABLE edi_documents ADD COLUMN IF NOT EXISTS zatca_request_id VARCHAR(128);
ALTER TABLE edi_documents ADD COLUMN IF NOT EXISTS cleared_xml TEXT;
ALTER TABLE edi_certificates ADD COLUMN IF NOT EXISTS compliance_csid VARCHAR(512);
ALTER TABLE edi_certificates ADD COLUMN IF NOT EXISTS production_csid VARCHAR(512);
ALTER TABLE edi_certificates ADD COLUMN IF NOT EXISTS compliance_request_id VARCHAR(128);
ALTER TABLE edi_certificates ADD COLUMN IF NOT EXISTS onboarding_status VARCHAR(32) DEFAULT 'pending';
CREATE TABLE edi_zatca_submissions (
    id BIGSERIAL PRIMARY KEY,
    edi_document_id BIGINT NOT NULL REFERENCES edi_documents(id) ON DELETE CASCADE,
    submission_type VARCHAR(16) NOT NULL,
    request_body TEXT NOT NULL,
    response_body TEXT,
    response_status VARCHAR(32),
    warnings JSONB,
    errors JSONB,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    company_id BIGINT NOT NULL
);