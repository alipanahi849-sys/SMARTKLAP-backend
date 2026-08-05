-- Migration: 041_create_cart_items
-- Purpose: User shopping cart items (shared food + merch)

CREATE TABLE IF NOT EXISTS cart_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id   UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    product_type VARCHAR(20) NOT NULL CHECK (product_type IN ('food', 'merch')),
    size         VARCHAR(50) NOT NULL DEFAULT '',
    quantity     INT NOT NULL CHECK (quantity > 0),
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT uidx_cart_items_user_product_size UNIQUE (user_id, product_id, size)
);

CREATE INDEX IF NOT EXISTS idx_cart_items_user_id ON cart_items(user_id);
CREATE INDEX IF NOT EXISTS idx_cart_items_product_id ON cart_items(product_id);
