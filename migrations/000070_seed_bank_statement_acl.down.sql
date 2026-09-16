-- migrations/000070_seed_bank_statement_acl.down.sql
DELETE FROM res_group_permissions
WHERE model = 'account.bank.statement';