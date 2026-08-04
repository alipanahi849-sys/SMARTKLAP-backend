-- Demo seed: fake shop products for mobile API testing (food + merch).
-- Safe to re-run: upserts fixed UUIDs and refreshes image URLs.

BEGIN;

INSERT INTO products (
    id, product_type, name, subname, description, category,
    price_cents, price_points, image_key, seller_name, available_sizes,
    is_active, created_at
) VALUES
  (
    'c2000000-0000-4000-8000-000000000001',
    'merch',
    'Sport T-shirt',
    'Home kit',
    'Official club home kit jersey.',
    't-shirts',
    3250, 3250,
    'https://picsum.photos/seed/shop-merch-shirt-home/400/400',
    'Sport Mall 2',
    '["M","L","XL","XXL"]'::jsonb,
    true,
    NOW() - INTERVAL '15 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000002',
    'merch',
    'Away T-shirt',
    'Away kit',
    'Lightweight away jersey for match days.',
    't-shirts',
    3250, 3250,
    'https://picsum.photos/seed/shop-merch-shirt-away/400/400',
    'Sport Mall 2',
    '["S","M","L","XL"]'::jsonb,
    true,
    NOW() - INTERVAL '14 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000003',
    'merch',
    'Match Ball',
    'Official size 5',
    'Premium match ball used on the pitch.',
    'balls',
    4500, 4500,
    'https://picsum.photos/seed/shop-merch-ball/400/400',
    'Sport Mall 2',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '13 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000004',
    'merch',
    'Club Sticker Pack',
    '10 stickers',
    'Collectible club logo stickers.',
    'stickers',
    500, 500,
    'https://picsum.photos/seed/shop-merch-stickers/400/400',
    'Fan Shop',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '12 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000005',
    'merch',
    'Training Suit',
    'Full set',
    'Lightweight training tracksuit.',
    'sport-suits',
    8900, 8900,
    'https://picsum.photos/seed/shop-merch-suit/400/400',
    'Sport Mall 2',
    '["M","L","XL"]'::jsonb,
    true,
    NOW() - INTERVAL '11 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000006',
    'merch',
    'Winter Hoodie',
    'Limited edition',
    'Warm hoodie with embroidered club crest.',
    't-shirts',
    5500, 5500,
    'https://picsum.photos/seed/shop-merch-hoodie/400/400',
    'Fan Shop',
    '["M","L","XL","XXL"]'::jsonb,
    true,
    NOW() - INTERVAL '10 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000007',
    'food',
    'Double Burger',
    '',
    'Double beef patty with cheese, lettuce and house sauce.',
    'sandwiches',
    820, 820,
    'https://picsum.photos/seed/shop-food-burger/400/400',
    'Stadium Snacks',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '9 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000008',
    'food',
    'Club Hot Dog',
    '',
    'Grilled sausage in a brioche bun with mustard.',
    'sandwiches',
    650, 650,
    'https://picsum.photos/seed/shop-food-hotdog/400/400',
    'Stadium Snacks',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '8 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000009',
    'food',
    'Loaded Nachos',
    '',
    'Crispy nachos with melted cheese and jalapeños.',
    'snacks',
    590, 590,
    'https://picsum.photos/seed/shop-food-nachos/400/400',
    'Stadium Snacks',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '7 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000010',
    'food',
    'Salted Popcorn',
    '',
    'Large bucket of fresh stadium popcorn.',
    'snacks',
    350, 350,
    'https://picsum.photos/seed/shop-food-popcorn/400/400',
    'Stadium Snacks',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '6 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000011',
    'food',
    'Cola Zero',
    '50 cl',
    'Chilled sugar-free cola.',
    'drinks',
    320, 320,
    'https://picsum.photos/seed/shop-food-cola/400/400',
    'Stadium Bar',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '5 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000012',
    'food',
    'Mineral Water',
    '50 cl',
    'Still mineral water.',
    'drinks',
    250, 250,
    'https://picsum.photos/seed/shop-food-water/400/400',
    'Stadium Bar',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '4 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000013',
    'merch',
    'Mini Ball',
    'Souvenir',
    'Size 1 souvenir ball for kids.',
    'balls',
    1800, 1800,
    'https://picsum.photos/seed/shop-merch-mini-ball/400/400',
    'Fan Shop',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '3 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000014',
    'merch',
    'Scarf',
    'Home colors',
    'Knitted supporter scarf in club colors.',
    'stickers',
    2200, 2200,
    'https://picsum.photos/seed/shop-merch-scarf/400/400',
    'Fan Shop',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '2 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000015',
    'food',
    'Chicken Wrap',
    '',
    'Grilled chicken wrap with fresh vegetables.',
    'sandwiches',
    780, 780,
    'https://picsum.photos/seed/shop-food-wrap/400/400',
    'Stadium Snacks',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '1 minute'
  )
ON CONFLICT (id) DO UPDATE SET
    image_key = EXCLUDED.image_key,
    updated_at = NOW();

-- Fix legacy migration seed rows (broken cdn.smartklap.com URLs).
UPDATE products SET image_key = 'https://picsum.photos/seed/shop-legacy-shirt/400/400', updated_at = NOW()
WHERE deleted_at IS NULL AND name = 'Sport T-shirt' AND image_key LIKE '%cdn.smartklap.com%';

UPDATE products SET image_key = 'https://picsum.photos/seed/shop-legacy-ball/400/400', updated_at = NOW()
WHERE deleted_at IS NULL AND name = 'Match Ball' AND image_key LIKE '%cdn.smartklap.com%';

UPDATE products SET image_key = 'https://picsum.photos/seed/shop-legacy-stickers/400/400', updated_at = NOW()
WHERE deleted_at IS NULL AND name = 'Club Sticker Pack' AND image_key LIKE '%cdn.smartklap.com%';

UPDATE products SET image_key = 'https://picsum.photos/seed/shop-legacy-suit/400/400', updated_at = NOW()
WHERE deleted_at IS NULL AND name = 'Training Suit' AND image_key LIKE '%cdn.smartklap.com%';

-- Ensure any active product without an image gets a placeholder.
UPDATE products SET
    image_key = 'https://picsum.photos/seed/shop-default-' || replace(id::text, '-', '') || '/400/400',
    updated_at = NOW()
WHERE deleted_at IS NULL AND (image_key IS NULL OR trim(image_key) = '');

COMMIT;
