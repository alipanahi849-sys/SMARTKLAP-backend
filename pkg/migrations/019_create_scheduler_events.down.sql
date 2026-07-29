DROP TRIGGER IF EXISTS trg_scheduler_events_updated_at ON scheduler_events;
DROP FUNCTION IF EXISTS update_scheduler_events_updated_at();
DROP INDEX IF EXISTS idx_scheduler_events_status;
DROP INDEX IF EXISTS idx_scheduler_events_pending_execute;
DROP INDEX IF EXISTS idx_scheduler_events_execute_at;
DROP INDEX IF EXISTS idx_scheduler_events_session_id;
DROP TABLE IF EXISTS scheduler_events;
