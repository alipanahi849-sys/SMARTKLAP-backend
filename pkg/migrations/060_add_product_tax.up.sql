-- Migration: 060_add_product_tax
-- VAT rate is required when registering products. price_cents stays net;
-- consumer-facing EUR amounts add this rate.

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS tax_rate_bps INTEGER NOT NULL DEFAULT 0;

ALTER TABLE products
    DROP CONSTRAINT IF EXISTS products_tax_rate_bps_check;

ALTER TABLE products
    ADD CONSTRAINT products_tax_rate_bps_check
    CHECK (tax_rate_bps >= 0 AND tax_rate_bps <= 10000);

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS tax_cents BIGINT NOT NULL DEFAULT 0;
