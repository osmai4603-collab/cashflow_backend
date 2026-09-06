-- 000002_create_partners_table.down.sql
-- Drop Res Partners table

DROP TRIGGER IF EXISTS trg_partners_updated_at ON res_partners;
DROP TABLE IF EXISTS res_partners CASCADE;
