-- Migration: 042_create_orders (rollback)

DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
