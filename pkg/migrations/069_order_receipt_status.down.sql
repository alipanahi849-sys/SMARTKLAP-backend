-- Migration: 069_order_receipt_status (down)

ALTER TABLE orders DROP COLUMN IF EXISTS receipt_sent_at;
ALTER TABLE orders DROP COLUMN IF EXISTS receipt_status;
