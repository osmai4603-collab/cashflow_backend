-- migrations/000070_seed_bank_statement_acl.up.sql
-- Seed ACL model-level permissions for account.bank.statement.
-- Group 1 (User): read/write, no delete. Group 2 (Administrator): full access.

INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'account.bank.statement', true, true, true, false),
    (2::BIGINT, 'account.bank.statement', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;