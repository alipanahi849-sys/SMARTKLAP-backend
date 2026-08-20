-- Migration: 061_add_product_discount
-- Optional percent-off per product (panel). price_cents stays the pre-discount
-- net; consumer-facing amounts apply this rate then VAT.

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS discount_rate_bps INTEGER NOT NULL DEFAULT 0;

ALTER TABLE products
    DROP CONSTRAINT IF EXISTS products_discount_rate_bps_check;

ALTER TABLE products
    ADD CONSTRAINT products_discount_rate_bps_check
    CHECK (discount_rate_bps >= 0 AND discount_rate_bps <= 10000);
