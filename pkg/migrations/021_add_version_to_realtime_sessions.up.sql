-- Migration: 021_add_version_to_realtime_sessions
-- Purpose: Adds optimistic concurrency control version column.

ALTER TABLE realtime_sessions
    ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
