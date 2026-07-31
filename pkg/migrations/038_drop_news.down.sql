-- Migration: 038_drop_news (rollback)
-- Recreate the news table if this migration is rolled back.

CREATE TABLE IF NOT EXISTS news (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    club_id      UUID REFERENCES clubs(id) ON DELETE SET NULL,
    title        VARCHAR(500) NOT NULL,
    body         TEXT,
    image_url    VARCHAR(500),
    published_at TIMESTAMP NOT NULL DEFAULT NOW(),
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMP,
    created_by   UUID,
    updated_by   UUID
);

CREATE INDEX IF NOT EXISTS idx_news_published_at ON news (published_at DESC);
CREATE INDEX IF NOT EXISTS idx_news_deleted_at   ON news (deleted_at);
