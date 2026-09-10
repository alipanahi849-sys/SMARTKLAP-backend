DROP TRIGGER IF EXISTS update_admin_refresh_tokens_updated_at ON admin_refresh_tokens;
DROP TRIGGER IF EXISTS update_admin_users_updated_at ON admin_users;

DROP TABLE IF EXISTS admin_refresh_tokens;
DROP TABLE IF EXISTS admin_users;
