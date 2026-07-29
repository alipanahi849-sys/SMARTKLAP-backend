-- Migration: 028_add_created_at_index_to_client_heartbeats
-- Purpose:
--   Speed up the heartbeat retention cleanup query
--   (DELETE FROM client_heartbeats WHERE created_at < cutoff) which previously
--   required a full table scan.
-- Deliverables: F-030

CREATE INDEX IF NOT EXISTS idx_client_heartbeats_created_at
    ON client_heartbeats (created_at);
