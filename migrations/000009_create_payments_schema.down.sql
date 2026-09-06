-- 000009_create_payments_schema.down.sql
-- Revert Payments schema

DROP TRIGGER IF EXISTS trg_account_payments_updated_at ON account_payments;

DROP TABLE IF EXISTS account_payment_reconciliations CASCADE;
DROP TABLE IF EXISTS account_payments CASCADE;
DROP SEQUENCE IF EXISTS account_payment_seq;
