-- 007. Нормализация таблиц «Моего пути» из 003. Данные переносятся, ничего не теряется.

-- 7.1 Наставник – свойство стажера, а не любого пользователя: переносим в intern_profiles.
-- Составные внешние ключи гарантируют, что профиль только у стажера, а наставник – с ролью mentor.
ALTER TABLE intern_profiles
    ADD COLUMN user_role   TEXT NOT NULL DEFAULT 'intern' CHECK (user_role = 'intern'),
    ADD COLUMN mentor_id   UUID NULL,
    ADD COLUMN mentor_role TEXT NOT NULL DEFAULT 'mentor' CHECK (mentor_role = 'mentor');

INSERT INTO intern_profiles (user_id, start_date, duration_weeks)
SELECT id, current_date, 24 FROM users WHERE role = 'intern'
ON CONFLICT (user_id) DO NOTHING;

UPDATE intern_profiles p SET mentor_id = u.mentor_id FROM users u WHERE u.id = p.user_id;

ALTER TABLE intern_profiles
    ADD CONSTRAINT intern_profiles_user_fk   FOREIGN KEY (user_id, user_role)     REFERENCES users (id, role) ON DELETE CASCADE,
    ADD CONSTRAINT intern_profiles_mentor_fk FOREIGN KEY (mentor_id, mentor_role) REFERENCES users (id, role);
CREATE INDEX idx_intern_profiles_mentor ON intern_profiles (mentor_id);

ALTER TABLE users DROP COLUMN mentor_id;

-- 7.2 Подсказки этапа: массив в ячейке нарушает 1НФ – отдельная таблица с порядком.
CREATE TABLE stage_suggestions (
    stage_id UUID NOT NULL REFERENCES onboarding_stages (id) ON DELETE CASCADE,
    position INT  NOT NULL,
    text     TEXT NOT NULL CHECK (length(text) BETWEEN 1 AND 80),
    PRIMARY KEY (stage_id, position)
);
INSERT INTO stage_suggestions (stage_id, position, text)
SELECT s.id, x.ord, x.q FROM onboarding_stages s, unnest(s.suggestions) WITH ORDINALITY AS x(q, ord);
ALTER TABLE onboarding_stages DROP COLUMN suggestions;
ALTER TABLE onboarding_stages ADD CONSTRAINT stages_color_check CHECK (color ~ '^#[0-9A-Fa-f]{6}$');

-- 7.3 Действие задачи: вместо JSON-полей – строка с типизированными ссылками на справочники.
CREATE TABLE task_actions (
    task_id      UUID PRIMARY KEY REFERENCES onboarding_tasks (id) ON DELETE CASCADE,
    kind         TEXT NOT NULL REFERENCES action_kinds (code),
    service_code TEXT NULL,
    service_role TEXT NULL,
    project_code TEXT NULL,
    leave_type   TEXT NULL REFERENCES leave_types (code),
    question     TEXT NULL,
    article_id   UUID NULL REFERENCES kb_articles (id) ON DELETE CASCADE,
    FOREIGN KEY (service_code, service_role) REFERENCES service_roles (service_code, code),
    FOREIGN KEY (project_code, service_code) REFERENCES projects (code, service_code),
    CHECK (kind <> 'access_request' OR (service_code IS NOT NULL AND service_role IS NOT NULL)),
    CHECK (kind <> 'leave_request'  OR leave_type IS NOT NULL),
    CHECK (kind <> 'ask_buddy'      OR question IS NOT NULL),
    CHECK (kind <> 'open_article'   OR article_id IS NOT NULL)
);
-- Перенос из action_kind/action_payload делается в 011 после заполнения services (seed).

-- 7.4 Карточки боковой панели ссылаются на людей и статьи, а не дублируют их имена.
ALTER TABLE onboarding_resources
    ADD COLUMN user_id    UUID NULL REFERENCES users (id) ON DELETE CASCADE,
    ADD COLUMN article_id UUID NULL REFERENCES kb_articles (id) ON DELETE CASCADE,
    ALTER COLUMN name DROP NOT NULL;
