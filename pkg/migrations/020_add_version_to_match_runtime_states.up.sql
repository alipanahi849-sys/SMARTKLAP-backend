-- Migration: 020_add_version_to_match_runtime_states
-- Purpose: Adds optimistic concurrency control version column.

ALTER TABLE match_runtime_states
    ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
