-- Migration: 044_add_sold_out_to_products
-- sold_out is set when limited stock reaches 0 after a successful order payment.

ALTER TABLE products ADD COLUMN IF NOT EXISTS sold_out BOOLEAN NOT NULL DEFAULT FALSE;
