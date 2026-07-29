-- Migration: 015_create_realtime_events
-- Purpose: Scheduled realtime events within a session (lyric sync, vibrate, flash, etc.)

CREATE TABLE IF NOT EXISTS realtime_events (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id    UUID        NOT NULL REFERENCES realtime_sessions(id) ON DELETE CASCADE,
    event_type    VARCHAR(50) NOT NULL
                  CHECK (event_type IN ('song_start','song_stop','lyric_sync','vibrate','flash','timer_sync')),
    execute_at_ms BIGINT      NOT NULL,
    payload_json  JSONB       NOT NULL DEFAULT '{}',
    created_at    TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_realtime_events_session_id
    ON realtime_events (session_id);

CREATE INDEX IF NOT EXISTS idx_realtime_events_execute_at_ms
    ON realtime_events (execute_at_ms);

CREATE INDEX IF NOT EXISTS idx_realtime_events_session_execute
    ON realtime_events (session_id, execute_at_ms);

CREATE INDEX IF NOT EXISTS idx_realtime_events_type
    ON realtime_events (event_type);
