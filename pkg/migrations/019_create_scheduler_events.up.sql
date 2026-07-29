-- Migration: 019_create_scheduler_events
-- Purpose: Durable backing store for the in-memory priority queue scheduler.
-- On restart, the queue is re-hydrated from pending records here.

CREATE TABLE IF NOT EXISTS scheduler_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   UUID        NOT NULL,
    event_type   VARCHAR(50) NOT NULL,
    execute_at   TIMESTAMP   NOT NULL,
    payload_json JSONB       NOT NULL DEFAULT '{}',
    status       VARCHAR(20) NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending','executed','cancelled')),
    created_at   TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP   NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scheduler_events_session_id
    ON scheduler_events (session_id);

CREATE INDEX IF NOT EXISTS idx_scheduler_events_execute_at
    ON scheduler_events (execute_at);

-- Primary access pattern: find all pending events due now.
CREATE INDEX IF NOT EXISTS idx_scheduler_events_pending_execute
    ON scheduler_events (execute_at)
    WHERE status = 'pending' AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_scheduler_events_status
    ON scheduler_events (status);

-- Auto-update updated_at.
CREATE OR REPLACE FUNCTION update_scheduler_events_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_scheduler_events_updated_at ON scheduler_events;
CREATE TRIGGER trg_scheduler_events_updated_at
    BEFORE UPDATE ON scheduler_events
    FOR EACH ROW EXECUTE FUNCTION update_scheduler_events_updated_at();
