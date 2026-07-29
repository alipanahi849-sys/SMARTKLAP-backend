DROP TRIGGER IF EXISTS update_match_song_schedules_updated_at ON match_song_schedules;
DROP INDEX IF EXISTS idx_match_song_schedules_deleted_at;
DROP INDEX IF EXISTS idx_match_song_schedules_is_active;
DROP INDEX IF EXISTS idx_match_song_schedules_event_type;
DROP INDEX IF EXISTS idx_match_song_schedules_scheduled_time;
DROP INDEX IF EXISTS idx_match_song_schedules_song_id;
DROP INDEX IF EXISTS idx_match_song_schedules_match_id;
DROP TABLE IF EXISTS match_song_schedules;
