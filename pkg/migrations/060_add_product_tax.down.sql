-- Migration: 060_add_product_tax (down)

ALTER TABLE orders
    DROP COLUMN IF EXISTS tax_cents;

ALTER TABLE products
    DROP CONSTRAINT IF EXISTS products_tax_rate_bps_check;

ALTER TABLE products
    DROP COLUMN IF EXISTS tax_rate_bps;
