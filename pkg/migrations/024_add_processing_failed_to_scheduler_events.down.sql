DROP INDEX IF EXISTS idx_scheduler_events_processing;

ALTER TABLE scheduler_events
    DROP CONSTRAINT IF EXISTS scheduler_events_status_check;

ALTER TABLE scheduler_events
    ADD CONSTRAINT scheduler_events_status_check
        CHECK (status IN ('pending', 'executed', 'cancelled'));

ALTER TABLE scheduler_events
    DROP COLUMN IF EXISTS fail_reason;
