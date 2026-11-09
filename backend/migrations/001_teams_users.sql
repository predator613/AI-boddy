-- Команды и пользователи. Авторизация в Авито уже есть, здесь только имитация для прототипа.

CREATE TABLE IF NOT EXISTS teams (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name          TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('intern', 'mentor', 'lead')),
    team_id       UUID NOT NULL REFERENCES teams (id),
    mentor_id     UUID NULL REFERENCES users (id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_team_id ON users (team_id);
CREATE INDEX IF NOT EXISTS idx_users_mentor_id ON users (mentor_id);
