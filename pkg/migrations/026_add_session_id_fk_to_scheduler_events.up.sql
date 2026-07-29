-- Migration: 026_add_session_id_fk_to_scheduler_events
-- Purpose:
--   Enforce referential integrity between scheduler_events and
--   realtime_sessions. Previously scheduler_events.session_id was an
--   unconstrained UUID, allowing orphaned events and preventing cascade
--   cleanup when a session is deleted.
-- Deliverables: F-024

-- Remove any orphaned scheduler events whose session no longer exists so the
-- FK can be added without violation.
DELETE FROM scheduler_events se
WHERE NOT EXISTS (
    SELECT 1 FROM realtime_sessions rs WHERE rs.id = se.session_id
);

ALTER TABLE scheduler_events
    DROP CONSTRAINT IF EXISTS fk_scheduler_events_session_id;

ALTER TABLE scheduler_events
    ADD CONSTRAINT fk_scheduler_events_session_id
        FOREIGN KEY (session_id) REFERENCES realtime_sessions(id) ON DELETE CASCADE;
