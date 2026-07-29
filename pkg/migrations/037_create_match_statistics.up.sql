-- Migration: 037_create_match_statistics
-- Purpose: Mobile Statistics module — match scores, per-match stats,
-- timeline events, and player profiles for squads.

-- Additive, nullable score columns: existing rows and API remain valid.
ALTER TABLE matches ADD COLUMN IF NOT EXISTS home_score     INTEGER;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS away_score     INTEGER;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS current_minute VARCHAR(10);

CREATE TABLE IF NOT EXISTS players (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    club_id              UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
    name                 VARCHAR(255) NOT NULL,
    jersey_number        INTEGER NOT NULL DEFAULT 0,
    position             VARCHAR(50) NOT NULL DEFAULT '',
    age                  INTEGER NOT NULL DEFAULT 0,
    preferred_foot       VARCHAR(10) NOT NULL DEFAULT '',
    nationality          VARCHAR(100) NOT NULL DEFAULT '',
    height_cm            INTEGER NOT NULL DEFAULT 0,
    weight_kg            INTEGER NOT NULL DEFAULT 0,
    weak_foot_percentage INTEGER NOT NULL DEFAULT 0,
    photo_url            VARCHAR(500),
    -- JSON array of radar chart entries: [{"label":"Attack","value":50}, ...]
    radar_stats          JSONB NOT NULL DEFAULT '[]'::jsonb,
    formation            VARCHAR(20) NOT NULL DEFAULT '',
    is_active            BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMP,
    created_by           UUID,
    updated_by           UUID
);

CREATE INDEX IF NOT EXISTS idx_players_club_id    ON players (club_id);
CREATE INDEX IF NOT EXISTS idx_players_deleted_at ON players (deleted_at);

CREATE TABLE IF NOT EXISTS match_stats (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id   UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    label      VARCHAR(100) NOT NULL,
    home_value INTEGER NOT NULL DEFAULT 0,
    away_value INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT uidx_match_stats_match_label UNIQUE (match_id, label)
);

CREATE INDEX IF NOT EXISTS idx_match_stats_match_id ON match_stats (match_id);

CREATE TABLE IF NOT EXISTS match_timeline_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id    UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    kind        VARCHAR(10) NOT NULL CHECK (kind IN ('marker', 'event')),
    side        VARCHAR(10) NOT NULL DEFAULT '' CHECK (side IN ('', 'home', 'away')),
    event_type  VARCHAR(30) NOT NULL DEFAULT '',
    player_name VARCHAR(255) NOT NULL DEFAULT '',
    minute      VARCHAR(10) NOT NULL DEFAULT '',
    score       VARCHAR(20) NOT NULL DEFAULT '',
    highlighted BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_match_timeline_events_match
    ON match_timeline_events (match_id, sort_order);
