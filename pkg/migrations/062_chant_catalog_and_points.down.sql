-- Migration: 062_chant_catalog_and_points (down)

ALTER TABLE app_settings
    DROP CONSTRAINT IF EXISTS app_settings_chant_points_check;

ALTER TABLE app_settings
    DROP COLUMN IF EXISTS chant_daily_target;

ALTER TABLE app_settings
    DROP COLUMN IF EXISTS chant_online_points;

ALTER TABLE app_settings
    DROP COLUMN IF EXISTS chant_song_points;

DROP TABLE IF EXISTS chant_listen_sessions;

DROP INDEX IF EXISTS idx_chant_completions_created;
DROP INDEX IF EXISTS uidx_chant_completions_catalog;
DROP INDEX IF EXISTS uidx_chant_completions_online;

-- Catalog rows have no chant_id and cannot be represented by the old schema.
DELETE FROM chant_completions WHERE source = 'catalog' OR chant_id IS NULL;

ALTER TABLE chant_completions
    DROP CONSTRAINT IF EXISTS chant_completions_source_check;

ALTER TABLE chant_completions
    ALTER COLUMN chant_id SET NOT NULL;

ALTER TABLE chant_completions
    ADD CONSTRAINT uidx_chant_completions_chant_user UNIQUE (chant_id, user_id);

ALTER TABLE chant_completions
    DROP COLUMN IF EXISTS source;

ALTER TABLE chant_completions
    DROP COLUMN IF EXISTS song_id;
