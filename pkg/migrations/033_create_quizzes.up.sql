-- Migration: 033_create_quizzes
-- Purpose: Mobile Guess module — per-match quizzes with options and one
-- answer per user per quiz.

CREATE TABLE IF NOT EXISTS quizzes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id   UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    title      VARCHAR(255) NOT NULL,
    quiz_type  VARCHAR(30) NOT NULL DEFAULT 'result'
        CHECK (quiz_type IN ('result', 'player', 'custom')),
    points     INTEGER NOT NULL DEFAULT 0 CHECK (points >= 0),
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    created_by UUID,
    updated_by UUID
);

CREATE INDEX IF NOT EXISTS idx_quizzes_match_id   ON quizzes (match_id);
CREATE INDEX IF NOT EXISTS idx_quizzes_deleted_at ON quizzes (deleted_at);

CREATE TABLE IF NOT EXISTS quiz_options (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id    UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    label      VARCHAR(255) NOT NULL,
    value      VARCHAR(255) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT uidx_quiz_options_quiz_value UNIQUE (quiz_id, value)
);

CREATE INDEX IF NOT EXISTS idx_quiz_options_quiz_id ON quiz_options (quiz_id);

CREATE TABLE IF NOT EXISTS quiz_answers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id       UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    choice        VARCHAR(255) NOT NULL,
    points_earned INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT uidx_quiz_answers_quiz_user UNIQUE (quiz_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_quiz_answers_user_id ON quiz_answers (user_id);
