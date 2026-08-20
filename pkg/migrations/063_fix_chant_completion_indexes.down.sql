ALTER TABLE chant_completions
    ALTER COLUMN song_id DROP NOT NULL;

DROP INDEX IF EXISTS idx_chant_completions_song_id;

-- Restores the pre-062 shape rather than the out-of-band index, which was never
-- part of this repository's schema.
CREATE UNIQUE INDEX IF NOT EXISTS uidx_chant_completions_chant_user
    ON chant_completions (chant_id, user_id)
    WHERE chant_id IS NOT NULL;
