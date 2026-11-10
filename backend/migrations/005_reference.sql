-- 005. Справочники. Значения, у которых есть свои атрибуты (название, цвет, правила),
-- выносим в отдельные таблицы, а в рабочих таблицах ссылаемся на них внешним ключом.

CREATE EXTENSION IF NOT EXISTS vector;

-- Роли пользователей (вместо CHECK в users).
CREATE TABLE roles (
    code  TEXT PRIMARY KEY,
    title TEXT NOT NULL
);
INSERT INTO roles (code, title) VALUES
    ('intern', 'стажер'), ('mentor', 'наставник'), ('lead', 'руководитель команды'),
    ('staff', 'сотрудник');                         -- согласующие вне команды (DevOps); в выборе тестовых пользователей не показываются

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_fk FOREIGN KEY (role) REFERENCES roles (code);
-- Пара (id, role) уникальна: на нее ссылаются таблицы, которым нужен пользователь конкретной роли.
ALTER TABLE users ADD CONSTRAINT users_id_role_key UNIQUE (id, role);
-- В команде один руководитель.
CREATE UNIQUE INDEX users_one_lead_per_team ON users (team_id) WHERE role = 'lead';

-- Уровни прав из ответа организаторов: чтение, редактирование, администрирование.
CREATE TABLE access_levels (
    code  TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    rank  INT  NOT NULL UNIQUE
);
INSERT INTO access_levels (code, title, rank) VALUES
    ('read', 'чтение', 1), ('edit', 'редактирование', 2), ('admin', 'администрирование', 3);

-- Корпоративные сервисы, к которым стажер запрашивает доступ (имитация).
-- Согласующий зависит от ресурса: auto – выдается сразу, lead – руководитель команды стажера,
-- project_owner – владелец проекта (projects.owner_id), devops – сотрудник из approver_id.
CREATE TABLE services (
    code          TEXT PRIMARY KEY,                 -- gitlab, test_stand, prod_logs...
    title         TEXT NOT NULL,
    request_title TEXT NOT NULL,                    -- «Доступ к тестовому стенду» (готовая фраза для заявки)
    description   TEXT NOT NULL DEFAULT '',
    approver_kind TEXT NOT NULL CHECK (approver_kind IN ('auto', 'lead', 'project_owner', 'devops')),
    approver_id   UUID NULL REFERENCES users (id),
    sla_hours     INT  NOT NULL DEFAULT 24 CHECK (sla_hours > 0),
    CHECK (approver_kind = 'devops' OR approver_id IS NULL)
);

-- Роли внутри сервиса (Developer, чтение...) и их уровень прав.
CREATE TABLE service_roles (
    service_code TEXT NOT NULL REFERENCES services (code) ON DELETE CASCADE,
    code         TEXT NOT NULL,
    title        TEXT NOT NULL,
    access_level TEXT NOT NULL REFERENCES access_levels (code),
    PRIMARY KEY (service_code, code)
);

-- Проекты внутри сервиса (репозиторий в GitLab, стенд). Владелец проекта согласует доступ.
CREATE TABLE projects (
    code         TEXT PRIMARY KEY,                  -- backend/auth-service, auth-staging
    service_code TEXT NOT NULL REFERENCES services (code) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    team_id      UUID NULL REFERENCES teams (id) ON DELETE SET NULL,
    owner_id     UUID NULL REFERENCES users (id),
    UNIQUE (code, service_code)                     -- для составного ключа из заявок и действий
);
CREATE INDEX idx_projects_service ON projects (service_code);

-- Кому какую роль в сервисе можно выдавать. Нет строки – нельзя.
CREATE TABLE service_access_policies (
    service_code TEXT NOT NULL,
    service_role TEXT NOT NULL,
    user_role    TEXT NOT NULL REFERENCES roles (code),
    allowed      BOOLEAN NOT NULL,
    reason       TEXT NOT NULL DEFAULT '',          -- показывается, если нельзя
    PRIMARY KEY (service_code, service_role, user_role),
    FOREIGN KEY (service_code, service_role) REFERENCES service_roles (service_code, code) ON DELETE CASCADE
);

-- Действия AI-buddy (кнопка «Сделать с AI-buddy»).
CREATE TABLE action_kinds (
    code               TEXT PRIMARY KEY,
    title              TEXT NOT NULL,
    needs_confirmation BOOLEAN NOT NULL               -- действие меняет данные – спросить стажера
);
INSERT INTO action_kinds (code, title, needs_confirmation) VALUES
    ('access_request', 'заявка на доступ', true),
    ('leave_request',  'заявка на отгул',  true),
    ('ask_buddy',      'вопрос помощнику', false),
    ('open_article',   'открыть статью',   false);

CREATE TABLE leave_types (
    code  TEXT PRIMARY KEY,
    title TEXT NOT NULL
);
INSERT INTO leave_types (code, title) VALUES
    ('day_off', 'отгул'), ('day_off_unpaid', 'отгул за свой счет'), ('study', 'учебный день');

CREATE TABLE request_types (
    code  TEXT PRIMARY KEY,
    title TEXT NOT NULL
);
INSERT INTO request_types (code, title) VALUES
    ('access', 'доступ'), ('leave', 'отгул'), ('other', 'другое');

CREATE TABLE meeting_kinds (
    code  TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    color TEXT NOT NULL CHECK (color ~ '^#[0-9A-Fa-f]{6}$')
);
INSERT INTO meeting_kinds (code, title, color) VALUES
    ('one_on_one', '1:1', '#DFCDF3'), ('review', 'разбор', '#BADAFB'), ('team', 'команда', '#C9FFBF');

CREATE TABLE notification_kinds (
    code  TEXT PRIMARY KEY,
    title TEXT NOT NULL
);
INSERT INTO notification_kinds (code, title) VALUES
    ('tasks', 'изменения задач и плана'), ('requests', 'статусы заявок'),
    ('meetings', 'встречи'), ('materials', 'новые материалы'),
    ('mentor', 'ответы наставника'), ('goals', 'цели стажировки'), ('tips', 'советы помощника');

CREATE TABLE support_problem_types (
    code  TEXT PRIMARY KEY,
    title TEXT NOT NULL
);
INSERT INTO support_problem_types (code, title) VALUES
    ('wrong_answer', 'Неверный ответ помощника'), ('request_failed', 'Не создалась заявка'),
    ('no_access', 'Нет нужного доступа'), ('other', 'Другое');
