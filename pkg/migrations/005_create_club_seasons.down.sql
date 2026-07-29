DROP TRIGGER IF EXISTS update_club_seasons_updated_at ON club_seasons;
DROP INDEX IF EXISTS idx_club_seasons_deleted_at;
DROP INDEX IF EXISTS idx_club_seasons_status;
DROP INDEX IF EXISTS idx_club_seasons_season_id;
DROP INDEX IF EXISTS idx_club_seasons_club_id;
DROP TABLE IF EXISTS club_seasons;
