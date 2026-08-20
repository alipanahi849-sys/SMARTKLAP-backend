-- Migration: 064_chant_completion_status
-- Purpose: Record that a fan walked out of a live chant instead of finishing it.
--
-- A cancellation is stored as a chant_completions row carrying zero points and
-- status = 'cancelled'. Reusing this table is deliberate: the partial unique
-- indexes from 062 are on (user_id, chant_id) and (user_id, song_id), so the
-- cancelled row takes the slot and a later attempt hits ON CONFLICT DO NOTHING.
-- That is exactly the rule we want — leaving burns the chant — without a second
-- table or an extra guard in the award path. It also puts cancellations on the
-- Home scoreboard for free, since that feed already reads this table.

ALTER TABLE chant_completions
    ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'completed';

ALTER TABLE chant_completions
    DROP CONSTRAINT IF EXISTS chant_completions_status_check;

ALTER TABLE chant_completions
    ADD CONSTRAINT chant_completions_status_check
    CHECK (status IN ('completed', 'cancelled'));
