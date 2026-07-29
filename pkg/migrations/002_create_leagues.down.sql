DROP TRIGGER IF EXISTS update_leagues_updated_at ON leagues;
DROP INDEX IF EXISTS idx_leagues_deleted_at;
DROP INDEX IF EXISTS idx_leagues_is_active;
DROP INDEX IF EXISTS idx_leagues_country;
DROP INDEX IF EXISTS idx_leagues_provider;
DROP TABLE IF EXISTS leagues;
