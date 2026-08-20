-- Migration: 059_add_zone_and_numeric_seat (down)

ALTER TABLE orders
    ALTER COLUMN seat_number TYPE VARCHAR(50)
    USING (COALESCE(seat_number::TEXT, ''));

UPDATE orders
SET seat_number = TRIM(zone) || ' / ' || seat_number
WHERE TRIM(zone) <> '' AND TRIM(seat_number) <> '';

ALTER TABLE orders
    ALTER COLUMN seat_number SET DEFAULT '';

ALTER TABLE orders
    ALTER COLUMN seat_number SET NOT NULL;

ALTER TABLE orders
    DROP COLUMN IF EXISTS zone;
