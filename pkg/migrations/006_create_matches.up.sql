CREATE TABLE IF NOT EXISTS matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    league_id UUID NOT NULL,
    season_id UUID NOT NULL,
    home_club_id UUID NOT NULL,
    away_club_id UUID NOT NULL,
    provider VARCHAR(50),
    provider_match_id VARCHAR(100),
    match_datetime TIMESTAMP NOT NULL,
    stadium_name VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'live', 'halftime', 'finished', 'cancelled')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by UUID,
    updated_by UUID,
    CONSTRAINT matches_league_id_fkey FOREIGN KEY (league_id) REFERENCES leagues(id) ON DELETE CASCADE,
    CONSTRAINT matches_season_id_fkey FOREIGN KEY (season_id) REFERENCES seasons(id) ON DELETE CASCADE,
    CONSTRAINT matches_home_club_id_fkey FOREIGN KEY (home_club_id) REFERENCES clubs(id) ON DELETE CASCADE,
    CONSTRAINT matches_away_club_id_fkey FOREIGN KEY (away_club_id) REFERENCES clubs(id) ON DELETE CASCADE,
    CONSTRAINT matches_different_clubs CHECK (home_club_id != away_club_id)
);

CREATE INDEX idx_matches_league_id ON matches(league_id);
CREATE INDEX idx_matches_season_id ON matches(season_id);
CREATE INDEX idx_matches_home_club_id ON matches(home_club_id);
CREATE INDEX idx_matches_away_club_id ON matches(away_club_id);
CREATE INDEX idx_matches_match_datetime ON matches(match_datetime);
CREATE INDEX idx_matches_status ON matches(status);
CREATE INDEX idx_matches_provider ON matches(provider, provider_match_id);
CREATE INDEX idx_matches_deleted_at ON matches(deleted_at);

CREATE TRIGGER update_matches_updated_at BEFORE UPDATE ON matches
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
