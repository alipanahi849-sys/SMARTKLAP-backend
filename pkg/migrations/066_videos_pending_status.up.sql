-- Migration: 066_videos_pending_status
-- Videos must be approved in admin before appearing in the public feed.

ALTER TABLE videos DROP CONSTRAINT IF EXISTS videos_status_check;
ALTER TABLE videos
    ADD CONSTRAINT videos_status_check
    CHECK (status IN ('pending', 'processing', 'published', 'rejected'));

ALTER TABLE videos ALTER COLUMN status SET DEFAULT 'pending';
