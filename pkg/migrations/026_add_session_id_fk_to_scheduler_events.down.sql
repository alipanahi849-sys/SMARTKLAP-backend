-- Migration: 026_add_session_id_fk_to_scheduler_events (down)

ALTER TABLE scheduler_events
    DROP CONSTRAINT IF EXISTS fk_scheduler_events_session_id;
