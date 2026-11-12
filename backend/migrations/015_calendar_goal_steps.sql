-- 015. Календарь стажера и шаги целей (docs/api-intern-calendar.md).
--
-- Календарь собирается из двух источников: встречи (meetings + meeting_participants, их назначает
-- наставник или руководитель) и собственные записи стажера (calendar_entries). Ответ стажера на
-- встречу – meeting_answers. Отдельного «календаря» не храним: встреча из кабинета наставника сразу
-- видна стажеру, потому что это та же строка meetings.

-- ===================== Календарь =====================

INSERT INTO meeting_kinds (code, title, color) VALUES
    ('lead', 'с руководителем', '#BADAFB'),
    ('intro', 'знакомство', '#F9EEDA');

ALTER TABLE meetings
    ADD COLUMN place TEXT NOT NULL DEFAULT '' CHECK (length(place) <= 120),
    ADD COLUMN note  TEXT NOT NULL DEFAULT '' CHECK (length(note) <= 300);

CREATE TABLE meeting_answers (
    meeting_id UUID NOT NULL REFERENCES meetings (id) ON DELETE CASCADE,
    intern_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    answer     TEXT NOT NULL CHECK (answer IN ('accept', 'move')),
    reason     TEXT NOT NULL DEFAULT '' CHECK (length(reason) <= 200),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (meeting_id, intern_id),
    -- ответить можно только на встречу, где стажер участник
    FOREIGN KEY (meeting_id, intern_id) REFERENCES meeting_participants (meeting_id, user_id) ON DELETE CASCADE
);

CREATE TABLE calendar_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    intern_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title        TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 120),
    kind         TEXT NOT NULL CHECK (kind IN ('focus', 'study', 'personal')),
    starts_at    TIMESTAMPTZ NOT NULL,
    duration_min INT  NOT NULL CHECK (duration_min BETWEEN 15 AND 480),
    shared       BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_calendar_entries_intern ON calendar_entries (intern_id, starts_at);

-- Места и подсказки для встреч из демо-данных.
UPDATE meetings SET place = 'переговорка «Кедр»', note = 'Принеси свой первый MR – разберем комментарии вместе.'
 WHERE id = 'f1000000-0000-4000-8000-000000000001';
UPDATE meetings SET place = 'видеозвонок' WHERE id = 'f1000000-0000-4000-8000-000000000002';
UPDATE meetings SET place = 'большая переговорка, 4 этаж', note = 'Каждый коротко рассказывает, что сделал за неделю.'
 WHERE id = 'f1000000-0000-4000-8000-000000000003';
UPDATE meetings SET kind = 'lead', place = 'кабинет руководителя', note = 'Подумай заранее, чему хочешь научиться за стажировку.'
 WHERE id = 'f1000000-0000-4000-8000-000000000004';

-- Анна: еженедельная 1:1 с наставником и знакомство с DevOps. Даты – от дня применения миграций.
INSERT INTO meetings (id, host_id, title, starts_at, kind, duration_min, remind_min, place, note) VALUES
 ('f1000000-0000-4000-8000-000000000005', 'aaaaaaaa-0000-4000-8000-000000000002', '1:1 с наставником',
  (current_date + 1 + time '15:00') AT TIME ZONE 'Europe/Moscow', 'one_on_one', 45, 60, 'видеозвонок',
  'Подготовь 2–3 вопроса: что застопорилось и что получилось за неделю.'),
 ('f1000000-0000-4000-8000-000000000006', 'aaaaaaaa-0000-4000-8000-000000000008', 'Знакомство с DevOps',
  (current_date + 2 + time '12:00') AT TIME ZONE 'Europe/Moscow', 'intro', 30, 30, 'видеозвонок',
  'Расскажут, как устроены стенды и пайплайны и где смотреть логи.');
INSERT INTO meeting_participants (meeting_id, user_id) VALUES
 ('f1000000-0000-4000-8000-000000000005', 'aaaaaaaa-0000-4000-8000-000000000001'),
 ('f1000000-0000-4000-8000-000000000006', 'aaaaaaaa-0000-4000-8000-000000000001'),
 ('f1000000-0000-4000-8000-000000000006', 'aaaaaaaa-0000-4000-8000-000000000006');

INSERT INTO meeting_answers (meeting_id, intern_id, answer) VALUES
 ('f1000000-0000-4000-8000-000000000003', 'aaaaaaaa-0000-4000-8000-000000000001', 'accept');

