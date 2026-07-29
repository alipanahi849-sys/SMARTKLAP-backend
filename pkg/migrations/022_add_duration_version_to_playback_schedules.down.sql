DROP INDEX IF EXISTS idx_playback_schedules_overlap;
ALTER TABLE playback_schedules
    DROP COLUMN IF EXISTS duration_ms,
    DROP COLUMN IF EXISTS version;
