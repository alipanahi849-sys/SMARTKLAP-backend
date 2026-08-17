-- Migration: 057_add_apple_id_to_users
-- Sign in with Apple subject (`sub`) so returning users can be found even
-- when Apple omits email on subsequent logins.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS apple_id VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_apple_id
    ON users (apple_id)
    WHERE apple_id IS NOT NULL;
