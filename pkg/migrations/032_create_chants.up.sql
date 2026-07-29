-- Migration: 032_create_chants
-- Purpose: Mobile Chants module — chants are scheduled crowd songs tied to a
-- match; completions award points to users.

CREATE TABLE IF NOT EXISTS chants (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id              UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    song_id               UUID NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    title                 VARCHAR(255) NOT NULL,
    points                INTEGER NOT NULL DEFAULT 0 CHECK (points >= 0),
    duration_seconds      INTEGER NOT NULL DEFAULT 0 CHECK (duration_seconds >= 0),
    scheduled_at          TIMESTAMP NOT NULL,
    flash_duration_ms     INTEGER NOT NULL DEFAULT 500 CHECK (flash_duration_ms >= 0),
    vibration_duration_ms INTEGER NOT NULL DEFAULT 500 CHECK (vibration_duration_ms >= 0),
    is_preview            BOOLEAN NOT NULL DEFAULT FALSE,
    is_active             BOOLEAN NOT NULL DEFAULT TRUE,
    created_at            TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMP,
    created_by            UUID,
    updated_by            UUID
);

CREATE INDEX IF NOT EXISTS idx_chants_match_id     ON chants (match_id);
CREATE INDEX IF NOT EXISTS idx_chants_scheduled_at ON chants (scheduled_at);
CREATE INDEX IF NOT EXISTS idx_chants_deleted_at   ON chants (deleted_at);

-- One completion per user per chant; awards points exactly once.
CREATE TABLE IF NOT EXISTS chant_completions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chant_id      UUID NOT NULL REFERENCES chants(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    points_earned INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT uidx_chant_completions_chant_user UNIQUE (chant_id, user_id)
);

-- "Today's points" queries filter completions by user and day.
CREATE INDEX IF NOT EXISTS idx_chant_completions_user_created
    ON chant_completions (user_id, created_at);
