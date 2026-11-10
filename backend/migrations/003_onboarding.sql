-- Страница «Мой путь»: шаблоны адаптации (их редактирует наставник/руководитель в консоли)
-- и личный прогресс стажера.

-- Срок стажировки стажера: от него считаются «Неделя N из 24» и сроки задач.
CREATE TABLE IF NOT EXISTS intern_profiles (
    user_id        UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    start_date     DATE NOT NULL,
    duration_weeks INT  NOT NULL DEFAULT 24 CHECK (duration_weeks BETWEEN 1 AND 104)
);

-- Шаблон: этапы адаптации команды. Номер этапа (01, 02...) берется из порядка position.
CREATE TABLE IF NOT EXISTS onboarding_stages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id     UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    position    INT  NOT NULL,
    title       TEXT NOT NULL,
    color       TEXT NOT NULL DEFAULT '#C9FFBF',
    tip         TEXT NOT NULL DEFAULT '',
    suggestions TEXT[] NOT NULL DEFAULT '{}',   -- быстрые вопросы к AI-buddy на этом этапе
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_stages_team ON onboarding_stages (team_id, position);

-- Шаблон: задачи этапа.
-- action_kind – что умеет сделать AI-buddy по кнопке «Сделать с AI-buddy» (например, access_request).
CREATE TABLE IF NOT EXISTS onboarding_tasks (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stage_id       UUID NOT NULL REFERENCES onboarding_stages (id) ON DELETE CASCADE,
    position       INT  NOT NULL,
    title          TEXT NOT NULL,
    note           TEXT NOT NULL DEFAULT '',
    due_day        INT  NULL CHECK (due_day IS NULL OR due_day >= 1),  -- день стажировки, к которому надо сделать
    action_kind    TEXT NULL,
    action_payload JSONB NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_tasks_stage ON onboarding_tasks (stage_id, position);

-- Шаблон: карточки справа от задач этапа (люди, доступы, инструменты, материалы).
CREATE TABLE IF NOT EXISTS onboarding_resources (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stage_id UUID NOT NULL REFERENCES onboarding_stages (id) ON DELETE CASCADE,
    position INT  NOT NULL,
    kind     TEXT NOT NULL CHECK (kind IN ('person', 'access', 'tool', 'article', 'link')),
    name     TEXT NOT NULL,
    note     TEXT NOT NULL DEFAULT '',
    tag      TEXT NOT NULL DEFAULT '',
    tag_kind TEXT NOT NULL DEFAULT 'none' CHECK (tag_kind IN ('ok', 'wait', 'none', 'mentor')),
    -- если задано, тег считается из статуса этой задачи у конкретного стажера
    -- (например, «GitLab» → «не запрошен» / «на согласовании» / «активен»)
    task_id  UUID NULL REFERENCES onboarding_tasks (id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_resources_stage ON onboarding_resources (stage_id, position);

-- Личный прогресс. Нет строки = задача «впереди» (todo).
CREATE TABLE IF NOT EXISTS intern_task_progress (
    intern_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    task_id    UUID NOT NULL REFERENCES onboarding_tasks (id) ON DELETE CASCADE,
    status     TEXT NOT NULL CHECK (status IN ('todo', 'in_progress', 'waiting', 'done')),
    note       TEXT NULL,          -- личная пометка, заменяет note из шаблона («Ждет подтверждения Олега»)
    updated_by UUID NULL REFERENCES users (id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (intern_id, task_id)
);

-- Встречи стажера с наставником (карточка «Встреча 1:1» на главной).
CREATE TABLE IF NOT EXISTS meetings (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    intern_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    host_id    UUID NOT NULL REFERENCES users (id),
    title      TEXT NOT NULL,
    starts_at  TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_meetings_intern ON meetings (intern_id, starts_at);

-- Вехи на шкале «Путь стажировки» (Старт, Первые задачи, Отзыв...).
CREATE TABLE IF NOT EXISTS track_milestones (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    week    INT  NOT NULL CHECK (week >= 1),
    title   TEXT NOT NULL,
    UNIQUE (team_id, week)
);
