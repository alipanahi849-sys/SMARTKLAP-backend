-- Add partial unique index to ensure only one active season per league
CREATE UNIQUE INDEX idx_seasons_league_id_active_unique ON seasons(league_id) WHERE is_active = true;
