-- Migration: 063_fix_chant_completion_indexes
-- Purpose: The server picked up an out-of-band "062_chant_song_points" migration
-- that never reached this repository. It left two unique indexes behind that a
-- fresh install does not create, so production and new databases disagreed.
--
-- The harmful one is uidx_chant_completions_song_user: unique on (song_id,
-- user_id) with no WHERE clause, so a song could be earned only once in total.
-- That breaks the intended split, where a song is earned once from the catalog
-- and again when an admin schedules it as an online chant — a unique violation
-- the ON CONFLICT arbiter in the repository cannot swallow, because it names a
-- different index. Migration 062 already installed the two partial indexes that
-- express the rule correctly, so these leftovers are only drift.

-- Drop the constraint before the index: where the name belongs to a table
-- constraint the index only backs it, and DROP INDEX refuses to touch it.
ALTER TABLE chant_completions
    DROP CONSTRAINT IF EXISTS uidx_chant_completions_song_user;

DROP INDEX IF EXISTS uidx_chant_completions_song_user;

-- Same shape as uidx_chant_completions_online for every row that index covers
-- (catalog rows have no chant_id), so this is redundant rather than wrong.
ALTER TABLE chant_completions
    DROP CONSTRAINT IF EXISTS uidx_chant_completions_chant_user;

DROP INDEX IF EXISTS uidx_chant_completions_chant_user;

-- Kept from the out-of-band migration and created here too, so new databases
-- match: CompletedSongIDs looks completions up by song.
CREATE INDEX IF NOT EXISTS idx_chant_completions_song_id
    ON chant_completions (song_id);

-- Production already has song_id NOT NULL while a fresh 062 leaves it nullable.
-- Both sources always write a song, and a nullable column would let a future bug
-- insert NULLs that the partial catalog index treats as distinct — unlimited
-- points. Converge on NOT NULL, but never fail the deploy over it: the API only
-- starts once migrations succeed.
UPDATE chant_completions cc
SET song_id = c.song_id
FROM chants c
WHERE cc.chant_id = c.id
  AND cc.song_id IS NULL;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM chant_completions WHERE song_id IS NULL) THEN
        ALTER TABLE chant_completions ALTER COLUMN song_id SET NOT NULL;
    ELSE
        RAISE WARNING 'chant_completions.song_id left nullable: % row(s) still NULL',
            (SELECT COUNT(*) FROM chant_completions WHERE song_id IS NULL);
    END IF;
END $$;
