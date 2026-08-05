ALTER TABLE products DROP CONSTRAINT IF EXISTS products_stock_quantity_check;
ALTER TABLE products DROP COLUMN IF EXISTS stock_quantity;
