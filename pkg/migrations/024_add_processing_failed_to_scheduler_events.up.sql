-- Migration: 024_add_processing_failed_to_scheduler_events
-- Purpose:
--   Extend the status enum to support event execution safety:
--     processing – claimed by a single worker via FOR UPDATE SKIP LOCKED
--     failed     – terminal error after processing
--   Also adds fail_reason column for diagnostic storage.

-- Drop the old constraint first, then re-add with the expanded set.
ALTER TABLE scheduler_events
    DROP CONSTRAINT IF EXISTS scheduler_events_status_check;

ALTER TABLE scheduler_events
    ADD CONSTRAINT scheduler_events_status_check
        CHECK (status IN ('pending', 'processing', 'executed', 'cancelled', 'failed'));

ALTER TABLE scheduler_events
    ADD COLUMN IF NOT EXISTS fail_reason TEXT;

-- Index to help workers efficiently find events stuck in processing
-- (useful for a future watchdog that resets timed-out claims).
CREATE INDEX IF NOT EXISTS idx_scheduler_events_processing
    ON scheduler_events (status, updated_at)
    WHERE status = 'processing';
