-- Football provider sync: featured club, provider IDs, match detail extras.

ALTER TABLE clubs ADD COLUMN IF NOT EXISTS provider VARCHAR(50);
ALTER TABLE clubs ADD COLUMN IF NOT EXISTS provider_team_id VARCHAR(100);
ALTER TABLE clubs ADD COLUMN IF NOT EXISTS venue_name VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_clubs_provider_team
    ON clubs (provider, provider_team_id)
    WHERE provider IS NOT NULL
      AND provider_team_id IS NOT NULL
      AND deleted_at IS NULL;

ALTER TABLE leagues ADD COLUMN IF NOT EXISTS logo_url VARCHAR(500);

ALTER TABLE seasons ADD COLUMN IF NOT EXISTS provider_season_id VARCHAR(100);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_seasons_league_provider
    ON seasons (league_id, provider_season_id)
    WHERE provider_season_id IS NOT NULL
      AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uidx_matches_provider
    ON matches (provider, provider_match_id)
    WHERE provider IS NOT NULL
      AND provider_match_id IS NOT NULL
      AND deleted_at IS NULL;

ALTER TABLE matches ADD COLUMN IF NOT EXISTS competition_logo_url VARCHAR(500);
ALTER TABLE matches ADD COLUMN IF NOT EXISTS details_synced_at TIMESTAMP;

ALTER TABLE match_timeline_events ADD COLUMN IF NOT EXISTS sub_player_name VARCHAR(255) NOT NULL DEFAULT '';

ALTER TABLE players ADD COLUMN IF NOT EXISTS provider VARCHAR(50);
ALTER TABLE players ADD COLUMN IF NOT EXISTS provider_player_id VARCHAR(100);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_players_provider
    ON players (provider, provider_player_id)
    WHERE provider IS NOT NULL
      AND provider_player_id IS NOT NULL
      AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS match_lineup_players (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id           UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    club_id            UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
    player_id          UUID REFERENCES players(id) ON DELETE SET NULL,
    side               VARCHAR(10) NOT NULL CHECK (side IN ('home', 'away')),
    name               VARCHAR(255) NOT NULL DEFAULT '',
    position           VARCHAR(50) NOT NULL DEFAULT '',
    jersey_number      INTEGER NOT NULL DEFAULT 0,
    photo_url          VARCHAR(500),
    is_starter         BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order         INTEGER NOT NULL DEFAULT 0,
    created_at         TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_match_lineup_players_match
    ON match_lineup_players (match_id, side, sort_order);

CREATE TABLE IF NOT EXISTS app_settings (
    id                SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    featured_club_id  UUID REFERENCES clubs(id) ON DELETE SET NULL,
    created_at        TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO app_settings (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING;
