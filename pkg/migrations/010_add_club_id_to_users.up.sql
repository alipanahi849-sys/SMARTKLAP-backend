ALTER TABLE users ADD COLUMN club_id UUID REFERENCES clubs(id) ON DELETE SET NULL;

CREATE INDEX idx_users_club_id ON users(club_id);
