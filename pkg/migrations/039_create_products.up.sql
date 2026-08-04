-- Migration: 039_create_products
-- Purpose: Mobile Store module — merch catalog with EUR and point pricing.

CREATE TABLE IF NOT EXISTS products (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(200) NOT NULL,
    subname      VARCHAR(200) NOT NULL DEFAULT '',
    description  TEXT NOT NULL DEFAULT '',
    category     VARCHAR(50) NOT NULL
        CHECK (category IN ('t-shirts', 'balls', 'stickers', 'sport-suits')),
    price_cents  BIGINT NOT NULL CHECK (price_cents >= 0),
    price_points INTEGER NOT NULL CHECK (price_points >= 0),
    image_key    VARCHAR(500) NOT NULL DEFAULT '',
    seller_name  VARCHAR(200) NOT NULL DEFAULT '',
    available_sizes JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_products_category_active ON products (category, is_active);
CREATE INDEX IF NOT EXISTS idx_products_deleted_at ON products (deleted_at);

-- Seed sample merch for development.
INSERT INTO products (name, subname, description, category, price_cents, price_points, image_key, seller_name, available_sizes)
VALUES
    (
        'Sport T-shirt',
        'Home kit',
        'Official club home kit jersey.',
        't-shirts',
        3250,
        3250,
        'https://cdn.smartklap.com/store/shirt.png',
        'Sport Mall 2',
        '["M","L","XL","XXL"]'::jsonb
    ),
    (
        'Match Ball',
        'Official size 5',
        'Premium match ball used on the pitch.',
        'balls',
        4500,
        4500,
        'https://cdn.smartklap.com/store/ball.png',
        'Sport Mall 2',
        '[]'::jsonb
    ),
    (
        'Club Sticker Pack',
        '10 stickers',
        'Collectible club logo stickers.',
        'stickers',
        500,
        500,
        'https://cdn.smartklap.com/store/stickers.png',
        'Fan Shop',
        '[]'::jsonb
    ),
    (
        'Training Suit',
        'Full set',
        'Lightweight training tracksuit.',
        'sport-suits',
        8900,
        8900,
        'https://cdn.smartklap.com/store/suit.png',
        'Sport Mall 2',
        '["M","L","XL"]'::jsonb
    );
