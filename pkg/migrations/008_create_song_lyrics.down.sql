DROP TRIGGER IF EXISTS update_song_lyrics_updated_at ON song_lyrics;
DROP INDEX IF EXISTS idx_song_lyrics_deleted_at;
DROP INDEX IF EXISTS idx_song_lyrics_language;
DROP INDEX IF EXISTS idx_song_lyrics_song_id;
DROP TABLE IF EXISTS song_lyrics;
