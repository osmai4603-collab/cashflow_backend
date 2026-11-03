-- Run with psql as a PostgreSQL administrator, connected to the postgres database.
DO $bootstrap$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_catalog.pg_roles
        WHERE rolname = 'cashflow'
    ) THEN
        CREATE ROLE cashflow LOGIN;
    END IF;
END
$bootstrap$;

SELECT format('CREATE DATABASE %I OWNER %I', 'train_db', 'cashflow')
WHERE NOT EXISTS (
    SELECT 1
    FROM pg_catalog.pg_database
    WHERE datname = 'train_db'
)
\gexec

GRANT CONNECT, TEMPORARY ON DATABASE train_db TO cashflow;

\connect train_db

GRANT USAGE, CREATE ON SCHEMA public TO cashflow;
