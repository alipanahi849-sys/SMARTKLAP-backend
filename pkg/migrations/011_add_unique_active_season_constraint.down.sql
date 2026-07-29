-- Remove partial unique index for active seasons per league
DROP INDEX IF EXISTS idx_seasons_league_id_active_unique;
