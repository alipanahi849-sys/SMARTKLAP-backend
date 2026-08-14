-- Migration: 054_add_google_id_to_users
-- Google Sign-In subject (`sub`) so an existing OTP account can be linked
-- and looked up without colliding on empty values.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS google_id VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_id
    ON users (google_id)
    WHERE google_id IS NOT NULL;
