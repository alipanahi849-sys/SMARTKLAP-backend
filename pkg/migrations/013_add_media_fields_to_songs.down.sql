ALTER TABLE songs
DROP CONSTRAINT IF EXISTS fk_songs_media_file;

DROP INDEX IF EXISTS idx_songs_storage_key;
DROP INDEX IF EXISTS idx_songs_media_file_id;

ALTER TABLE songs
DROP COLUMN IF EXISTS sample_rate,
DROP COLUMN IF EXISTS bitrate,
DROP COLUMN IF EXISTS duration_ms,
DROP COLUMN IF EXISTS file_size,
DROP COLUMN IF EXISTS mime_type,
DROP COLUMN IF EXISTS storage_key,
DROP COLUMN IF EXISTS media_file_id;
