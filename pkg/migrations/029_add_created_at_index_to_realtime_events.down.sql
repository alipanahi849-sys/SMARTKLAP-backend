-- Migration: 029_add_created_at_index_to_realtime_events (down)

DROP INDEX IF EXISTS idx_realtime_events_created_at;
