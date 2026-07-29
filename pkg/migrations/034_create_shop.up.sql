-- Migration: 034_create_shop
-- Purpose: Mobile Snacks + Store + Cart + Orders. Prices are stored in cents
-- to avoid floating point; points_price supports the EUR/POINT toggle.

CREATE TABLE IF NOT EXISTS snacks (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    price_cents  BIGINT NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
    points_price INTEGER NOT NULL DEFAULT 0 CHECK (points_price >= 0),
    category     VARCHAR(30) NOT NULL DEFAULT 'snacks'
        CHECK (category IN ('sandwiches', 'snacks', 'drinks')),
    image_url    VARCHAR(500),
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMP,
    created_by   UUID,
    updated_by   UUID
);

CREATE INDEX IF NOT EXISTS idx_snacks_category   ON snacks (category);
CREATE INDEX IF NOT EXISTS idx_snacks_deleted_at ON snacks (deleted_at);

CREATE TABLE IF NOT EXISTS products (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(255) NOT NULL,
    seller_name  VARCHAR(255),
    description  TEXT,
    price_cents  BIGINT NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
    points_price INTEGER NOT NULL DEFAULT 0 CHECK (points_price >= 0),
    category     VARCHAR(30) NOT NULL DEFAULT 't-shirts'
        CHECK (category IN ('t-shirts', 'balls', 'stickers', 'sport-suits')),
    image_url    VARCHAR(500),
    -- JSON array of size labels, e.g. ["M","L","XL","XXL"].
    sizes        JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMP,
    created_by   UUID,
    updated_by   UUID
);

CREATE INDEX IF NOT EXISTS idx_products_category   ON products (category);
CREATE INDEX IF NOT EXISTS idx_products_deleted_at ON products (deleted_at);

-- One shared cart per user across snacks and merch (product_type discriminates).
CREATE TABLE IF NOT EXISTS cart_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_type VARCHAR(10) NOT NULL CHECK (product_type IN ('snack', 'merch')),
    product_id   UUID NOT NULL,
    quantity     INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    size         VARCHAR(10) NOT NULL DEFAULT '',
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT uidx_cart_items_user_product UNIQUE (user_id, product_type, product_id, size)
);

CREATE INDEX IF NOT EXISTS idx_cart_items_user_id ON cart_items (user_id);

CREATE TABLE IF NOT EXISTS orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    delivery_method VARCHAR(10) NOT NULL CHECK (delivery_method IN ('seat', 'pickup')),
    seat_number     VARCHAR(20) NOT NULL DEFAULT '',
    subtotal_cents  BIGINT NOT NULL DEFAULT 0,
    shipping_cents  BIGINT NOT NULL DEFAULT 0,
    total_cents     BIGINT NOT NULL DEFAULT 0,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending_payment'
        CHECK (status IN ('pending_payment', 'paid', 'cancelled')),
    payment_method  VARCHAR(30) NOT NULL DEFAULT '',
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_user_created ON orders (user_id, created_at DESC);

-- Immutable snapshot of purchased items (name/price at checkout time).
CREATE TABLE IF NOT EXISTS order_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id     UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_type VARCHAR(10) NOT NULL CHECK (product_type IN ('snack', 'merch')),
    product_id   UUID NOT NULL,
    name         VARCHAR(255) NOT NULL,
    price_cents  BIGINT NOT NULL DEFAULT 0,
    quantity     INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    size         VARCHAR(10) NOT NULL DEFAULT '',
    image_url    VARCHAR(500),
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items (order_id);
