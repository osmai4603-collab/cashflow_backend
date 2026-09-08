-- Delivery Carrier Table
CREATE TABLE IF NOT EXISTS delivery_carrier (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    sequence INTEGER DEFAULT 10,
    delivery_type VARCHAR(50) NOT NULL, -- 'fixed', 'base_on_rule'
    integration_level VARCHAR(50) DEFAULT 'rate',
    invoice_policy VARCHAR(50) DEFAULT 'estimated',
    product_id INTEGER NOT NULL REFERENCES product_template(id),
    fixed_price DECIMAL(19,4) DEFAULT 0,
    margin DECIMAL(19,4) DEFAULT 0,
    fixed_margin DECIMAL(19,4) DEFAULT 0,
    free_over BOOLEAN DEFAULT FALSE,
    amount DECIMAL(19,4) DEFAULT 0,
    max_weight DECIMAL(19,4),
    max_volume DECIMAL(19,4),
    company_id INTEGER REFERENCES res_company(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Delivery Price Rules Table
CREATE TABLE IF NOT EXISTS delivery_price_rule (
    id SERIAL PRIMARY KEY,
    carrier_id INTEGER NOT NULL REFERENCES delivery_carrier(id) ON DELETE CASCADE,
    sequence INTEGER DEFAULT 10,
    variable VARCHAR(50) NOT NULL, -- 'weight', 'volume', 'wv', 'price', 'quantity'
    operator VARCHAR(10) NOT NULL, -- '==', '<=', '<', '>=', '>'
    max_value DECIMAL(19,4) NOT NULL,
    list_base_price DECIMAL(19,4) DEFAULT 0,
    list_price DECIMAL(19,4) DEFAULT 0,
    variable_factor VARCHAR(50) DEFAULT 'weight'
);

-- Delivery Zip Prefix Table
CREATE TABLE IF NOT EXISTS delivery_zip_prefix (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE
);

-- Carrier - Country Relation
CREATE TABLE IF NOT EXISTS delivery_carrier_country_rel (
    carrier_id INTEGER REFERENCES delivery_carrier(id) ON DELETE CASCADE,
    country_id INTEGER REFERENCES res_country(id) ON DELETE CASCADE,
    PRIMARY KEY (carrier_id, country_id)
);

-- Carrier - State Relation
CREATE TABLE IF NOT EXISTS delivery_carrier_state_rel (
    carrier_id INTEGER REFERENCES delivery_carrier(id) ON DELETE CASCADE,
    state_id INTEGER REFERENCES res_country_state(id) ON DELETE CASCADE,
    PRIMARY KEY (carrier_id, state_id)
);

-- Carrier - Zip Prefix Relation
CREATE TABLE IF NOT EXISTS delivery_carrier_zip_prefix_rel (
    carrier_id INTEGER REFERENCES delivery_carrier(id) ON DELETE CASCADE,
    zip_prefix_id INTEGER REFERENCES delivery_zip_prefix(id) ON DELETE CASCADE,
    PRIMARY KEY (carrier_id, zip_prefix_id)
);

-- Add delivery fields to sale_order
ALTER TABLE sale_order ADD COLUMN IF NOT EXISTS carrier_id INTEGER REFERENCES delivery_carrier(id);
ALTER TABLE sale_order ADD COLUMN IF NOT EXISTS shipping_weight DECIMAL(19,4) DEFAULT 0;
ALTER TABLE sale_order ADD COLUMN IF NOT EXISTS delivery_message TEXT;
ALTER TABLE sale_order ADD COLUMN IF NOT EXISTS recompute_delivery_price BOOLEAN DEFAULT FALSE;

-- Add delivery flag to sale_order_line
ALTER TABLE sale_order_line ADD COLUMN IF NOT EXISTS is_delivery BOOLEAN DEFAULT FALSE;

-- Add delivery fields to stock_picking
ALTER TABLE stock_picking ADD COLUMN IF NOT EXISTS carrier_id INTEGER REFERENCES delivery_carrier(id);
ALTER TABLE stock_picking ADD COLUMN IF NOT EXISTS carrier_tracking_ref VARCHAR(255);
ALTER TABLE stock_picking ADD COLUMN IF NOT EXISTS weight DECIMAL(19,4) DEFAULT 0;
ALTER TABLE stock_picking ADD COLUMN IF NOT EXISTS shipping_weight DECIMAL(19,4) DEFAULT 0;
ALTER TABLE stock_picking ADD COLUMN IF NOT EXISTS number_of_packages INTEGER DEFAULT 0;
