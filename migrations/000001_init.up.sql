-- Initial schema migration
-- Cashflow Backend System
CREATE TABLE IF NOT EXISTS schema_initialization (
    id SERIAL PRIMARY KEY,
    initialized_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
