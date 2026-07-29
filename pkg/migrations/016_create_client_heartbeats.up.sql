-- Migration: 016_create_client_heartbeats
-- Purpose: Records individual client clock-sync round-trips for drift analysis

CREATE TABLE IF NOT EXISTS client_heartbeats (
    id               UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id       UUID      NOT NULL REFERENCES realtime_sessions(id) ON DELETE CASCADE,
    user_id          UUID      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_timestamp BIGINT    NOT NULL,
    server_timestamp BIGINT    NOT NULL,
    drift_ms         BIGINT    NOT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Queries: recent heartbeats per user in a session.
CREATE INDEX IF NOT EXISTS idx_client_heartbeats_session_user
    ON client_heartbeats (session_id, user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_client_heartbeats_session_id
    ON client_heartbeats (session_id);

-- Allows drift aggregation queries.
CREATE INDEX IF NOT EXISTS idx_client_heartbeats_drift
    ON client_heartbeats (session_id, drift_ms);
