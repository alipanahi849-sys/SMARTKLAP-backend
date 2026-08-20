-- Migration: 059_add_zone_and_numeric_seat
-- Store stadium zone separately and keep seat_number as an integer.

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS zone VARCHAR(50) NOT NULL DEFAULT '';

UPDATE orders
SET
    zone = TRIM(SPLIT_PART(seat_number, ' / ', 1)),
    seat_number = TRIM(SPLIT_PART(seat_number, ' / ', 2))
WHERE position(' / ' IN seat_number) > 0;

ALTER TABLE orders
    ALTER COLUMN seat_number DROP DEFAULT;

ALTER TABLE orders
    ALTER COLUMN seat_number DROP NOT NULL;

ALTER TABLE orders
    ALTER COLUMN seat_number TYPE INTEGER
    USING (
        CASE
            WHEN TRIM(seat_number) ~ '^[0-9]+$' THEN TRIM(seat_number)::INTEGER
            ELSE NULL
        END
    );
