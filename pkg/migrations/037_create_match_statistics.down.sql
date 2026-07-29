DROP TABLE IF EXISTS match_timeline_events;
DROP TABLE IF EXISTS match_stats;
DROP TABLE IF EXISTS players;
ALTER TABLE matches DROP COLUMN IF EXISTS current_minute;
ALTER TABLE matches DROP COLUMN IF EXISTS away_score;
ALTER TABLE matches DROP COLUMN IF EXISTS home_score;
