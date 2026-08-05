-- Migration: 042_create_cart_items
-- Per-user cart lines; stock is NOT decremented until payment completes.

CREATE TABLE IF NOT EXISTS cart_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id   UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    product_type VARCHAR(20) NOT NULL,
    size         VARCHAR(50) NOT NULL DEFAULT '',
    quantity     INTEGER NOT NULL CHECK (quantity > 0),
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT cart_items_product_type_check CHECK (product_type IN ('food', 'merch')),
    CONSTRAINT cart_items_user_product_size_unique UNIQUE (user_id, product_id, size)
);

CREATE INDEX IF NOT EXISTS idx_cart_items_user_id ON cart_items (user_id);
