-- Migration: 022_add_duration_version_to_playback_schedules
-- Purpose:
--   duration_ms: enables overlap detection across playback windows.
--   version:     enables optimistic concurrency control.

ALTER TABLE playback_schedules
    ADD COLUMN IF NOT EXISTS duration_ms BIGINT NOT NULL DEFAULT 0
        CHECK (duration_ms >= 0),
    ADD COLUMN IF NOT EXISTS version     BIGINT NOT NULL DEFAULT 0;

-- Index to speed up the overlap-detection query which filters on
-- (match_id, scheduled_at, duration_ms) for non-cancelled, non-deleted rows.
CREATE INDEX IF NOT EXISTS idx_playback_schedules_overlap
    ON playback_schedules (match_id, scheduled_at, duration_ms)
    WHERE deleted_at IS NULL AND status NOT IN ('cancelled');
