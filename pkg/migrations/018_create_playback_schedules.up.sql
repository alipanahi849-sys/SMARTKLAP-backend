-- Migration: 018_create_playback_schedules
-- Purpose: Song playback event scheduling records (no audio streaming)

CREATE TABLE IF NOT EXISTS playback_schedules (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id     UUID        NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    song_id      UUID        NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    scheduled_at TIMESTAMP   NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending','active','completed','cancelled')),
    created_at   TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP   NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMP,
    created_by   UUID
);

CREATE INDEX IF NOT EXISTS idx_playback_schedules_match_id
    ON playback_schedules (match_id);

CREATE INDEX IF NOT EXISTS idx_playback_schedules_song_id
    ON playback_schedules (song_id);

CREATE INDEX IF NOT EXISTS idx_playback_schedules_scheduled_at
    ON playback_schedules (scheduled_at);

-- Efficient upcoming-songs query.
CREATE INDEX IF NOT EXISTS idx_playback_schedules_match_pending
    ON playback_schedules (match_id, scheduled_at)
    WHERE deleted_at IS NULL AND status = 'pending';

CREATE INDEX IF NOT EXISTS idx_playback_schedules_status
    ON playback_schedules (status);

-- Auto-update updated_at.
CREATE OR REPLACE FUNCTION update_playback_schedules_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_playback_schedules_updated_at ON playback_schedules;
CREATE TRIGGER trg_playback_schedules_updated_at
    BEFORE UPDATE ON playback_schedules
    FOR EACH ROW EXECUTE FUNCTION update_playback_schedules_updated_at();
