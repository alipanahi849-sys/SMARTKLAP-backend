-- Migration: 027_extend_realtime_events_event_type_constraint
-- Purpose:
--   Extend the realtime_events.event_type CHECK constraint to support the
--   Phase 4.2 dotted event types delivered over WebSocket, while keeping the
--   constraint strict (closed allowlist).
-- Deliverables: F-023

ALTER TABLE realtime_events
    DROP CONSTRAINT IF EXISTS realtime_events_event_type_check;

ALTER TABLE realtime_events
    ADD CONSTRAINT realtime_events_event_type_check
        CHECK (event_type IN (
            -- Phase 4.0 foundation event types
            'song_start',
            'song_stop',
            'lyric_sync',
            'vibrate',
            'flash',
            'timer_sync',
            -- Phase 4.2 realtime delivery event types
            'match.runtime.updated',
            'song.playback.started',
            'song.playback.cancelled',
            'lyrics.line.changed',
            'server.notification'
        ));
