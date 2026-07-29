CREATE TABLE IF NOT EXISTS match_song_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id UUID NOT NULL,
    song_id UUID NOT NULL,
    scheduled_time TIMESTAMP NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by UUID,
    updated_by UUID,
    CONSTRAINT match_song_schedules_match_id_fkey FOREIGN KEY (match_id) REFERENCES matches(id) ON DELETE CASCADE,
    CONSTRAINT match_song_schedules_song_id_fkey FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
);

CREATE INDEX idx_match_song_schedules_match_id ON match_song_schedules(match_id);
CREATE INDEX idx_match_song_schedules_song_id ON match_song_schedules(song_id);
CREATE INDEX idx_match_song_schedules_scheduled_time ON match_song_schedules(scheduled_time);
CREATE INDEX idx_match_song_schedules_event_type ON match_song_schedules(event_type);
CREATE INDEX idx_match_song_schedules_is_active ON match_song_schedules(is_active);
CREATE INDEX idx_match_song_schedules_deleted_at ON match_song_schedules(deleted_at);

CREATE TRIGGER update_match_song_schedules_updated_at BEFORE UPDATE ON match_song_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
