DROP INDEX IF EXISTS idx_songs_category;

ALTER TABLE songs
    DROP COLUMN IF EXISTS category;
