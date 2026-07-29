-- Migration: 017_create_match_runtime_states
-- Purpose: Durable match timer state — survives server restarts

CREATE TABLE IF NOT EXISTS match_runtime_states (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id        UUID        NOT NULL UNIQUE REFERENCES matches(id) ON DELETE CASCADE,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','running','paused','ended')),
    started_at      TIMESTAMP,
    paused_at       TIMESTAMP,
    ended_at        TIMESTAMP,
    total_paused_ms BIGINT      NOT NULL DEFAULT 0
                    CHECK (total_paused_ms >= 0),
    created_at      TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP   NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP,
    created_by      UUID,
    updated_by      UUID
);

CREATE INDEX IF NOT EXISTS idx_match_runtime_states_match_id
    ON match_runtime_states (match_id);

CREATE INDEX IF NOT EXISTS idx_match_runtime_states_status
    ON match_runtime_states (status);

-- Auto-update updated_at.
CREATE OR REPLACE FUNCTION update_match_runtime_states_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_match_runtime_states_updated_at ON match_runtime_states;
CREATE TRIGGER trg_match_runtime_states_updated_at
    BEFORE UPDATE ON match_runtime_states
    FOR EACH ROW EXECUTE FUNCTION update_match_runtime_states_updated_at();
