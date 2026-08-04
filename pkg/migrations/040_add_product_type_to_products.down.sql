ALTER TABLE products DROP CONSTRAINT IF EXISTS products_product_type_check;
ALTER TABLE products DROP COLUMN IF EXISTS product_type;

ALTER TABLE products
    ADD CONSTRAINT products_category_check
        CHECK (category IN ('t-shirts', 'balls', 'stickers', 'sport-suits'));
