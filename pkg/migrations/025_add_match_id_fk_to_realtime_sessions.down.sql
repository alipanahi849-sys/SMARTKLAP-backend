-- Migration: 025_add_match_id_fk_to_realtime_sessions (down)

ALTER TABLE realtime_sessions
    DROP CONSTRAINT IF EXISTS fk_realtime_sessions_match_id;
