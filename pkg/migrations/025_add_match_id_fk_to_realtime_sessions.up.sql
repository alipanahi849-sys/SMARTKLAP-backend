-- Migration: 025_add_match_id_fk_to_realtime_sessions
-- Purpose:
--   Enforce referential integrity between realtime_sessions and matches.
--   Previously realtime_sessions.match_id was an unconstrained UUID, allowing
--   orphaned sessions and preventing cascade cleanup when a match is deleted.
-- Deliverables: F-025

-- Remove any orphaned sessions whose match no longer exists so the FK can be
-- added without violation. Safe: these rows reference non-existent matches.
DELETE FROM realtime_sessions rs
WHERE NOT EXISTS (
    SELECT 1 FROM matches m WHERE m.id = rs.match_id
);

ALTER TABLE realtime_sessions
    DROP CONSTRAINT IF EXISTS fk_realtime_sessions_match_id;

ALTER TABLE realtime_sessions
    ADD CONSTRAINT fk_realtime_sessions_match_id
        FOREIGN KEY (match_id) REFERENCES matches(id) ON DELETE CASCADE;
