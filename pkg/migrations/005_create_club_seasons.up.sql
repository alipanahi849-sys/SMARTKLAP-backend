CREATE TABLE IF NOT EXISTS club_seasons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    club_id UUID NOT NULL,
    season_id UUID NOT NULL,
    joined_at DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'withdrawn')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by UUID,
    updated_by UUID,
    CONSTRAINT club_seasons_club_id_fkey FOREIGN KEY (club_id) REFERENCES clubs(id) ON DELETE CASCADE,
    CONSTRAINT club_seasons_season_id_fkey FOREIGN KEY (season_id) REFERENCES seasons(id) ON DELETE CASCADE,
    CONSTRAINT club_seasons_unique UNIQUE (club_id, season_id)
);

CREATE INDEX idx_club_seasons_club_id ON club_seasons(club_id);
CREATE INDEX idx_club_seasons_season_id ON club_seasons(season_id);
CREATE INDEX idx_club_seasons_status ON club_seasons(status);
CREATE INDEX idx_club_seasons_deleted_at ON club_seasons(deleted_at);

CREATE TRIGGER update_club_seasons_updated_at BEFORE UPDATE ON club_seasons
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
