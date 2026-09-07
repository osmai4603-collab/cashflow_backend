INSERT INTO res_group_permissions (group_id, model, can_read, can_create, can_update, can_delete)
SELECT defaults.group_id, defaults.model, defaults.can_read, defaults.can_create, defaults.can_update, defaults.can_delete
FROM (VALUES
    (1::BIGINT, 'sale.order', true, true, true, false), (2::BIGINT, 'sale.order', true, true, true, true),
    (1::BIGINT, 'purchase.order', true, true, true, false), (2::BIGINT, 'purchase.order', true, true, true, true),
    (1::BIGINT, 'account.account', true, false, false, false), (2::BIGINT, 'account.account', true, true, true, true),
    (1::BIGINT, 'account.journal', true, false, false, false), (2::BIGINT, 'account.journal', true, true, true, true),
    (1::BIGINT, 'account.tax', true, false, false, false), (2::BIGINT, 'account.tax', true, true, true, true),
    (1::BIGINT, 'account.payment_term', true, false, false, false), (2::BIGINT, 'account.payment_term', true, true, true, true),
    (1::BIGINT, 'account.move', true, true, true, false), (2::BIGINT, 'account.move', true, true, true, true),
    (1::BIGINT, 'account.report', true, false, false, false), (2::BIGINT, 'account.report', true, true, true, true),
    (1::BIGINT, 'stock.warehouse', true, false, false, false), (2::BIGINT, 'stock.warehouse', true, true, true, true),
    (1::BIGINT, 'stock.location', true, false, false, false), (2::BIGINT, 'stock.location', true, true, true, true),
    (1::BIGINT, 'stock.picking', true, true, true, false), (2::BIGINT, 'stock.picking', true, true, true, true),
    (1::BIGINT, 'stock.quant', true, true, true, false), (2::BIGINT, 'stock.quant', true, true, true, true),
    (1::BIGINT, 'stock.move', true, false, false, false), (2::BIGINT, 'stock.move', true, true, true, true),
    (1::BIGINT, 'account.payment', true, true, true, false), (2::BIGINT, 'account.payment', true, true, true, true),
    (1::BIGINT, 'hr.department', true, false, false, false), (2::BIGINT, 'hr.department', true, true, true, true),
    (1::BIGINT, 'hr.job', true, false, false, false), (2::BIGINT, 'hr.job', true, true, true, true),
    (1::BIGINT, 'hr.employee', true, false, false, false), (2::BIGINT, 'hr.employee', true, true, true, true),
    (1::BIGINT, 'hr.leave_allocation', true, true, true, false), (2::BIGINT, 'hr.leave_allocation', true, true, true, true),
    (1::BIGINT, 'hr.leave_request', true, true, true, false), (2::BIGINT, 'hr.leave_request', true, true, true, true),
    (1::BIGINT, 'crm.lead', true, true, true, false), (2::BIGINT, 'crm.lead', true, true, true, true),
    (1::BIGINT, 'crm.stage', true, false, false, false), (2::BIGINT, 'crm.stage', true, true, true, true),
    (1::BIGINT, 'crm.lost_reason', true, false, false, false), (2::BIGINT, 'crm.lost_reason', true, true, true, true),
    (1::BIGINT, 'crm.tag', true, true, true, false), (2::BIGINT, 'crm.tag', true, true, true, true)
) AS defaults(group_id, model, can_read, can_create, can_update, can_delete)
WHERE EXISTS (SELECT 1 FROM res_groups WHERE id = defaults.group_id)
ON CONFLICT (group_id, model) DO NOTHING;
