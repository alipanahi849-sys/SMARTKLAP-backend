ALTER TABLE songs
ADD COLUMN media_file_id UUID,
ADD COLUMN storage_key VARCHAR(500),
ADD COLUMN mime_type VARCHAR(100),
ADD COLUMN file_size BIGINT,
ADD COLUMN duration_ms BIGINT,
ADD COLUMN bitrate INTEGER,
ADD COLUMN sample_rate INTEGER;

CREATE INDEX idx_songs_media_file_id ON songs(media_file_id);
CREATE INDEX idx_songs_storage_key ON songs(storage_key);

ALTER TABLE songs
ADD CONSTRAINT fk_songs_media_file
FOREIGN KEY (media_file_id) REFERENCES media_files(id) ON DELETE SET NULL;
