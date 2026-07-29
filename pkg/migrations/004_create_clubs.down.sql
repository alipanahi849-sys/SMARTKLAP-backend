DROP TRIGGER IF EXISTS update_clubs_updated_at ON clubs;
DROP INDEX IF EXISTS idx_clubs_deleted_at;
DROP INDEX IF EXISTS idx_clubs_name;
DROP INDEX IF EXISTS idx_clubs_is_active;
DROP INDEX IF EXISTS idx_clubs_country;
DROP TABLE IF EXISTS clubs;