UPDATE onboarding_resources r SET user_id = u.id, name = NULL
FROM onboarding_stages s, users u
WHERE r.kind = 'person' AND s.id = r.stage_id AND u.team_id = s.team_id AND u.name = r.name;
-- Карточки людей, которых нет среди пользователей команды (добавлены вручную), становятся ссылками.
UPDATE onboarding_resources SET kind = 'link' WHERE kind = 'person' AND user_id IS NULL;
ALTER TABLE onboarding_resources ADD CONSTRAINT resources_target_check CHECK (
    (kind = 'person'  AND user_id IS NOT NULL) OR
    (kind = 'article' AND (article_id IS NOT NULL OR name IS NOT NULL)) OR
    (kind IN ('access', 'tool', 'link') AND (name IS NOT NULL))
);

-- 7.5 Встречи: вид, длительность, статус и несколько участников (встреча со всеми стажерами).
ALTER TABLE meetings
    ADD COLUMN kind         TEXT NOT NULL DEFAULT 'one_on_one' REFERENCES meeting_kinds (code),
    ADD COLUMN duration_min INT  NOT NULL DEFAULT 60 CHECK (duration_min BETWEEN 15 AND 480),
    ADD COLUMN status       TEXT NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'cancelled')),
    ADD COLUMN remind_min   INT  NULL CHECK (remind_min IS NULL OR remind_min BETWEEN 5 AND 1440);

CREATE TABLE meeting_participants (
    meeting_id UUID NOT NULL REFERENCES meetings (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    PRIMARY KEY (meeting_id, user_id)
);
INSERT INTO meeting_participants (meeting_id, user_id) SELECT id, intern_id FROM meetings;
DROP INDEX IF EXISTS idx_meetings_intern;
ALTER TABLE meetings DROP COLUMN intern_id;
CREATE INDEX idx_meetings_host_time ON meetings (host_id, starts_at);
CREATE INDEX idx_meeting_participants_user ON meeting_participants (user_id);

-- 7.6 Журнал изменений плана (раздел «План» в кабинете наставника).
CREATE TABLE template_changes (
    id          BIGSERIAL PRIMARY KEY,
    team_id     UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    actor_id    UUID NOT NULL REFERENCES users (id),
    entity_kind TEXT NOT NULL CHECK (entity_kind IN ('stage', 'task', 'resource', 'material', 'meeting')),
    entity_id   UUID NULL,
    action      TEXT NOT NULL CHECK (action IN ('create', 'update', 'delete')),
    summary     TEXT NOT NULL,
    notify      BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_template_changes_team ON template_changes (team_id, created_at DESC);

-- 7.7 Цели стажировки. На 1-й неделе стажер обсуждает с руководителем цели, критерии успеха
-- и OKR команды; на 2-й вносит их в трекер прогресса и дальше отмечает продвижение.
CREATE TABLE team_okrs (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    period  TEXT NOT NULL,                          -- «Q4 2026»
    title   TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 200)
);
CREATE INDEX idx_team_okrs_team ON team_okrs (team_id);

CREATE TABLE intern_goals (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    intern_id    UUID NOT NULL REFERENCES intern_profiles (user_id) ON DELETE CASCADE,
    position     INT  NOT NULL CHECK (position > 0),
    title        TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 120),
    criterion    TEXT NOT NULL CHECK (length(criterion) BETWEEN 1 AND 160),
    due_week     INT  NULL CHECK (due_week BETWEEN 1 AND 52),
    progress_pct INT  NOT NULL DEFAULT 0 CHECK (progress_pct BETWEEN 0 AND 100),
    okr_id       UUID NULL REFERENCES team_okrs (id) ON DELETE SET NULL,
    created_by   UUID NOT NULL REFERENCES users (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (intern_id, position) DEFERRABLE INITIALLY DEFERRED
);

-- История прогресса: наставник видит, кто и когда обновил цель.
CREATE TABLE goal_updates (
    id           BIGSERIAL PRIMARY KEY,
    goal_id      UUID NOT NULL REFERENCES intern_goals (id) ON DELETE CASCADE,
    author_id    UUID NOT NULL REFERENCES users (id),
    progress_pct INT  NOT NULL CHECK (progress_pct BETWEEN 0 AND 100),
    note         TEXT NOT NULL DEFAULT '' CHECK (length(note) <= 500),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_goal_updates_goal ON goal_updates (goal_id, created_at DESC);

-- Согласование целей руководителем: оба поля заполнены или оба пусты.
ALTER TABLE intern_profiles
    ADD COLUMN goals_agreed_by UUID NULL REFERENCES users (id),
    ADD COLUMN goals_agreed_at TIMESTAMPTZ NULL,
    ADD CONSTRAINT intern_profiles_goals_agreed_check CHECK ((goals_agreed_by IS NULL) = (goals_agreed_at IS NULL));
