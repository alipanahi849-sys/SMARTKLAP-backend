-- Migration: 047_create_orders
-- Checkout orders created from the user's cart.

CREATE TABLE IF NOT EXISTS orders (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    delivery_method  VARCHAR(20) NOT NULL,
    seat_number      VARCHAR(50) NOT NULL DEFAULT '',
    status           VARCHAR(30) NOT NULL DEFAULT 'pending_payment',
    payment_method   VARCHAR(20),
    subtotal_cents   BIGINT NOT NULL DEFAULT 0 CHECK (subtotal_cents >= 0),
    subtotal_points  INTEGER NOT NULL DEFAULT 0 CHECK (subtotal_points >= 0),
    shipping_cents   BIGINT NOT NULL DEFAULT 0 CHECK (shipping_cents >= 0),
    shipping_points  INTEGER NOT NULL DEFAULT 0 CHECK (shipping_points >= 0),
    total_cents      BIGINT NOT NULL DEFAULT 0 CHECK (total_cents >= 0),
    total_points     INTEGER NOT NULL DEFAULT 0 CHECK (total_points >= 0),
    paid_at          TIMESTAMP,
    created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT orders_delivery_method_check CHECK (delivery_method IN ('seat', 'pickup')),
    CONSTRAINT orders_status_check CHECK (status IN ('pending_payment', 'paid', 'cancelled')),
    CONSTRAINT orders_payment_method_check CHECK (
        payment_method IS NULL OR payment_method IN ('card', 'points')
    )
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders (user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);

CREATE TABLE IF NOT EXISTS order_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id     UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id   UUID NOT NULL REFERENCES products(id),
    product_type VARCHAR(20) NOT NULL,
    size         VARCHAR(50) NOT NULL DEFAULT '',
    name         VARCHAR(200) NOT NULL,
    quantity     INTEGER NOT NULL CHECK (quantity > 0),
    price_cents  BIGINT NOT NULL CHECK (price_cents >= 0),
    price_points INTEGER NOT NULL CHECK (price_points >= 0),
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT order_items_product_type_check CHECK (product_type IN ('food', 'merch'))
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items (order_id);
