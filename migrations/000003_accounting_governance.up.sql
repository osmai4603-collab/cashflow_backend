ALTER TABLE account_moves
    ADD COLUMN IF NOT EXISTS created_by UUID;