INSERT INTO calendar_entries (intern_id, title, kind, starts_at, duration_min, shared) VALUES
 ('aaaaaaaa-0000-4000-8000-000000000001', 'Фокус: задача по auth-service', 'focus',
  (current_date + 1 + time '10:00') AT TIME ZONE 'Europe/Moscow', 120, true),
 ('aaaaaaaa-0000-4000-8000-000000000001', 'Пара в университете', 'study',
  (current_date + 3 + time '09:00') AT TIME ZONE 'Europe/Moscow', 180, true);

-- ===================== Шаги целей =====================
-- Процент цели больше не выставляется руками: это доля выполненных шагов. Шаг-задача выполнен,
-- когда стажер закрыл задачу «Моего пути»; шаг наставника или руководителя подтверждает он сам.
-- Процент не хранится, а считается запросом (intern_goals.progress_pct – только для целей без шагов).

CREATE TABLE goal_steps (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id   UUID NOT NULL REFERENCES intern_goals (id) ON DELETE CASCADE,
    position  INT  NOT NULL CHECK (position > 0),
    title     TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 120),
    source    TEXT NOT NULL CHECK (source IN ('task', 'mentor', 'lead')),
    task_id   UUID NULL REFERENCES onboarding_tasks (id) ON DELETE SET NULL,
    done_at   TIMESTAMPTZ NULL,
    done_by   UUID NULL REFERENCES users (id),
    UNIQUE (goal_id, position),
    CHECK ((done_at IS NULL) = (done_by IS NULL)),
    CHECK (source <> 'task' OR done_at IS NULL)  -- шаг-задача закрывается задачей, а не отметкой
);
CREATE INDEX idx_goal_steps_goal ON goal_steps (goal_id, position);

INSERT INTO goal_steps (goal_id, position, title, source, task_id, done_at, done_by) VALUES
 -- Анна: «Пройти путь задачи до релиза»
 ('f3000000-0000-4000-8000-000000000001', 1, 'Доступ к GitLab', 'task', 'b7000000-0000-4000-8000-000000000204', NULL, NULL),
 ('f3000000-0000-4000-8000-000000000001', 2, 'Рабочее окружение настроено', 'task', 'b7000000-0000-4000-8000-000000000301', NULL, NULL),
 ('f3000000-0000-4000-8000-000000000001', 3, 'Разобраться в код-ревью', 'task', 'b7000000-0000-4000-8000-000000000302', NULL, NULL),
 ('f3000000-0000-4000-8000-000000000001', 4, 'Первый MR прошел ревью', 'mentor', NULL, NULL, NULL),
 ('f3000000-0000-4000-8000-000000000001', 5, 'Merge в main', 'mentor', NULL, NULL, NULL),
 -- Анна: «Разобраться в сервисе авторизации»
 ('f3000000-0000-4000-8000-000000000002', 1, 'Доступ к тестовому стенду', 'task', 'b7000000-0000-4000-8000-000000000203', NULL, NULL),
 ('f3000000-0000-4000-8000-000000000002', 2, 'Разбор архитектуры auth-service с наставником', 'mentor', NULL,
  now() - interval '2 days', 'aaaaaaaa-0000-4000-8000-000000000002'),
 ('f3000000-0000-4000-8000-000000000002', 3, 'Разобраться с CI/CD команды', 'task', 'b7000000-0000-4000-8000-000000000303', NULL, NULL),
 ('f3000000-0000-4000-8000-000000000002', 4, 'Демо сервиса команде', 'lead', NULL, NULL, NULL),
 -- Анна: «Самостоятельно закрывать задачи уровня junior»
 ('f3000000-0000-4000-8000-000000000003', 1, 'Первая задача без подсказок', 'mentor', NULL, NULL, NULL),
 ('f3000000-0000-4000-8000-000000000003', 2, 'Вторая задача без подсказок', 'mentor', NULL, NULL, NULL),
 ('f3000000-0000-4000-8000-000000000003', 3, 'Третья задача без подсказок', 'lead', NULL, NULL, NULL),
 -- Дмитрий: первая цель уже выполнена
 ('f3000000-0000-4000-8000-000000000011', 1, 'Первый MR прошел ревью', 'mentor', NULL,
  now() - interval '10 days', 'aaaaaaaa-0000-4000-8000-000000000002'),
 ('f3000000-0000-4000-8000-000000000011', 2, 'Merge в main', 'mentor', NULL,
  now() - interval '8 days', 'aaaaaaaa-0000-4000-8000-000000000002');
