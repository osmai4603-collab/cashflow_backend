ALTER TABLE stock_picking DROP COLUMN IF EXISTS carrier_id;
ALTER TABLE stock_picking DROP COLUMN IF EXISTS carrier_tracking_ref;
ALTER TABLE stock_picking DROP COLUMN IF EXISTS weight;
ALTER TABLE stock_picking DROP COLUMN IF EXISTS shipping_weight;
ALTER TABLE stock_picking DROP COLUMN IF EXISTS number_of_packages;

ALTER TABLE sale_order_line DROP COLUMN IF EXISTS is_delivery;

ALTER TABLE sale_order DROP COLUMN IF EXISTS carrier_id;
ALTER TABLE sale_order DROP COLUMN IF EXISTS shipping_weight;
ALTER TABLE sale_order DROP COLUMN IF EXISTS delivery_message;
ALTER TABLE sale_order DROP COLUMN IF EXISTS recompute_delivery_price;

DROP TABLE IF EXISTS delivery_carrier_zip_prefix_rel;
DROP TABLE IF EXISTS delivery_carrier_state_rel;
DROP TABLE IF EXISTS delivery_carrier_country_rel;
DROP TABLE IF EXISTS delivery_zip_prefix;
DROP TABLE IF EXISTS delivery_price_rule;
DROP TABLE IF EXISTS delivery_carrier;
