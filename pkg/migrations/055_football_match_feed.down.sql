DROP TABLE IF EXISTS app_settings;
DROP TABLE IF EXISTS match_lineup_players;

DROP INDEX IF EXISTS uidx_players_provider;
ALTER TABLE players DROP COLUMN IF EXISTS provider_player_id;
ALTER TABLE players DROP COLUMN IF EXISTS provider;

ALTER TABLE match_timeline_events DROP COLUMN IF EXISTS sub_player_name;

ALTER TABLE matches DROP COLUMN IF EXISTS details_synced_at;
ALTER TABLE matches DROP COLUMN IF EXISTS competition_logo_url;
DROP INDEX IF EXISTS uidx_matches_provider;

DROP INDEX IF EXISTS uidx_seasons_league_provider;
ALTER TABLE seasons DROP COLUMN IF EXISTS provider_season_id;

ALTER TABLE leagues DROP COLUMN IF EXISTS logo_url;

DROP INDEX IF EXISTS uidx_clubs_provider_team;
ALTER TABLE clubs DROP COLUMN IF EXISTS venue_name;
ALTER TABLE clubs DROP COLUMN IF EXISTS provider_team_id;
ALTER TABLE clubs DROP COLUMN IF EXISTS provider;
