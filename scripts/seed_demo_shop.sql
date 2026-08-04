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
  ),
  (
    'c2000000-0000-4000-8000-000000000016',
    'food',
    'Veggie Pizza Slice',
    '',
    'Stone-baked slice with mozzarella and peppers.',
    'snacks',
    490, 490,
    'https://loremflickr.com/400/400/pizza,food/all',
    'Stadium Snacks',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '16 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000017',
    'food',
    'French Fries',
    'Large',
    'Crispy golden fries with sea salt.',
    'snacks',
    420, 420,
    'https://loremflickr.com/400/400/fries,food/all',
    'Stadium Snacks',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '17 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000018',
    'food',
    'Energy Drink',
    '33 cl',
    'Cold citrus energy drink.',
    'drinks',
    380, 380,
    'https://loremflickr.com/400/400/energy,drink/all',
    'Stadium Bar',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '18 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000019',
    'food',
    'Orange Juice',
    'Fresh',
    'Freshly squeezed orange juice.',
    'drinks',
    360, 360,
    'https://loremflickr.com/400/400/juice,orange/all',
    'Stadium Bar',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '19 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000020',
    'food',
    'Club Sandwich',
    '',
    'Triple-layer sandwich with chicken and bacon.',
    'sandwiches',
    750, 750,
    'https://loremflickr.com/400/400/sandwich,food/all',
    'Stadium Snacks',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '20 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000021',
    'food',
    'BBQ Wings',
    '6 pcs',
    'Smoky barbecue chicken wings.',
    'snacks',
    690, 690,
    'https://loremflickr.com/400/400/wings,food/all',
    'Stadium Snacks',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '21 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000022',
    'food',
    'Chocolate Muffin',
    '',
    'Soft muffin with dark chocolate chips.',
    'snacks',
    310, 310,
    'https://loremflickr.com/400/400/muffin,food/all',
    'Stadium Snacks',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '22 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000023',
    'food',
    'Iced Coffee',
    '',
    'Cold brew with ice and milk.',
    'drinks',
    410, 410,
    'https://loremflickr.com/400/400/coffee,drink/all',
    'Stadium Bar',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '23 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000024',
    'merch',
    'Goalkeeper Gloves',
    'Pro',
    'Match-grade goalkeeper gloves.',
    'sport-suits',
    4200, 4200,
    'https://loremflickr.com/400/400/gloves,football/all',
    'Sport Mall 2',
    '["M","L"]'::jsonb,
    true,
    NOW() - INTERVAL '24 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000025',
    'merch',
    'Captain Armband',
    '',
    'Elastic captain armband in club colors.',
    'stickers',
    900, 900,
    'https://loremflickr.com/400/400/football,captain/all',
    'Fan Shop',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '25 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000026',
    'merch',
    'Stadium Cap',
    '',
    'Adjustable cap with embroidered crest.',
    't-shirts',
    1800, 1800,
    'https://loremflickr.com/400/400/cap,sport/all',
    'Fan Shop',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '26 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000027',
    'merch',
    'Fan Flag',
    'Large',
    'Supporter flag for the stands.',
    'stickers',
    1500, 1500,
    'https://loremflickr.com/400/400/flag,football/all',
    'Fan Shop',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '27 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000028',
    'merch',
    'Training Shorts',
    '',
    'Breathable training shorts.',
    'sport-suits',
    2400, 2400,
    'https://loremflickr.com/400/400/shorts,sport/all',
    'Sport Mall 2',
    '["M","L","XL"]'::jsonb,
    true,
    NOW() - INTERVAL '28 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000029',
    'merch',
    'Socks Pack',
    '3 pairs',
    'Match-day football socks.',
    'sport-suits',
    1200, 1200,
    'https://loremflickr.com/400/400/socks,sport/all',
    'Sport Mall 2',
    '["M","L"]'::jsonb,
    true,
    NOW() - INTERVAL '29 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000030',
    'merch',
    'Water Bottle',
    '750 ml',
    'Reusable club water bottle.',
    'balls',
    1600, 1600,
    'https://loremflickr.com/400/400/bottle,sport/all',
    'Fan Shop',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '30 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000031',
    'food',
    'Pretzel',
    'Salted',
    'Warm salted stadium pretzel.',
    'snacks',
    280, 280,
    'https://loremflickr.com/400/400/pretzel,food/all',
    'Stadium Snacks',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '31 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000032',
    'food',
    'Fish & Chips',
    '',
    'Beer-battered fish with thick-cut chips.',
    'sandwiches',
    920, 920,
    'https://loremflickr.com/400/400/fish,chips/all',
    'Stadium Snacks',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '32 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000033',
    'food',
    'Sparkling Water',
    '50 cl',
    'Chilled sparkling mineral water.',
    'drinks',
    270, 270,
    'https://loremflickr.com/400/400/sparkling,water/all',
    'Stadium Bar',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '33 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000034',
    'merch',
    'Retro Jersey',
    '1998 edition',
    'Classic throwback home jersey.',
    't-shirts',
    6200, 6200,
    'https://loremflickr.com/400/400/jersey,vintage/all',
    'Sport Mall 2',
    '["M","L","XL"]'::jsonb,
    true,
    NOW() - INTERVAL '34 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000035',
    'merch',
    'Pump Ball',
    'With pump',
    'Training ball with mini hand pump.',
    'balls',
    3900, 3900,
    'https://loremflickr.com/400/400/football,training/all',
    'Sport Mall 2',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '35 minutes'
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
