DROP TRIGGER IF EXISTS trg_realtime_sessions_updated_at ON realtime_sessions;
DROP FUNCTION IF EXISTS update_realtime_sessions_updated_at();
DROP INDEX IF EXISTS idx_realtime_sessions_match_active;
DROP INDEX IF EXISTS idx_realtime_sessions_match_id;
DROP INDEX IF EXISTS idx_realtime_sessions_status;
DROP TABLE IF EXISTS realtime_sessions;
