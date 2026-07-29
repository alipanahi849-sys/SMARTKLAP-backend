DROP TRIGGER IF EXISTS update_seasons_updated_at ON seasons;
DROP INDEX IF EXISTS idx_seasons_deleted_at;
DROP INDEX IF EXISTS idx_seasons_start_date;
DROP INDEX IF EXISTS idx_seasons_is_active;
DROP INDEX IF EXISTS idx_seasons_league_id;
DROP TABLE IF EXISTS seasons;
