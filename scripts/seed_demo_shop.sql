-- Demo seed: fake shop products for mobile API testing (food + merch).
-- Safe to re-run: upserts fixed UUIDs and refreshes descriptions, sizes and images.

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
    'Official club home kit jersey for the current season. Made from lightweight breathable polyester with moisture-wicking finish, reinforced stitching on shoulders and side panels, and the club crest embroidered on the chest. Ideal for match days, training sessions, or everyday supporter wear. Machine washable at 30°C; do not tumble dry.',
    't-shirts',
    3250, 3250,
    'https://loremflickr.com/400/400/shirt,jersey,sport/all',
    'Sport Mall 2',
    '["S","M","L","XL","XXL"]'::jsonb,
    true,
    NOW() - INTERVAL '15 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000002',
    'merch',
    'Away T-shirt',
    'Away kit',
    'Lightweight away jersey designed for warm evening kick-offs and travel days. Features a slim athletic fit, mesh ventilation under the arms, sublimated sponsor print, and contrast collar detail in away colours. Same premium fabric as the home shirt with a softer hand-feel for all-day comfort inside and outside the stadium.',
    't-shirts',
    3250, 3250,
    'https://loremflickr.com/400/400/shirt,jersey,football/all',
    'Sport Mall 2',
    '["S","M","L","XL","XXL"]'::jsonb,
    true,
    NOW() - INTERVAL '14 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000003',
    'merch',
    'Match Ball',
    'Size 5 official',
    'FIFA-quality match ball built for consistent flight and reliable grip in all weather conditions. Hand-stitched panels, latex bladder, and textured surface for improved control during passes and shots. Used by the first team in league fixtures; also popular for serious training and five-a-side. Ships inflated; pump sold separately.',
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
    'Collectible sticker sheet with ten weather-resistant vinyl decals featuring the club badge, motto, and iconic stadium silhouette. Perfect for laptops, water bottles, scooters, and guitar cases. UV-coated to resist fading in sunlight; peels cleanly without leaving residue. A favourite gift for young supporters and away-day travellers.',
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
    'Complete two-piece training tracksuit with zip jacket and tapered joggers in club colours. Soft brushed inner layer keeps muscles warm before sessions; elastic cuffs and drawcord waist ensure a secure fit during drills. Side pockets on jacket and pants; club logo on chest and thigh. Designed for travel, warm-ups, and casual supporter wear.',
    'sport-suits',
    8900, 8900,
    'https://loremflickr.com/400/400/tracksuit,sport/all',
    'Sport Mall 2',
    '["S","M","L","XL","XXL"]'::jsonb,
    true,
    NOW() - INTERVAL '11 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000006',
    'merch',
    'Winter Hoodie',
    'Limited edition',
    'Limited-edition heavyweight hoodie with fleece-lined hood, kangaroo pocket, and ribbed hem and cuffs. Embroidered club crest on the front and season year on the sleeve. Relaxed fit layers easily over a match shirt on cold evening games. Double-stitched seams and premium cotton blend for durability through the winter schedule.',
    't-shirts',
    5500, 5500,
    'https://loremflickr.com/400/400/hoodie,sport/all',
    'Fan Shop',
    '["S","M","L","XL","XXL"]'::jsonb,
    true,
    NOW() - INTERVAL '10 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000007',
    'food',
    'Double Burger',
    'With cheese',
    'Two flame-grilled beef patties stacked with melted cheddar, crisp lettuce, ripe tomato, pickled gherkins, and our signature smoky house sauce on a toasted brioche bun. Served hot from the stadium grill — juicy, filling, and made for halftime hunger. Contains gluten and dairy; ask staff for allergen information before ordering.',
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
    'With mustard',
    'Premium grilled pork sausage in a soft brioche bun with classic yellow mustard and optional fried onions. Quick to serve, easy to eat in the stands, and consistently one of our best-selling match-day snacks. Best enjoyed immediately while hot; pairs well with cola or mineral water from the bar nearby.',
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
    'Extra cheese',
    'Generous portion of crispy tortilla chips topped with warm melted cheese, jalapeño slices, fresh salsa, and a dollop of sour cream. Built for sharing but often finished solo during tense second halves. Vegetarian-friendly base; add extra cheese or peppers on request at the counter while stocks last.',
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
    'Large bucket',
    'Large bucket of freshly popped corn seasoned with fine sea salt — light, crunchy, and perfect for nibbling through the full ninety minutes. Popped on site throughout the day for maximum freshness. No butter added by default; request a butter topping at the kiosk if you prefer a richer flavour.',
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
    'Chilled 50 cl sugar-free cola served ice-cold from the fridge unit. Zero sugar and zero calories with the familiar cola taste supporters expect on a long match day. Sealed bottle with twist cap — easy to carry back to your seat without spills in the stands.',
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
    'Still natural mineral water in a 50 cl recyclable bottle. Clean, refreshing, and ideal for hot afternoon fixtures or as a mixer-free option for all ages. Stored chilled; grab one on your way to your block or keep it in your bag for the journey home after the final whistle.',
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
    'Size 1 souvenir football with soft touch foam core — safe for indoor play and young fans learning their first keepy-uppies. Printed with club colours and badge; includes a small display stand hook on the packaging. A popular pocket-money purchase from the fan store on family match days.',
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
    'Knitted acrylic supporter scarf in home club colours with fringe ends and woven badge at the centre. Long enough to hold aloft during chants and warm enough for winter evening fixtures. Double-sided pattern visible from both ends of the stand; a classic match-day essential for loyal supporters.',
    'stickers',
    2200, 2200,
    'https://loremflickr.com/400/400/scarf,football/all',
    'Fan Shop',
    '["S","M","L"]'::jsonb,
    true,
    NOW() - INTERVAL '2 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000015',
    'food',
    'Chicken Wrap',
    'Grilled fresh',
    'Grilled chicken strips wrapped in a soft flour tortilla with mixed salad, tomato, cucumber, and garlic-yogurt sauce. Balanced and portable — easier to eat in your seat than a plate meal. Prepared fresh at the counter; hold upright to keep the filling inside on the walk back from the kiosk.',
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
    'Stone baked',
    'Stone-baked slice with tomato base, mozzarella, roasted peppers, red onion, and Italian herbs. Crisp base with a chewy rim, reheated to order so the cheese melts right when you collect. Vegetarian; one slice is a solid snack, two slices make a proper halftime meal with a drink.',
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
    'Large carton of golden fries cut from fresh potatoes, fried until crisp outside and fluffy inside, finished with sea salt. Served in a cardboard tray with a dip slot — add ketchup or mayo from the condiment station. Best eaten within ten minutes while still hot and crunchy.',
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
    '33 cl citrus energy drink served cold with caffeine and B vitamins for an extra boost during evening kick-offs. Sweet-tasting and carbonated; not recommended for children or anyone sensitive to caffeine. Drink responsibly and balance with water on hot match days.',
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
    'Freshly squeezed orange juice with no added sugar — bright, tangy, and a lighter alternative to soft drinks. Served over ice in a cup with lid for carry-back to your seat. Popular with families; contains natural fruit sugars and vitamin C.',
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
    'Triple layer',
    'Triple-layer toasted sandwich with grilled chicken, crispy bacon, lettuce, tomato, and mayonnaise on white bread. Cut diagonally and boxed for easy handling in the stands. Hearty enough to replace a full meal if you arrived straight from work without dinner.',
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
    'Six oven-roasted chicken wings glazed in smoky barbecue sauce with a sticky, tangy finish. Served in a box with napkins included — expect messy fingers and happy faces. Mild spice level suitable for most palates; ask for extra sauce on the side if you like them extra glossy.',
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
    'Dark chocolate',
    'Soft bakery muffin loaded with dark chocolate chips and a domed top that stays moist until full time. Individually wrapped for hygiene and easy storage in a bag if you want to save it for after the match. Pairs perfectly with iced coffee from the same kiosk.',
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
    'Cold brew',
    'Cold brew coffee poured over ice with a splash of milk — smooth, slightly sweet, and refreshing during warm afternoon fixtures. Served in a cup with lid and straw. Contains caffeine; not suitable for young children unless decaf is requested (subject to availability).',
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
    'Match-grade goalkeeper gloves with latex palm for superior grip in wet and dry conditions. Finger spines support hand shape on hard shots; elastic wrist strap keeps the glove secure during dives. Used by academy keepers and Sunday league players alike — rinse after use and air dry.',
    'sport-suits',
    4200, 4200,
    'https://loremflickr.com/400/400/gloves,football/all',
    'Sport Mall 2',
    '["S","M","L","XL"]'::jsonb,
    true,
    NOW() - INTERVAL '24 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000025',
    'merch',
    'Captain Armband',
    '',
    'Elasticated captain armband in club colours with Velcro closure for quick changes on the pitch. Fits comfortably over long or short sleeves; high-visibility band so referees and supporters recognise the skipper instantly. One per pack; ideal for five-a-side teams and school tournaments.',
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
    'Adjustable curved-peak cap with embroidered club crest on the front and ventilation eyelets for summer fixtures. Metal buckle closure fits most head sizes; structured front panel keeps shape in a bag. Sun protection for afternoon games and a casual supporter look on travel days.',
    't-shirts',
    1800, 1800,
    'https://loremflickr.com/400/400/cap,sport/all',
    'Fan Shop',
    '["S","M","L","XL","XXL"]'::jsonb,
    true,
    NOW() - INTERVAL '26 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000027',
    'merch',
    'Fan Flag',
    'Large',
    'Large supporter flag in club colours with reinforced stitching along the edge and two attachment loops for poles or railings. Lightweight polyester flies well in the wind during chants and kick-off. Folds into a compact square for away trips; wipe clean with a damp cloth after use.',
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
    'Lightweight training shorts with elastic waistband, internal drawcord, and mesh lining for breathability during summer sessions. Club badge on the left leg; side pockets sized for keys and phone during warm-ups. Quick-dry fabric — wash after every session to maintain freshness.',
    'sport-suits',
    2400, 2400,
    'https://loremflickr.com/400/400/shorts,sport/all',
    'Sport Mall 2',
    '["S","M","L","XL","XXL"]'::jsonb,
    true,
    NOW() - INTERVAL '28 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000029',
    'merch',
    'Socks Pack',
    '3 pairs',
    'Three pairs of match-day football socks with cushioned foot bed, arch support, and ribbed leg for a stay-up fit inside boots. Club colour stripes around the cuff; reinforced toe and heel for durability through a full season. One size range covers youth to adult players.',
    'sport-suits',
    1200, 1200,
    'https://loremflickr.com/400/400/socks,sport/all',
    'Sport Mall 2',
    '["S","M","L","XL","XXL"]'::jsonb,
    true,
    NOW() - INTERVAL '29 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000030',
    'merch',
    'Water Bottle',
    '750 ml',
    'Reusable 750 ml BPA-free water bottle with club logo print and leak-proof flip lid. Fits standard cup holders and backpack side pockets — refill at stadium fountains throughout the day. Dishwasher safe on the top rack; reduce single-use plastic on every visit.',
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
    'Warm baked pretzel twisted by hand, brushed with butter and finished with coarse salt crystals. Soft inside with a chewy crust — classic stadium snack from the central bakery counter. Best served warm within minutes of purchase; mustard dip available on request.',
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
    'Beer battered',
    'Beer-battered cod fillet with thick-cut chips, mushy peas, and a lemon wedge. Fried to order for a crisp coating and flaky fish inside. A hearty British stadium classic — allow a few extra minutes at busy halftimes when the queue is longest.',
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
    'Chilled sparkling mineral water with fine bubbles — refreshing alternative to still water or sugary soft drinks. 50 cl glass bottle with screw cap; ideal for sharing at a table or sipping solo in the stand. Store upright and open carefully to avoid fizz overflow.',
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
    '1998 classic',
    'Classic throwback home jersey inspired by the 1998 promotion-winning season. Relaxed retro cut, bold collar, and woven crest patch in vintage style. Breathable cotton-poly blend for comfort on casual match days and collector display. A conversation starter in the pub before kick-off.',
    't-shirts',
    6200, 6200,
    'https://loremflickr.com/400/400/jersey,vintage/all',
    'Sport Mall 2',
    '["S","M","L","XL","XXL"]'::jsonb,
    true,
    NOW() - INTERVAL '34 minutes'
  ),
  (
    'c2000000-0000-4000-8000-000000000035',
    'merch',
    'Pump Ball',
    'With pump',
    'Size 5 training ball with durable PVC outer and included mini hand pump in the box. Suitable for garden practice, school clubs, and casual kickabouts. Pump clips to the ball bag loop; inflation needle stored inside the pump handle. Not match-grade but excellent value for everyday use.',
    'balls',
    3900, 3900,
    'https://loremflickr.com/400/400/football,training/all',
    'Sport Mall 2',
    '[]'::jsonb,
    true,
    NOW() - INTERVAL '35 minutes'
  )
