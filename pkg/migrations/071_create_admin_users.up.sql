-- Admin (staff) identity is separate from app (fan) users.
CREATE TABLE IF NOT EXISTS admin_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    is_active BOOLEAN DEFAULT true
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_users_email ON admin_users(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_admin_users_deleted_at ON admin_users(deleted_at);

CREATE TABLE IF NOT EXISTS admin_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    admin_user_id UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    token VARCHAR(500) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    ip_address VARCHAR(45),
    user_agent VARCHAR(500)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_refresh_tokens_token ON admin_refresh_tokens(token) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_admin_refresh_tokens_admin_user_id ON admin_refresh_tokens(admin_user_id);
CREATE INDEX IF NOT EXISTS idx_admin_refresh_tokens_expires_at ON admin_refresh_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_admin_refresh_tokens_revoked_at ON admin_refresh_tokens(revoked_at);
CREATE INDEX IF NOT EXISTS idx_admin_refresh_tokens_deleted_at ON admin_refresh_tokens(deleted_at);

DROP TRIGGER IF EXISTS update_admin_users_updated_at ON admin_users;
CREATE TRIGGER update_admin_users_updated_at
    BEFORE UPDATE ON admin_users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_admin_refresh_tokens_updated_at ON admin_refresh_tokens;
CREATE TRIGGER update_admin_refresh_tokens_updated_at
    BEFORE UPDATE ON admin_refresh_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Migrate existing staff accounts that only had the admin role on the shared users table.
INSERT INTO admin_users (id, created_at, updated_at, email, first_name, last_name, is_active)
SELECT u.id, u.created_at, u.updated_at, u.email, u.first_name, u.last_name, u.is_active
FROM users u
INNER JOIN user_roles ur ON ur.user_id = u.id
INNER JOIN roles r ON r.id = ur.role_id AND r.name = 'admin' AND r.deleted_at IS NULL
WHERE u.deleted_at IS NULL
ON CONFLICT (id) DO NOTHING;

-- Remove the admin role from app users; staff now live in admin_users only.
DELETE FROM user_roles
WHERE role_id IN (SELECT id FROM roles WHERE name = 'admin' AND deleted_at IS NULL);
