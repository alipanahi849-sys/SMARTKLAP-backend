-- Migration: 039_create_products
-- Purpose: Mobile Store module — merch catalog with EUR and point pricing.
-- Handles legacy products table from removed 034_create_shop migration.

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

-- Upgrade legacy schema (034_create_shop) when table already exists.
ALTER TABLE products ADD COLUMN IF NOT EXISTS subname VARCHAR(200) NOT NULL DEFAULT '';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'products' AND column_name = 'points_price'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'products' AND column_name = 'price_points'
    ) THEN
        ALTER TABLE products RENAME COLUMN points_price TO price_points;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'products' AND column_name = 'image_url'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'products' AND column_name = 'image_key'
    ) THEN
        ALTER TABLE products RENAME COLUMN image_url TO image_key;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'products' AND column_name = 'sizes'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'products' AND column_name = 'available_sizes'
    ) THEN
        ALTER TABLE products RENAME COLUMN sizes TO available_sizes;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_products_category_active ON products (category, is_active);
CREATE INDEX IF NOT EXISTS idx_products_deleted_at ON products (deleted_at);

-- Seed sample merch when the catalog is empty.
INSERT INTO products (name, subname, description, category, price_cents, price_points, image_key, seller_name, available_sizes)
SELECT v.name, v.subname, v.description, v.category, v.price_cents, v.price_points, v.image_key, v.seller_name, v.available_sizes
FROM (
    VALUES
        (
            'Sport T-shirt',
            'Home kit',
            'Official club home kit jersey.',
            't-shirts',
            3250::bigint,
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
            4500::bigint,
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
            500::bigint,
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
            8900::bigint,
            8900,
            'https://cdn.smartklap.com/store/suit.png',
            'Sport Mall 2',
            '["M","L","XL"]'::jsonb
        )
) AS v(name, subname, description, category, price_cents, price_points, image_key, seller_name, available_sizes)
WHERE NOT EXISTS (SELECT 1 FROM products LIMIT 1);
