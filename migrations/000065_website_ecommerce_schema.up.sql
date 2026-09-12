-- migrations/000065_website_ecommerce_schema.up.sql

CREATE TABLE website_sites (
    id                  BIGSERIAL PRIMARY KEY,
    name                VARCHAR(128) NOT NULL,
    domain              VARCHAR(256) NOT NULL UNIQUE,
    company_id          BIGINT NOT NULL REFERENCES res_companies(id),
    default_language    VARCHAR(10) NOT NULL DEFAULT 'ar',
    supported_langs     TEXT[] NOT NULL DEFAULT ARRAY['ar', 'en'],
    pricelist_id        BIGINT NOT NULL REFERENCES product_pricelists(id),
    warehouse_id        BIGINT NOT NULL REFERENCES stock_warehouses(id),
    header_logo_url     TEXT,
    favicon_url         TEXT,
    google_analytics_id VARCHAR(64),
    theme_config        JSONB NOT NULL DEFAULT '{}',
    active              BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE website_pages (
    id               BIGSERIAL PRIMARY KEY,
    website_id       BIGINT NOT NULL REFERENCES website_sites(id) ON DELETE CASCADE,
    title            VARCHAR(256) NOT NULL,
    slug             VARCHAR(256) NOT NULL,
    content_json     JSONB NOT NULL DEFAULT '{}',
    meta_title       VARCHAR(256),
    meta_description TEXT,
    meta_keywords    VARCHAR(256),
    is_published     BOOLEAN NOT NULL DEFAULT FALSE,
    is_homepage      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(website_id, slug)
);

CREATE TABLE website_menus (
    id         BIGSERIAL PRIMARY KEY,
    website_id BIGINT NOT NULL REFERENCES website_sites(id) ON DELETE CASCADE,
    parent_id  BIGINT REFERENCES website_menus(id) ON DELETE CASCADE,
    name       VARCHAR(128) NOT NULL,
    url        VARCHAR(512) NOT NULL,
    sequence   INT NOT NULL DEFAULT 10,
    new_window BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE ecommerce_carts (
    id                  BIGSERIAL PRIMARY KEY,
    website_id          BIGINT NOT NULL REFERENCES website_sites(id),
    session_uuid        VARCHAR(64) NOT NULL,
    partner_id          BIGINT REFERENCES res_partners(id),
    pricelist_id        BIGINT NOT NULL REFERENCES product_pricelists(id),
    currency            VARCHAR(3) NOT NULL DEFAULT 'SAR',
    state               VARCHAR(32) NOT NULL DEFAULT 'active',
    shipping_address_id BIGINT REFERENCES res_partners(id),
    invoice_address_id  BIGINT REFERENCES res_partners(id),
    delivery_method_id  BIGINT REFERENCES delivery_carrier(id),
    shipping_amount     NUMERIC(15,4) NOT NULL DEFAULT 0,
    coupon_code         VARCHAR(64),
    discount_amount     NUMERIC(15,4) NOT NULL DEFAULT 0,
    amount_untaxed      NUMERIC(15,4) NOT NULL DEFAULT 0,
    amount_tax          NUMERIC(15,4) NOT NULL DEFAULT 0,
    amount_total        NUMERIC(15,4) NOT NULL DEFAULT 0,
    converted_order_id  BIGINT REFERENCES sale_orders(id),
    last_activity_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ecommerce_carts_session ON ecommerce_carts(session_uuid);
CREATE INDEX idx_ecommerce_carts_state_activity ON ecommerce_carts(state, last_activity_at);

CREATE TABLE ecommerce_cart_lines (
    id          BIGSERIAL PRIMARY KEY,
    cart_id     BIGINT NOT NULL REFERENCES ecommerce_carts(id) ON DELETE CASCADE,
    product_id  BIGINT NOT NULL REFERENCES product_templates(id),
    quantity    NUMERIC(15,4) NOT NULL DEFAULT 1,
    price_unit  NUMERIC(15,4) NOT NULL DEFAULT 0,
    discount    NUMERIC(5,2) NOT NULL DEFAULT 0,
    price_total NUMERIC(15,4) NOT NULL DEFAULT 0,
    tax_ids     BIGINT[] DEFAULT ARRAY[]::BIGINT[],
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE portal_users (
    id              BIGSERIAL PRIMARY KEY,
    partner_id      BIGINT NOT NULL REFERENCES res_partners(id) ON DELETE CASCADE,
    email           VARCHAR(128) NOT NULL UNIQUE,
    password_hash   VARCHAR(256) NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at   TIMESTAMPTZ,
    company_id      BIGINT NOT NULL REFERENCES res_companies(id),
    invite_token    VARCHAR(128) UNIQUE,
    invite_accepted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
