CREATE TABLE stock_routes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    sequence INT NOT NULL DEFAULT 10,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    company_id BIGINT
);
CREATE TABLE stock_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    action VARCHAR(32) NOT NULL,
    route_id BIGINT NOT NULL REFERENCES stock_routes(id) ON DELETE CASCADE,
    location_src_id BIGINT,
    location_dest_id BIGINT NOT NULL REFERENCES stock_locations(id),
    picking_type_id BIGINT NOT NULL,
    procure_method VARCHAR(32) NOT NULL DEFAULT 'make_to_stock',
    warehouse_id BIGINT,
    company_id BIGINT NOT NULL,
    sequence INT NOT NULL DEFAULT 10,
    active BOOLEAN NOT NULL DEFAULT TRUE
);
CREATE INDEX idx_stock_rules_route_dest ON stock_rules(route_id, location_dest_id, active, sequence);