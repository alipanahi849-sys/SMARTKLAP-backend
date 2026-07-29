-- Migration: 036_create_videos
-- Purpose: Mobile Video module — user-generated media feed with likes.

CREATE TABLE IF NOT EXISTS videos (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_type    VARCHAR(10) NOT NULL CHECK (media_type IN ('image', 'video')),
    caption       TEXT,
    -- JSON array of hashtags extracted from the caption, e.g. ["BestPlayer"].
    tags          JSONB NOT NULL DEFAULT '[]'::jsonb,
    storage_key   VARCHAR(500) NOT NULL,
    thumbnail_key VARCHAR(500) NOT NULL DEFAULT '',
    mime_type     VARCHAR(100) NOT NULL,
    file_size     BIGINT NOT NULL DEFAULT 0,
    status        VARCHAR(20) NOT NULL DEFAULT 'published'
        CHECK (status IN ('processing', 'published', 'rejected')),
    likes_count   INTEGER NOT NULL DEFAULT 0 CHECK (likes_count >= 0),
    views_count   INTEGER NOT NULL DEFAULT 0 CHECK (views_count >= 0),
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_videos_user_id    ON videos (user_id);
CREATE INDEX IF NOT EXISTS idx_videos_feed       ON videos (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_videos_deleted_at ON videos (deleted_at);

CREATE TABLE IF NOT EXISTS video_likes (
    video_id   UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY (video_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_video_likes_user_id ON video_likes (user_id);
