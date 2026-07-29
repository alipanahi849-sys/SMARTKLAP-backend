-- Migration: 027_extend_realtime_events_event_type_constraint (down)
-- Restores the original Phase 4.0 event_type allowlist.

ALTER TABLE realtime_events
    DROP CONSTRAINT IF EXISTS realtime_events_event_type_check;

ALTER TABLE realtime_events
    ADD CONSTRAINT realtime_events_event_type_check
        CHECK (event_type IN ('song_start','song_stop','lyric_sync','vibrate','flash','timer_sync'));
