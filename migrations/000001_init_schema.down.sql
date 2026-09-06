-- 000001_init_schema.down.sql
-- Rollback base database configuration

DROP FUNCTION IF EXISTS update_updated_at_column();
DROP EXTENSION IF EXISTS "citext";
DROP EXTENSION IF EXISTS "uuid-ossp";
