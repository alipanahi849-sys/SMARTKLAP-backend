-- Migration: 028_add_created_at_index_to_client_heartbeats (down)

DROP INDEX IF EXISTS idx_client_heartbeats_created_at;
