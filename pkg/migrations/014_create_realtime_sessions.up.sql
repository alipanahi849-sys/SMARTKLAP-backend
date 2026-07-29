-- Migration: 014_create_realtime_sessions
-- Purpose: Realtime session lifecycle per match

CREATE TABLE IF NOT EXISTS realtime_sessions (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id    UUID        NOT NULL,
    started_at  TIMESTAMP,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending','running','paused','completed')),
    created_at  TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP   NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMP,
    created_by  UUID
);

-- Only one active session per match at a time.
CREATE UNIQUE INDEX IF NOT EXISTS idx_realtime_sessions_match_active
    ON realtime_sessions (match_id)
    WHERE deleted_at IS NULL AND status IN ('pending','running','paused');

CREATE INDEX IF NOT EXISTS idx_realtime_sessions_match_id
    ON realtime_sessions (match_id);

CREATE INDEX IF NOT EXISTS idx_realtime_sessions_status
    ON realtime_sessions (status);

-- Auto-update updated_at.
CREATE OR REPLACE FUNCTION update_realtime_sessions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_realtime_sessions_updated_at ON realtime_sessions;
CREATE TRIGGER trg_realtime_sessions_updated_at
    BEFORE UPDATE ON realtime_sessions
    FOR EACH ROW EXECUTE FUNCTION update_realtime_sessions_updated_at();
