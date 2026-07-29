-- Migration: 031_create_profiles
-- Purpose: The profiles GORM model existed without a backing migration.
-- The mobile Profile endpoints (avatar upload) require this table.

CREATE TABLE IF NOT EXISTS profiles (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    bio           TEXT,
    avatar_url    VARCHAR(500),
    date_of_birth DATE,
    country       VARCHAR(100),
    city          VARCHAR(100),
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_profiles_deleted_at ON profiles (deleted_at);
