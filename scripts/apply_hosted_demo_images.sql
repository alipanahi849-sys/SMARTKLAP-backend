-- Point demo product and news images at files hosted on this API.
-- PUBLIC_BASE is rewritten by the apply script before running.

BEGIN;

UPDATE products SET image_key = CASE name
    WHEN 'Double Burger'       THEN 'PUBLIC_BASE/food/double-burger.jpg'
    WHEN 'Club Hot Dog'        THEN 'PUBLIC_BASE/food/club-hot-dog.jpg'
    WHEN 'Loaded Nachos'       THEN 'PUBLIC_BASE/food/loaded-nachos.jpg'
    WHEN 'Salted Popcorn'      THEN 'PUBLIC_BASE/food/salted-popcorn.jpg'
    WHEN 'Cola Zero'           THEN 'PUBLIC_BASE/food/cola-zero.jpg'
    WHEN 'Mineral Water'       THEN 'PUBLIC_BASE/food/mineral-water.jpg'
    WHEN 'Chicken Wrap'        THEN 'PUBLIC_BASE/food/chicken-wrap.jpg'
    WHEN 'Veggie Pizza Slice'  THEN 'PUBLIC_BASE/food/veggie-pizza.jpg'
    WHEN 'French Fries'        THEN 'PUBLIC_BASE/food/french-fries.jpg'
    WHEN 'Energy Drink'        THEN 'PUBLIC_BASE/food/energy-drink.jpg'
    WHEN 'Orange Juice'        THEN 'PUBLIC_BASE/food/orange-juice.jpg'
    WHEN 'Club Sandwich'       THEN 'PUBLIC_BASE/food/club-sandwich.jpg'
    WHEN 'BBQ Wings'           THEN 'PUBLIC_BASE/food/bbq-wings.jpg'
    WHEN 'Chocolate Muffin'    THEN 'PUBLIC_BASE/food/chocolate-muffin.jpg'
    WHEN 'Iced Coffee'         THEN 'PUBLIC_BASE/food/iced-coffee.jpg'
    WHEN 'Pretzel'             THEN 'PUBLIC_BASE/food/pretzel.jpg'
    WHEN 'Fish & Chips'        THEN 'PUBLIC_BASE/food/fish-chips.jpg'
    WHEN 'Sparkling Water'     THEN 'PUBLIC_BASE/food/sparkling-water.jpg'
    WHEN 'Sport T-shirt'       THEN 'PUBLIC_BASE/merch/sport-tshirt.jpg'
    WHEN 'Away T-shirt'        THEN 'PUBLIC_BASE/merch/away-tshirt.jpg'
    WHEN 'Match Ball'          THEN 'PUBLIC_BASE/merch/match-ball.jpg'
    WHEN 'Club Sticker Pack'   THEN 'PUBLIC_BASE/merch/sticker-pack.jpg'
    WHEN 'Training Suit'       THEN 'PUBLIC_BASE/merch/training-suit.jpg'
    WHEN 'Winter Hoodie'       THEN 'PUBLIC_BASE/merch/winter-hoodie.jpg'
    WHEN 'Mini Ball'           THEN 'PUBLIC_BASE/merch/mini-ball.jpg'
    WHEN 'Scarf'               THEN 'PUBLIC_BASE/merch/scarf.jpg'
    WHEN 'Goalkeeper Gloves'   THEN 'PUBLIC_BASE/merch/gk-gloves.jpg'
    WHEN 'Captain Armband'     THEN 'PUBLIC_BASE/merch/captain-armband.jpg'
    WHEN 'Stadium Cap'         THEN 'PUBLIC_BASE/merch/stadium-cap.jpg'
    WHEN 'Fan Flag'            THEN 'PUBLIC_BASE/merch/fan-flag.jpg'
    WHEN 'Training Shorts'     THEN 'PUBLIC_BASE/merch/training-shorts.jpg'
    WHEN 'Socks Pack'          THEN 'PUBLIC_BASE/merch/socks-pack.jpg'
    WHEN 'Water Bottle'        THEN 'PUBLIC_BASE/merch/water-bottle.jpg'
    WHEN 'Retro Jersey'        THEN 'PUBLIC_BASE/merch/retro-jersey.jpg'
    WHEN 'Pump Ball'           THEN 'PUBLIC_BASE/merch/pump-ball.jpg'
    ELSE image_key
END,
updated_at = NOW()
WHERE deleted_at IS NULL
  AND name IN (
    'Double Burger', 'Club Hot Dog', 'Loaded Nachos', 'Salted Popcorn', 'Cola Zero',
    'Mineral Water', 'Chicken Wrap', 'Veggie Pizza Slice', 'French Fries', 'Energy Drink',
    'Orange Juice', 'Club Sandwich', 'BBQ Wings', 'Chocolate Muffin', 'Iced Coffee',
    'Pretzel', 'Fish & Chips', 'Sparkling Water',
    'Sport T-shirt', 'Away T-shirt', 'Match Ball', 'Club Sticker Pack', 'Training Suit',
    'Winter Hoodie', 'Mini Ball', 'Scarf', 'Goalkeeper Gloves', 'Captain Armband',
    'Stadium Cap', 'Fan Flag', 'Training Shorts', 'Socks Pack', 'Water Bottle',
    'Retro Jersey', 'Pump Ball'
  );

UPDATE news SET image_url = CASE id::text
    WHEN 'd1000000-0000-4000-8000-000000000001' THEN 'PUBLIC_BASE/news/welcome-season.jpg'
    WHEN 'd1000000-0000-4000-8000-000000000002' THEN 'PUBLIC_BASE/news/match-preview.jpg'
    WHEN 'd1000000-0000-4000-8000-000000000003' THEN 'PUBLIC_BASE/news/home-kit.jpg'
    WHEN 'd1000000-0000-4000-8000-000000000004' THEN 'PUBLIC_BASE/news/fan-zone.jpg'
    WHEN 'd1000000-0000-4000-8000-000000000005' THEN 'PUBLIC_BASE/news/player-month.jpg'
    WHEN 'd1000000-0000-4000-8000-000000000006' THEN 'PUBLIC_BASE/news/derby-tickets.jpg'
    WHEN 'd1000000-0000-4000-8000-000000000007' THEN 'PUBLIC_BASE/news/academy-night.jpg'
    WHEN 'd1000000-0000-4000-8000-000000000008' THEN 'PUBLIC_BASE/news/squad-update.jpg'
    WHEN 'd1000000-0000-4000-8000-000000000009' THEN 'PUBLIC_BASE/news/community.jpg'
    WHEN 'd1000000-0000-4000-8000-000000000010' THEN 'PUBLIC_BASE/news/chant-week.jpg'
    WHEN 'd1000000-0000-4000-8000-000000000011' THEN 'PUBLIC_BASE/news/stadium-food.jpg'
    WHEN 'd1000000-0000-4000-8000-000000000012' THEN 'PUBLIC_BASE/news/membership.jpg'
    ELSE image_url
END,
updated_at = NOW()
WHERE deleted_at IS NULL;

COMMIT;
