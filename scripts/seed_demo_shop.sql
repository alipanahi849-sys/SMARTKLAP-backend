-- Demo seed: fake shop products for mobile API testing (food + merch).
-- Safe to re-run: upserts fixed UUIDs and refreshes category-relevant image URLs.

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
    'https://loremflickr.com/400/400/shirt,jersey,sport/all',
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
    'https://loremflickr.com/400/400/shirt,jersey,football/all',
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
    'https://loremflickr.com/400/400/soccer,ball/all',
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
    'https://loremflickr.com/400/400/sticker,sheet/all',
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
    'https://loremflickr.com/400/400/tracksuit,sport/all',
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
    'https://loremflickr.com/400/400/hoodie,sport/all',
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
    'https://loremflickr.com/400/400/burger,food/all',
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
    'https://loremflickr.com/400/400/hotdog,food/all',
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
    'https://loremflickr.com/400/400/nachos,food/all',
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
    'https://loremflickr.com/400/400/popcorn,food/all',
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
    'https://loremflickr.com/400/400/cola,drink/all',
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
    'https://loremflickr.com/400/400/water,bottle/all',
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
    'https://loremflickr.com/400/400/football,mini/all',
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
    'https://loremflickr.com/400/400/scarf,football/all',
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
    'https://loremflickr.com/400/400/wrap,food/all',
    'Stadium Snacks',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '1 minute'
  )
ON CONFLICT (id) DO UPDATE SET
    image_key = EXCLUDED.image_key,
    updated_at = NOW();

-- Refresh legacy migration rows and any leftover random/picsum/cdn URLs.
UPDATE products SET image_key = 'https://loremflickr.com/400/400/shirt,jersey,sport/all', updated_at = NOW()
WHERE deleted_at IS NULL AND name = 'Sport T-shirt' AND id::text NOT LIKE 'c2000000%';

UPDATE products SET image_key = 'https://loremflickr.com/400/400/soccer,ball/all', updated_at = NOW()
WHERE deleted_at IS NULL AND name = 'Match Ball' AND id::text NOT LIKE 'c2000000%';

UPDATE products SET image_key = 'https://loremflickr.com/400/400/sticker,sheet/all', updated_at = NOW()
WHERE deleted_at IS NULL AND name = 'Club Sticker Pack' AND id::text NOT LIKE 'c2000000%';

UPDATE products SET image_key = 'https://loremflickr.com/400/400/tracksuit,sport/all', updated_at = NOW()
WHERE deleted_at IS NULL AND name = 'Training Suit' AND id::text NOT LIKE 'c2000000%';

UPDATE products SET image_key = CASE name
    WHEN 'Sport T-shirt'     THEN 'https://loremflickr.com/400/400/shirt,jersey,sport/all'
    WHEN 'Away T-shirt'      THEN 'https://loremflickr.com/400/400/shirt,jersey,football/all'
    WHEN 'Match Ball'        THEN 'https://loremflickr.com/400/400/soccer,ball/all'
    WHEN 'Club Sticker Pack' THEN 'https://loremflickr.com/400/400/sticker,sheet/all'
    WHEN 'Training Suit'     THEN 'https://loremflickr.com/400/400/tracksuit,sport/all'
    WHEN 'Winter Hoodie'     THEN 'https://loremflickr.com/400/400/hoodie,sport/all'
    WHEN 'Double Burger'     THEN 'https://loremflickr.com/400/400/burger,food/all'
    WHEN 'Club Hot Dog'      THEN 'https://loremflickr.com/400/400/hotdog,food/all'
    WHEN 'Loaded Nachos'     THEN 'https://loremflickr.com/400/400/nachos,food/all'
    WHEN 'Salted Popcorn'    THEN 'https://loremflickr.com/400/400/popcorn,food/all'
    WHEN 'Cola Zero'         THEN 'https://loremflickr.com/400/400/cola,drink/all'
    WHEN 'Mineral Water'     THEN 'https://loremflickr.com/400/400/water,bottle/all'
    WHEN 'Mini Ball'         THEN 'https://loremflickr.com/400/400/football,mini/all'
    WHEN 'Scarf'             THEN 'https://loremflickr.com/400/400/scarf,football/all'
    WHEN 'Chicken Wrap'      THEN 'https://loremflickr.com/400/400/wrap,food/all'
    ELSE image_key
END,
updated_at = NOW()
WHERE deleted_at IS NULL
  AND (
    image_key LIKE '%picsum.photos%'
    OR image_key LIKE '%cdn.smartklap.com%'
    OR image_key LIKE '%unsplash.com%'
  );

COMMIT;
