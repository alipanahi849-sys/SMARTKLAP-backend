-- Migration: 041_add_stock_quantity_to_products
-- NULL stock_quantity = unlimited inventory; non-NULL must be >= 0.

ALTER TABLE products ADD COLUMN IF NOT EXISTS stock_quantity INTEGER NULL;

ALTER TABLE products DROP CONSTRAINT IF EXISTS products_stock_quantity_check;
ALTER TABLE products ADD CONSTRAINT products_stock_quantity_check
    CHECK (stock_quantity IS NULL OR stock_quantity >= 0);
