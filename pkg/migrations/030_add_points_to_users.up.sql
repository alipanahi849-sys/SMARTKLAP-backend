-- Migration: 030_add_points_to_users
-- Purpose: Mobile API — users accumulate points from chants and quizzes.
-- Backward compatible: additive column with a default.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS points INTEGER NOT NULL DEFAULT 0;

-- Leaderboard and rank queries order by points.
CREATE INDEX IF NOT EXISTS idx_users_points ON users (points DESC);
