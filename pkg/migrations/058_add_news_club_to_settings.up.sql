-- Admin-selected club whose news is shown in the mobile feed.

ALTER TABLE app_settings
    ADD COLUMN IF NOT EXISTS news_club_id UUID REFERENCES clubs(id) ON DELETE SET NULL;
