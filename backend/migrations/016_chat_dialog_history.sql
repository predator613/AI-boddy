-- 016. История диалогов с AI-buddy (docs/api-chat-dialogs.md).
--
-- Переписка стажера с помощником делится на диалоги. Открыт всегда один (closed_at IS NULL),
-- «Новый диалог» закрывает его и открывает следующий. Заголовок, итог и время последнего
-- сообщения не храним: они считаются по сообщениям диалога (3НФ, нечего синхронизировать).
-- Сообщение попадает в открытый диалог триггером, поэтому все места, где backend пишет в чат
-- (ответ помощника, решение по заявке), работают без изменений.

CREATE TABLE chat_dialogs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    closed_at       TIMESTAMPTZ NULL,
    CHECK (closed_at IS NULL OR closed_at >= started_at)
);
CREATE UNIQUE INDEX chat_dialogs_open_key ON chat_dialogs (conversation_id) WHERE closed_at IS NULL;
CREATE INDEX idx_chat_dialogs_conversation ON chat_dialogs (conversation_id, started_at DESC);

ALTER TABLE messages ADD COLUMN dialog_id UUID NULL REFERENCES chat_dialogs (id) ON DELETE CASCADE;
CREATE INDEX idx_messages_dialog ON messages (dialog_id, created_at);

-- Перенос: то, что было скрыто «Новым диалогом» (до cleared_at), – закрытый диалог, остальное – открытый.
INSERT INTO chat_dialogs (conversation_id, started_at, closed_at)
SELECT c.id, min(m.created_at), c.cleared_at
FROM conversations c JOIN messages m ON m.conversation_id = c.id AND m.created_at <= c.cleared_at
WHERE c.kind = 'buddy' AND c.cleared_at IS NOT NULL
GROUP BY c.id, c.cleared_at;

INSERT INTO chat_dialogs (conversation_id, started_at)
SELECT c.id, COALESCE(c.cleared_at, (SELECT min(created_at) FROM messages WHERE conversation_id = c.id), c.created_at)
FROM conversations c WHERE c.kind = 'buddy';

UPDATE messages m SET dialog_id = d.id
FROM chat_dialogs d
WHERE d.conversation_id = m.conversation_id
  AND m.created_at >= d.started_at AND (d.closed_at IS NULL OR m.created_at <= d.closed_at);
-- сообщения раньше первого диалога (на всякий случай) – в самый ранний диалог
UPDATE messages m SET dialog_id = (
    SELECT id FROM chat_dialogs d WHERE d.conversation_id = m.conversation_id ORDER BY started_at LIMIT 1)
WHERE m.dialog_id IS NULL AND m.conversation_id IN (SELECT id FROM conversations WHERE kind = 'buddy');

ALTER TABLE conversations DROP COLUMN cleared_at;

-- Новое сообщение в чате с помощником – в открытый диалог (если его нет – открываем).
CREATE FUNCTION messages_to_open_dialog() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.dialog_id IS NULL AND EXISTS (SELECT 1 FROM conversations WHERE id = NEW.conversation_id AND kind = 'buddy') THEN
        SELECT id INTO NEW.dialog_id FROM chat_dialogs WHERE conversation_id = NEW.conversation_id AND closed_at IS NULL;
        IF NEW.dialog_id IS NULL THEN
            INSERT INTO chat_dialogs (conversation_id) VALUES (NEW.conversation_id) RETURNING id INTO NEW.dialog_id;
        END IF;
    END IF;
    RETURN NEW;
END $$;

CREATE TRIGGER messages_dialog BEFORE INSERT ON messages
    FOR EACH ROW EXECUTE FUNCTION messages_to_open_dialog();
