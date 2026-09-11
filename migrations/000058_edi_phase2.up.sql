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