-- Migration: 029_add_created_at_index_to_realtime_events
-- Purpose:
--   Support the realtime_events retention cleanup query
--   (DELETE FROM realtime_events WHERE created_at < cutoff) without a full
--   table scan.
-- Deliverables: F-031 (retention performance)

CREATE INDEX IF NOT EXISTS idx_realtime_events_created_at
    ON realtime_events (created_at);
