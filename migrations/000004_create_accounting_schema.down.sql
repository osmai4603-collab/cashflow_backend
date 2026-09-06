-- 000004_create_accounting_schema.down.sql
-- Rollback accounting schema

DROP TABLE IF EXISTS account_move_lines CASCADE;
DROP TABLE IF EXISTS account_moves CASCADE;
DROP TABLE IF EXISTS account_payment_term_lines CASCADE;
DROP TABLE IF EXISTS account_payment_terms CASCADE;
DROP TABLE IF EXISTS account_taxes CASCADE;
DROP TABLE IF EXISTS account_journals CASCADE;
DROP TABLE IF EXISTS account_accounts CASCADE;
