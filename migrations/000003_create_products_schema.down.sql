-- 000003_create_products_schema.down.sql
-- Rollback product catalog schema

DROP TABLE IF EXISTS product_pricelist_items CASCADE;
DROP TABLE IF EXISTS product_pricelists CASCADE;
DROP TABLE IF EXISTS product_variant_attributes CASCADE;
DROP TABLE IF EXISTS product_variants CASCADE;
DROP TABLE IF EXISTS product_attribute_values CASCADE;
DROP TABLE IF EXISTS product_attributes CASCADE;
DROP TABLE IF EXISTS product_templates CASCADE;
DROP TABLE IF EXISTS product_categories CASCADE;
DROP TABLE IF EXISTS uom_uoms CASCADE;
