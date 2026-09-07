-- Align legacy MRP and landed-cost literals with Odoo 19.
UPDATE stock_landed_cost_lines
SET split_method = 'by_current_cost_price'
WHERE split_method = 'by_current_cost';

UPDATE mrp_workorders
SET state = 'ready'
WHERE state = 'pending';

UPDATE mrp_productions
SET reservation_state = 'waiting'
WHERE reservation_state = 'partial';
