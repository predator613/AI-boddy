# Инструкция для backend: история диалогов в чате с AI-buddy

Колонка слева в чате (как в макете): прошлые диалоги, сгруппированные по «Сегодня / На этой неделе / Раньше». Прошлый диалог открывается только для чтения. Фронт уже вызывает эти ручки. Пока их нет, список пустой и в колонке виден только «Текущий диалог»: ошибка 404 не показывается, чат работает как раньше. В режиме без backend ручки имитирует `frontend/src/mocks/pending.ts`. Схема ответа – `frontend/src/schemas/chatDialogs.ts`.

## Данные

Сейчас «Новый диалог» (`POST /api/chat/reset`) ставит `conversations.cleared_at`, и старые сообщения просто скрываются. Для истории нужно хранить границы диалогов:

```sql
-- 016_chat_dialog_history.sql
CREATE TABLE chat_dialogs (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  started_at  timestamptz NOT NULL DEFAULT now(),
  closed_at   timestamptz,          -- NULL у текущего диалога
  title       text NOT NULL DEFAULT '',
  kind        text NOT NULL DEFAULT 'answer' CHECK (kind IN ('request','handoff','answer')),
  preview     text NOT NULL DEFAULT ''
);
ALTER TABLE chat_messages ADD COLUMN dialog_id uuid REFERENCES chat_dialogs(id);
```

(Имена таблиц сообщений – как в `012_chat_requests.sql`.)

- `title` – первая реплика стажера, до 60 символов.
- `kind` и `preview` – итог диалога: создана заявка (`request`, «Заявка №1042 создана» / «Заявка одобрена»), вопрос передан наставнику (`handoff`, «Передано наставнику»), иначе `answer` («Ответ из справочника»). Обновлять при каждом ответе помощника.
- `POST /api/chat/reset` закрывает текущий диалог (`closed_at = now()`) и открывает новый. Если в текущем нет сообщений, ничего не делать.

## Ручки (только свой пользователь)

| Метод | Путь | Ответ |
|---|---|---|
| GET | `/api/chat/dialogs` | `[{ id, title, preview, kind, updatedAt, current }]`, новые сверху, до 50 штук; `current: true` у открытого диалога |
| GET | `/api/chat/dialogs/{id}/messages` | `{ "messages": ChatMessage[] }` – тот же формат, что `GET /api/chat/messages`; чужой диалог – 404 |

`GET /api/chat/messages` по-прежнему отдает сообщения текущего диалога. Карточки подтверждения в прошлых диалогах приходят с итоговым `state` (`confirmed` или `cancelled`), фронт их не дает нажать.

## Проверка

1. Войти `intern_anna`, задать пару вопросов, нажать «Новый диалог» – слева появляется диалог с первой репликой.
2. Открыть его – видны сообщения и плашка «Это прошлый диалог», поле ввода скрыто.
3. Создать заявку в чате, начать новый диалог – у прошлого подпись «Заявка №… создана».
