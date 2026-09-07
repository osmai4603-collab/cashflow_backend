DELETE FROM res_group_permissions
WHERE model IN (
    'sale.order', 'purchase.order', 'account.account', 'account.journal', 'account.tax',
    'account.payment_term', 'account.move', 'account.report', 'stock.warehouse', 'stock.location',
    'stock.picking', 'stock.quant', 'stock.move', 'account.payment', 'hr.department', 'hr.job',
    'hr.employee', 'hr.leave_allocation', 'hr.leave_request', 'crm.lead', 'crm.stage',
    'crm.lost_reason', 'crm.tag'
);
