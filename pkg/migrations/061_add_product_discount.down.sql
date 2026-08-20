-- Migration: 061_add_product_discount (down)

ALTER TABLE products
    DROP CONSTRAINT IF EXISTS products_discount_rate_bps_check;

ALTER TABLE products
    DROP COLUMN IF EXISTS discount_rate_bps;
