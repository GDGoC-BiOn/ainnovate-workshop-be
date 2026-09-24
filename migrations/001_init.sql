-- 001_init.sql
-- arahin-mini database schema
-- Requires PostgreSQL 13+ (uses the built-in gen_random_uuid()).

CREATE TABLE IF NOT EXISTS users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL CHECK (length(trim(name)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS lessons (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title      TEXT NOT NULL CHECK (length(trim(title)) > 0),
    objective  TEXT NOT NULL CHECK (length(trim(objective)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A learner's mastery of a lesson, on a 0..100 scale.
CREATE TABLE IF NOT EXISTS masteries (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lesson_id  UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    score      INTEGER NOT NULL CHECK (score BETWEEN 0 AND 100),
    PRIMARY KEY (user_id, lesson_id)
);

CREATE TABLE IF NOT EXISTS quizzes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lesson_id  UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    difficulty TEXT NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_quizzes_user   ON quizzes(user_id);
CREATE INDEX IF NOT EXISTS idx_quizzes_lesson ON quizzes(lesson_id);

CREATE TABLE IF NOT EXISTS questions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id        UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    question       TEXT NOT NULL CHECK (length(trim(question)) > 0),
    options        JSONB NOT NULL CHECK (jsonb_typeof(options) = 'array' AND jsonb_array_length(options) = 4),
    correct_option INTEGER NOT NULL CHECK (correct_option BETWEEN 0 AND 3),
    position       INTEGER NOT NULL CHECK (position >= 0),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_questions_quiz ON questions(quiz_id);