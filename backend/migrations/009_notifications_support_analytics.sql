-- 009. Уведомления, настройки, поддержка, события для метрик, outbox, feature flags.

CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT NOT NULL REFERENCES notification_kinds (code),
    text       TEXT NOT NULL,
    link       TEXT NULL,                           -- путь во фронте: /requests, /chat...
    change_id  BIGINT NULL REFERENCES template_changes (id) ON DELETE SET NULL,
    request_id UUID NULL REFERENCES requests (id) ON DELETE CASCADE,
    meeting_id UUID NULL REFERENCES meetings (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at    TIMESTAMPTZ NULL
);
CREATE INDEX idx_notifications_unread ON notifications (user_id, created_at DESC) WHERE read_at IS NULL;

-- Настройки из профиля. Хранятся на сервере, потому что стиль ответа нужен помощнику.
CREATE TABLE user_settings (
    user_id      UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    answer_style TEXT NOT NULL DEFAULT 'normal' CHECK (answer_style IN ('short', 'normal', 'detailed'))
);
CREATE TABLE notification_prefs (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind    TEXT NOT NULL REFERENCES notification_kinds (code),
    enabled BOOLEAN NOT NULL,
    PRIMARY KEY (user_id, kind)
);

-- Обращение в поддержку («Сообщить о проблеме»), можно с привязкой к неверному ответу.
CREATE TABLE support_tickets (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number       BIGINT GENERATED ALWAYS AS IDENTITY (START WITH 501) UNIQUE,
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    problem_type TEXT NOT NULL REFERENCES support_problem_types (code),
    text         TEXT NOT NULL CHECK (length(text) BETWEEN 1 AND 2000),
    message_id   UUID NULL REFERENCES messages (id) ON DELETE SET NULL,
    status       TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'closed')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Продуктовые события (имитация Clickstream). Только добавление; props – свободные атрибуты события.
CREATE TABLE event_types (
    code        TEXT PRIMARY KEY,
    description TEXT NOT NULL
);
CREATE TABLE events (
    id          BIGSERIAL PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    user_id     UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    event_type  TEXT NOT NULL REFERENCES event_types (code),
    entity_kind TEXT NULL,
    entity_id   TEXT NULL,
    props       JSONB NOT NULL DEFAULT '{}'
);
CREATE INDEX idx_events_type_time ON events (event_type, occurred_at);
CREATE INDEX idx_events_user_time ON events (user_id, occurred_at);

-- Transactional outbox: событие пишется в той же транзакции, что и данные,
-- фоновый воркер публикует его в Redis Streams (вместо Kafka) и ставит published_at.
CREATE TABLE outbox (
    id           BIGSERIAL PRIMARY KEY,
    topic        TEXT NOT NULL,                     -- requests, notifications, analytics
    payload      JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ NULL,
    attempts     INT NOT NULL DEFAULT 0,
    last_error   TEXT NULL
);
CREATE INDEX idx_outbox_pending ON outbox (id) WHERE published_at IS NULL;

-- Feature flags и их переопределение для команды.
CREATE TABLE feature_flags (
    key         TEXT PRIMARY KEY,
    enabled     BOOLEAN NOT NULL,
    description TEXT NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE feature_flag_overrides (
    key     TEXT NOT NULL REFERENCES feature_flags (key) ON DELETE CASCADE,
    team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL,
    PRIMARY KEY (key, team_id)
);