ON CONFLICT (id) DO UPDATE SET
    subname = EXCLUDED.subname,
    description = EXCLUDED.description,
    available_sizes = EXCLUDED.available_sizes,
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

UPDATE products SET
    subname = 'Home kit',
    description = 'Official club home kit jersey for the current season. Lightweight breathable polyester with moisture-wicking finish and embroidered crest. Available in a full range of adult sizes from XS to 3XL.',
    available_sizes = '["S","M","L","XL","XXL"]'::jsonb,
    updated_at = NOW()
WHERE deleted_at IS NULL AND name = 'Sport T-shirt' AND id::text NOT LIKE 'c2000000%';

UPDATE products SET
    description = 'FIFA-quality match ball built for consistent flight and reliable grip in all weather. Hand-stitched panels with textured surface for improved control during passes and shots.',
    updated_at = NOW()
WHERE deleted_at IS NULL AND name = 'Match Ball' AND id::text NOT LIKE 'c2000000%';

UPDATE products SET
    description = 'Collectible sticker sheet with ten weather-resistant vinyl decals featuring the club badge and stadium silhouette. UV-coated to resist fading.',
    updated_at = NOW()
WHERE deleted_at IS NULL AND name = 'Club Sticker Pack' AND id::text NOT LIKE 'c2000000%';

UPDATE products SET
    description = 'Complete two-piece training tracksuit with zip jacket and tapered joggers in club colours. Available in sizes XS through 3XL for a comfortable fit.',
    available_sizes = '["S","M","L","XL","XXL"]'::jsonb,
    updated_at = NOW()
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
