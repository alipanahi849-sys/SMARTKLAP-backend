-- Migration: 062_chant_catalog_and_points
-- Purpose: The Chants screen now lists the predefined song catalog separately
-- from scheduled online chants. Catalog listens and online chants award
-- different, admin-configurable point values, and each is earned once:
-- once per song for the catalog, once per chant for online.

-- ── completions: one table, two sources ──────────────────────────────────────
ALTER TABLE chant_completions
    ADD COLUMN IF NOT EXISTS song_id UUID REFERENCES songs(id) ON DELETE CASCADE;

ALTER TABLE chant_completions
    ADD COLUMN IF NOT EXISTS source VARCHAR(16) NOT NULL DEFAULT 'online';

-- Catalog completions have no chant row behind them.
ALTER TABLE chant_completions
    ALTER COLUMN chant_id DROP NOT NULL;

UPDATE chant_completions cc
SET song_id = c.song_id
FROM chants c
WHERE cc.chant_id = c.id
  AND cc.song_id IS NULL;

ALTER TABLE chant_completions
    DROP CONSTRAINT IF EXISTS uidx_chant_completions_chant_user;

ALTER TABLE chant_completions
    DROP CONSTRAINT IF EXISTS chant_completions_source_check;

ALTER TABLE chant_completions
    ADD CONSTRAINT chant_completions_source_check
    CHECK (source IN ('catalog', 'online'));

-- An online chant is earned once per chant; a catalog song once per song,
-- no matter how many chants reuse it.
CREATE UNIQUE INDEX IF NOT EXISTS uidx_chant_completions_online
    ON chant_completions (user_id, chant_id)
    WHERE source = 'online';

CREATE UNIQUE INDEX IF NOT EXISTS uidx_chant_completions_catalog
    ON chant_completions (user_id, song_id)
    WHERE source = 'catalog';

CREATE INDEX IF NOT EXISTS idx_chant_completions_created
    ON chant_completions (created_at DESC);

-- ── listen sessions: proof the song was played through ───────────────────────
-- Written when lyrics are fetched; POST /complete rejects awards that arrive
-- sooner than the track could have finished.
CREATE TABLE IF NOT EXISTS chant_listen_sessions (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id    UUID NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    source     VARCHAR(16) NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, song_id, source),
    CONSTRAINT chant_listen_sessions_source_check CHECK (source IN ('catalog', 'online'))
);

-- ── admin-configurable point values ──────────────────────────────────────────
ALTER TABLE app_settings
    ADD COLUMN IF NOT EXISTS chant_song_points INTEGER NOT NULL DEFAULT 100;

ALTER TABLE app_settings
    ADD COLUMN IF NOT EXISTS chant_online_points INTEGER NOT NULL DEFAULT 200;

ALTER TABLE app_settings
    ADD COLUMN IF NOT EXISTS chant_daily_target INTEGER NOT NULL DEFAULT 500;

ALTER TABLE app_settings
    DROP CONSTRAINT IF EXISTS app_settings_chant_points_check;

ALTER TABLE app_settings
    ADD CONSTRAINT app_settings_chant_points_check
    CHECK (chant_song_points >= 0 AND chant_online_points >= 0 AND chant_daily_target > 0);
