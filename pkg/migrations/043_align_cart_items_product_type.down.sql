-- No rollback: snack rows were migrated to food.

ALTER TABLE cart_items DROP CONSTRAINT IF EXISTS cart_items_product_type_check;
ALTER TABLE cart_items ADD CONSTRAINT cart_items_product_type_check
    CHECK (product_type IN ('snack', 'merch'));

UPDATE cart_items SET product_type = 'snack' WHERE product_type = 'food';
