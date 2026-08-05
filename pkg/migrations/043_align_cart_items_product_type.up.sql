-- Migration: 043_align_cart_items_product_type
-- Extend legacy cart_items (snack|merch) to also accept food.

ALTER TABLE cart_items DROP CONSTRAINT IF EXISTS cart_items_product_type_check;
ALTER TABLE cart_items ADD CONSTRAINT cart_items_product_type_check
    CHECK (product_type IN ('food', 'snack', 'merch'));

UPDATE cart_items SET product_type = 'food' WHERE product_type = 'snack';

ALTER TABLE cart_items DROP CONSTRAINT IF EXISTS cart_items_product_type_check;
ALTER TABLE cart_items ADD CONSTRAINT cart_items_product_type_check
    CHECK (product_type IN ('food', 'merch'));
