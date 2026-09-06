-- 000008_create_crm_schema.down.sql
-- Revert CRM schema

DROP TRIGGER IF EXISTS trg_crm_leads_updated_at ON crm_leads;
DROP TRIGGER IF EXISTS trg_crm_tags_updated_at ON crm_tags;
DROP TRIGGER IF EXISTS trg_crm_lost_reasons_updated_at ON crm_lost_reasons;
DROP TRIGGER IF EXISTS trg_crm_stages_updated_at ON crm_stages;

DROP TABLE IF EXISTS crm_lead_tags CASCADE;
DROP TABLE IF EXISTS crm_leads CASCADE;
DROP TABLE IF EXISTS crm_tags CASCADE;
DROP TABLE IF EXISTS crm_lost_reasons CASCADE;
DROP TABLE IF EXISTS crm_stages CASCADE;
