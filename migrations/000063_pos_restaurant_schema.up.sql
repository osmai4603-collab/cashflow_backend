CREATE TABLE IF NOT EXISTS restaurant_floors (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    pos_config_id BIGINT NOT NULL REFERENCES pos_configs(id) ON DELETE CASCADE,
    sequence INT NOT NULL DEFAULT 10,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    company_id BIGINT NOT NULL REFERENCES res_companies(id)
);

CREATE TABLE IF NOT EXISTS restaurant_tables (
    id BIGSERIAL PRIMARY KEY,
    floor_id BIGINT NOT NULL REFERENCES restaurant_floors(id) ON DELETE CASCADE,
    name VARCHAR(32) NOT NULL,
    seats INT NOT NULL DEFAULT 4 CHECK (seats > 0),
    shape VARCHAR(16) NOT NULL DEFAULT 'square',
    position_x NUMERIC(10,2) NOT NULL DEFAULT 0,
    position_y NUMERIC(10,2) NOT NULL DEFAULT 0,
    width NUMERIC(10,2) NOT NULL DEFAULT 100,
    height NUMERIC(10,2) NOT NULL DEFAULT 100,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    company_id BIGINT NOT NULL REFERENCES res_companies(id)
);

CREATE TABLE IF NOT EXISTS pos_kitchen_tickets (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES pos_orders(id) ON DELETE CASCADE,
    table_id BIGINT REFERENCES restaurant_tables(id),
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    course VARCHAR(32) NOT NULL DEFAULT 'main',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pos_kitchen_ticket_lines (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL REFERENCES pos_kitchen_tickets(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES product_templates(id),
    product_name VARCHAR(256) NOT NULL,
    qty NUMERIC(15,4) NOT NULL CHECK (qty > 0),
    notes TEXT
);