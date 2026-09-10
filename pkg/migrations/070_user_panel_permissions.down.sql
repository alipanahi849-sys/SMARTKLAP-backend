-- Migration: 070_user_panel_permissions (down)

ALTER TABLE users DROP COLUMN IF EXISTS panel_permissions;
