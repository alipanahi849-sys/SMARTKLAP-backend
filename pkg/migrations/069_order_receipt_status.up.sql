-- Migration: 069_order_receipt_status
-- Track whether the Stripe invoice/receipt was emailed to the buyer.

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS receipt_status VARCHAR(30) NOT NULL DEFAULT 'none';

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS receipt_sent_at TIMESTAMP;

-- Backfill known payment outcomes (historical Stripe sends are unknown → pending).
UPDATE orders
SET receipt_status = 'not_applicable'
WHERE payment_method = 'points'
  AND status = 'paid'
  AND receipt_status = 'none';

UPDATE orders
SET receipt_status = 'pending'
WHERE payment_method = 'card'
  AND status = 'paid'
  AND receipt_status = 'none';
