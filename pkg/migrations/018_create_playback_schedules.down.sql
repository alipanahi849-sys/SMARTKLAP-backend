DROP TRIGGER IF EXISTS trg_playback_schedules_updated_at ON playback_schedules;
DROP FUNCTION IF EXISTS update_playback_schedules_updated_at();
DROP INDEX IF EXISTS idx_playback_schedules_status;
DROP INDEX IF EXISTS idx_playback_schedules_match_pending;
DROP INDEX IF EXISTS idx_playback_schedules_scheduled_at;
DROP INDEX IF EXISTS idx_playback_schedules_song_id;
DROP INDEX IF EXISTS idx_playback_schedules_match_id;
DROP TABLE IF EXISTS playback_schedules;
