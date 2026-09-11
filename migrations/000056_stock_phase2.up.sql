CREATE TABLE stock_package_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    height NUMERIC(10,2) NOT NULL DEFAULT 0,
    width NUMERIC(10,2) NOT NULL DEFAULT 0,
    length NUMERIC(10,2) NOT NULL DEFAULT 0,
    max_weight NUMERIC(10,2) NOT NULL DEFAULT 0,
    barcode VARCHAR(128),
    sequence INT NOT NULL DEFAULT 10,
    company_id BIGINT
);

CREATE TABLE stock_packages (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,
    package_type_id BIGINT REFERENCES stock_package_types(id),
    location_id BIGINT NOT NULL REFERENCES stock_locations(id),
    company_id BIGINT NOT NULL,
    weight NUMERIC(10,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE product_packagings (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    product_id BIGINT NOT NULL,
    barcode VARCHAR(128),
    qty NUMERIC(15,4) NOT NULL DEFAULT 1,
    package_type_id BIGINT REFERENCES stock_package_types(id),
    company_id BIGINT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

ALTER TABLE stock_move_lines ADD COLUMN IF NOT EXISTS result_package_id BIGINT REFERENCES stock_packages(id);
ALTER TABLE stock_quants ADD COLUMN IF NOT EXISTS package_id BIGINT REFERENCES stock_packages(id);

CREATE TABLE stock_barcode_nomenclatures (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    company_id BIGINT NOT NULL
);
CREATE TABLE stock_barcode_rules (
    id BIGSERIAL PRIMARY KEY,
    nomenclature_id BIGINT NOT NULL REFERENCES stock_barcode_nomenclatures(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    sequence INT NOT NULL DEFAULT 10,
    encoding VARCHAR(32) NOT NULL,
    type VARCHAR(32) NOT NULL,
    pattern TEXT NOT NULL,
    gs1_content_type VARCHAR(64),
    associated BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE stock_storage_categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    max_weight NUMERIC(10,2) NOT NULL DEFAULT 0,
    allow_new_product VARCHAR(16) NOT NULL DEFAULT 'same',
    company_id BIGINT NOT NULL
);
CREATE TABLE stock_storage_category_capacities (
    id BIGSERIAL PRIMARY KEY,
    storage_category_id BIGINT NOT NULL REFERENCES stock_storage_categories(id) ON DELETE CASCADE,
    package_type_id BIGINT NOT NULL REFERENCES stock_package_types(id),
    quantity INT NOT NULL DEFAULT 0
);
CREATE TABLE stock_putaway_rules (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT,
    category_id BIGINT,
    location_in_id BIGINT NOT NULL REFERENCES stock_locations(id),
    location_out_id BIGINT NOT NULL REFERENCES stock_locations(id),
    storage_category_id BIGINT REFERENCES stock_storage_categories(id),
    sequence INT NOT NULL DEFAULT 10,
    company_id BIGINT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);