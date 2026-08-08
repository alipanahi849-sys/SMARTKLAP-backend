-- Migration: 050_drop_stripe_from_orders
-- Remove Stripe-specific schema now that only points payment is supported.

DROP TABLE IF EXISTS stripe_payment_events;

DROP INDEX IF EXISTS idx_orders_stripe_payment_intent_id;

ALTER TABLE orders DROP COLUMN IF EXISTS stripe_payment_intent_id;
