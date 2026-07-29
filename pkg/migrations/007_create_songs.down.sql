DROP TRIGGER IF EXISTS update_songs_updated_at ON songs;
DROP INDEX IF EXISTS idx_songs_deleted_at;
DROP INDEX IF EXISTS idx_songs_title;
DROP INDEX IF EXISTS idx_songs_is_active;
DROP INDEX IF EXISTS idx_songs_album;
DROP INDEX IF EXISTS idx_songs_artist;
DROP TABLE IF EXISTS songs;
