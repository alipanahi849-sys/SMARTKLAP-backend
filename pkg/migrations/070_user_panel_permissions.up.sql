-- Migration: 070_user_panel_permissions
-- Section-level access for the admin dashboard (JSON array of section keys).

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS panel_permissions JSONB NOT NULL DEFAULT '[]'::jsonb;

-- Existing full admins keep unrestricted access via the admin role;
-- no backfill of panel_permissions required.
