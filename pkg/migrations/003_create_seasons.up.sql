CREATE TABLE IF NOT EXISTS seasons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    league_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by UUID,
    updated_by UUID,
    CONSTRAINT seasons_league_id_fkey FOREIGN KEY (league_id) REFERENCES leagues(id) ON DELETE CASCADE,
    CONSTRAINT seasons_valid_dates CHECK (start_date <= end_date)
);

CREATE INDEX idx_seasons_league_id ON seasons(league_id);
CREATE INDEX idx_seasons_is_active ON seasons(is_active);
CREATE INDEX idx_seasons_start_date ON seasons(start_date);
CREATE INDEX idx_seasons_deleted_at ON seasons(deleted_at);

CREATE TRIGGER update_seasons_updated_at BEFORE UPDATE ON seasons
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
