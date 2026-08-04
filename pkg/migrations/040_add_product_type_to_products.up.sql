-- Migration: 040_add_product_type_to_products
-- Purpose: Distinguish food items (no sizes) from merch/clothing (optional sizes).

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS product_type VARCHAR(20) NOT NULL DEFAULT 'merch';

UPDATE products SET product_type = 'merch' WHERE product_type IS NULL OR product_type = '';

ALTER TABLE products DROP CONSTRAINT IF EXISTS products_category_check;
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_product_type_check;

ALTER TABLE products
    ADD CONSTRAINT products_product_type_check
        CHECK (product_type IN ('food', 'merch'));
