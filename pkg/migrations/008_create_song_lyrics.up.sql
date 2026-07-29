CREATE TABLE IF NOT EXISTS song_lyrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    song_id UUID NOT NULL,
    language VARCHAR(10) NOT NULL,
    lyrics TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by UUID,
    updated_by UUID,
    CONSTRAINT song_lyrics_song_id_fkey FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE,
    CONSTRAINT song_lyrics_unique UNIQUE (song_id, language)
);

CREATE INDEX idx_song_lyrics_song_id ON song_lyrics(song_id);
CREATE INDEX idx_song_lyrics_language ON song_lyrics(language);
CREATE INDEX idx_song_lyrics_deleted_at ON song_lyrics(deleted_at);

CREATE TRIGGER update_song_lyrics_updated_at BEFORE UPDATE ON song_lyrics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
