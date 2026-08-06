-- Migration: 045_create_product_size_stocks
-- Per-size inventory for merch products with available_sizes.

CREATE TABLE IF NOT EXISTS product_size_stocks (
    product_id     UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    size           VARCHAR(50) NOT NULL,
    stock_quantity INTEGER NULL,
    sold_out       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (product_id, size),
    CONSTRAINT product_size_stocks_stock_quantity_check
        CHECK (stock_quantity IS NULL OR stock_quantity >= 0)
);

CREATE INDEX IF NOT EXISTS idx_product_size_stocks_product_id
    ON product_size_stocks (product_id);

-- Backfill from product-level stock for existing sized merch.
INSERT INTO product_size_stocks (product_id, size, stock_quantity, sold_out)
SELECT
    p.id,
    sz.val,
    p.stock_quantity,
    CASE WHEN p.stock_quantity = 0 AND p.sold_out THEN TRUE ELSE FALSE END
FROM products p
CROSS JOIN LATERAL jsonb_array_elements_text(p.available_sizes) AS sz(val)
WHERE p.product_type = 'merch'
  AND p.deleted_at IS NULL
  AND jsonb_array_length(p.available_sizes) > 0
ON CONFLICT (product_id, size) DO NOTHING;

UPDATE products
SET stock_quantity = NULL,
    sold_out = FALSE,
    updated_at = NOW()
WHERE product_type = 'merch'
  AND deleted_at IS NULL
  AND jsonb_array_length(available_sizes) > 0;
