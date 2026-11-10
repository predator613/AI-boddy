-- 008. Заявки и чат с помощником.

-- Заявка. Название («Доступ к GitLab») не храним: оно выводится из типа и деталей.
-- status – текущее состояние; полная история в request_events (обе записи – в одной транзакции).
CREATE TABLE requests (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number        BIGINT GENERATED ALWAYS AS IDENTITY (START WITH 1040) UNIQUE,
    type          TEXT NOT NULL REFERENCES request_types (code),
    requester_id  UUID NOT NULL REFERENCES users (id),
    status        TEXT NOT NULL CHECK (status IN ('draft', 'waiting', 'approved', 'done', 'rejected', 'cancelled')),
    via           TEXT NOT NULL CHECK (via IN ('buddy', 'auto', 'manual')),
    task_id       UUID NULL REFERENCES onboarding_tasks (id) ON DELETE SET NULL,
    approver_id   UUID NULL REFERENCES users (id),
    external_ref  TEXT NULL,                        -- номер во внешнем (mock) сервисе
    reject_reason TEXT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at    TIMESTAMPTZ NULL,
    CHECK (status <> 'rejected' OR reject_reason IS NOT NULL)
);
CREATE INDEX idx_requests_requester ON requests (requester_id, created_at DESC);
CREATE INDEX idx_requests_approver_open ON requests (approver_id) WHERE status = 'waiting';

-- Поля заявки из ответа организаторов: ресурс, проект, уровень прав (через роль), обоснование.
CREATE TABLE access_request_details (
    request_id    UUID PRIMARY KEY REFERENCES requests (id) ON DELETE CASCADE,
    service_code  TEXT NOT NULL,
    service_role  TEXT NOT NULL,
    project_code  TEXT NULL,
    justification TEXT NOT NULL DEFAULT '' CHECK (length(justification) <= 500),
    FOREIGN KEY (service_code, service_role) REFERENCES service_roles (service_code, code),
    FOREIGN KEY (project_code, service_code) REFERENCES projects (code, service_code)
);

CREATE TABLE leave_request_details (
    request_id UUID PRIMARY KEY REFERENCES requests (id) ON DELETE CASCADE,
    leave_date DATE NOT NULL,
    leave_type TEXT NOT NULL REFERENCES leave_types (code),
    comment    TEXT NOT NULL DEFAULT ''
);

CREATE TABLE request_events (
    id         BIGSERIAL PRIMARY KEY,
    request_id UUID NOT NULL REFERENCES requests (id) ON DELETE CASCADE,
    status     TEXT NOT NULL CHECK (status IN ('draft', 'waiting', 'approved', 'done', 'rejected', 'cancelled')),
    actor_kind TEXT NOT NULL CHECK (actor_kind IN ('user', 'buddy', 'service', 'system')),
    actor_id   UUID NULL REFERENCES users (id),
    note       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (actor_kind <> 'user' OR actor_id IS NOT NULL)
);
CREATE INDEX idx_request_events_request ON request_events (request_id, created_at);

-- Переписка: с помощником (kind = buddy) или с наставником (kind = mentor).
CREATE TABLE conversations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind       TEXT NOT NULL CHECK (kind IN ('buddy', 'mentor')),
    intern_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    mentor_id  UUID NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((kind = 'buddy' AND mentor_id IS NULL) OR (kind = 'mentor' AND mentor_id IS NOT NULL))
);
CREATE UNIQUE INDEX conversations_buddy_key  ON conversations (intern_id) WHERE kind = 'buddy';
CREATE UNIQUE INDEX conversations_mentor_key ON conversations (intern_id, mentor_id) WHERE kind = 'mentor';

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    author_kind     TEXT NOT NULL CHECK (author_kind IN ('user', 'assistant', 'mentor', 'system')),
    author_id       UUID NULL REFERENCES users (id),
    kind            TEXT NOT NULL DEFAULT 'text' CHECK (kind IN ('text', 'confirm', 'result', 'error')),
    text            TEXT NOT NULL,
    mood            TEXT NULL CHECK (mood IN ('calm', 'happy', 'think', 'mentor', 'sad')),
    request_id      UUID NULL REFERENCES requests (id) ON DELETE SET NULL, -- карточка подтверждения или результата
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (author_kind NOT IN ('user', 'mentor') OR author_id IS NOT NULL),
    CHECK (kind NOT IN ('confirm', 'result') OR request_id IS NOT NULL)
);
CREATE INDEX idx_messages_conversation ON messages (conversation_id, created_at);

-- Источники ответа (статья, фрагмент или подпись вроде «Мой путь → этап 02»).
CREATE TABLE message_sources (
    message_id UUID NOT NULL REFERENCES messages (id) ON DELETE CASCADE,
    position   INT  NOT NULL,
    article_id UUID NULL REFERENCES kb_articles (id) ON DELETE SET NULL,
    chunk_id   BIGINT NULL REFERENCES kb_chunks (id) ON DELETE SET NULL,
    label      TEXT NULL,
    PRIMARY KEY (message_id, position),
    CHECK (article_id IS NOT NULL OR label IS NOT NULL)
);

-- До какого момента пользователь прочитал переписку. Число непрочитанных считается.
CREATE TABLE conversation_reads (
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    last_read_at    TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (conversation_id, user_id)
);

-- Вопрос, который помощник передал наставнику.
CREATE TABLE handoffs (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id        UUID NOT NULL UNIQUE REFERENCES messages (id) ON DELETE CASCADE, -- вопрос в чате с помощником
    forwarded_id       UUID NOT NULL REFERENCES messages (id) ON DELETE CASCADE,        -- копия в переписке с наставником
    answer_id          UUID NULL REFERENCES messages (id) ON DELETE SET NULL,
    answer_article_id  UUID NULL REFERENCES kb_articles (id) ON DELETE SET NULL,       -- «добавить ответ в справочник»
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    answered_at        TIMESTAMPTZ NULL
);

-- Оценка ответа помощника (метрика качества).
CREATE TABLE message_feedback (
    message_id UUID NOT NULL REFERENCES messages (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    rating     SMALLINT NOT NULL CHECK (rating IN (-1, 1)),
    comment    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (message_id, user_id)
);

-- Журнал вызовов инструментов (MCP) и модели. Сырые входы и выходы протокола – JSONB,
-- это журнал, а не рабочие данные: по нему не строятся связи.
CREATE TABLE agent_tool_calls (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    message_id      UUID NULL REFERENCES messages (id) ON DELETE SET NULL,
    server          TEXT NOT NULL,                  -- mock-services, llm
    tool            TEXT NOT NULL,                  -- access.request, leave.request, chat.completion
    status          TEXT NOT NULL CHECK (status IN ('ok', 'error', 'denied', 'timeout')),
    input           JSONB NOT NULL DEFAULT '{}',
    output          JSONB NOT NULL DEFAULT '{}',
    latency_ms      INT  NOT NULL CHECK (latency_ms >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_agent_tool_calls_time ON agent_tool_calls (created_at);
