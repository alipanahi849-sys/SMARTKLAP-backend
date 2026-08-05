-- Migration: 042_create_orders
-- Purpose: Orders and order line items (checkout from cart)

CREATE TABLE IF NOT EXISTS orders (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status           VARCHAR(30) NOT NULL DEFAULT 'pending_payment'
                     CHECK (status IN ('pending_payment', 'paid', 'cancelled')),
    delivery_method  VARCHAR(20) NOT NULL CHECK (delivery_method IN ('seat', 'pickup')),
    seat_number      VARCHAR(50) NOT NULL DEFAULT '',
    subtotal_cents   BIGINT NOT NULL DEFAULT 0,
    shipping_cents   BIGINT NOT NULL DEFAULT 0,
    total_cents      BIGINT NOT NULL DEFAULT 0,
    created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

CREATE TABLE IF NOT EXISTS order_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id     UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id   UUID NOT NULL,
    product_type VARCHAR(20) NOT NULL,
    name         VARCHAR(200) NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    price_cents  BIGINT NOT NULL,
    quantity     INT NOT NULL CHECK (quantity > 0),
    size         VARCHAR(50) NOT NULL DEFAULT '',
    image_key    VARCHAR(500) NOT NULL DEFAULT '',
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
