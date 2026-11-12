# Инструкция для backend: календарь стажера и шаги целей

Фронт уже вызывает эти ручки. Пока их нет в backend, отвечают моки браузера (`VITE_PENDING_MOCKS`, по умолчанию включен):
- календарь – `frontend/src/mocks/mentorData.ts`, массив `calendarHandlers`;
- шаги целей – `frontend/src/mocks/goalsData.ts`.

Когда ручка готова, удалите ее обработчик из `calendarHandlers` и проверьте страницу с `VITE_PENDING_MOCKS=false`. Схемы ответов: `frontend/src/schemas/calendar.ts` и `frontend/src/schemas/goals.ts`. Поля в JSON – camelCase, ошибки – `{ "error": { "code", "message" } }`.

## 1. Календарь стажера (`/calendar`)

Все ручки – только для роли `intern`, наставнику и руководителю отвечать 403. Стажер видит только свои события.

### Что показывает экран

В календаре три вида событий:
- встречи, которые назначил наставник: таблица `meetings` из кабинета наставника, `intern_id` = стажер или `NULL` (встреча для всех стажеров);
- командные и прочие встречи (знакомства, руководитель) – те же `meetings` с полем `kind`;
- собственные записи стажера: фокус-время, учеба, личное.

Сроки задач фронт берет сам из `GET /api/path` (`stages[].tasks[].dueDate`), отдельная ручка не нужна.

### Таблицы (миграция `015_calendar.sql`)

```sql
-- если meetings уже есть – добавить колонки
ALTER TABLE meetings ADD COLUMN kind text NOT NULL DEFAULT 'mentor'
  CHECK (kind IN ('mentor','team','lead','intro'));
ALTER TABLE meetings ADD COLUMN duration_min int NOT NULL DEFAULT 60;
ALTER TABLE meetings ADD COLUMN place text NOT NULL DEFAULT '';
ALTER TABLE meetings ADD COLUMN note text NOT NULL DEFAULT '';

-- ответ стажера на встречу
CREATE TABLE meeting_answers (
  meeting_id uuid REFERENCES meetings(id) ON DELETE CASCADE,
  intern_id  uuid REFERENCES users(id) ON DELETE CASCADE,
  answer     text NOT NULL CHECK (answer IN ('accept','move')),
  reason     text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (meeting_id, intern_id)
);

-- свои записи стажера
CREATE TABLE calendar_entries (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  intern_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title        text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 120),
  kind         text NOT NULL CHECK (kind IN ('focus','study','personal')),
  starts_at    timestamptz NOT NULL,
  duration_min int NOT NULL CHECK (duration_min BETWEEN 15 AND 480),
  shared       boolean NOT NULL DEFAULT true,
  created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ON calendar_entries (intern_id, starts_at);
```

### Объект `CalendarEvent`

```json
{
  "id": "uuid",
  "title": "1:1 с Игорем",
  "kind": "mentor | team | lead | intro | focus | study | personal",
  "startsAt": "2026-11-12T15:00:00+03:00",
  "durationMin": 60,
  "withName": "Игорь, наставник",
  "place": "видеозвонок",
  "note": "текст подсказки под встречей",
  "status": "pending | accepted | move_requested | own",
  "shared": false
}
```

- Встречи: `status` берется из `meeting_answers`. Нет ответа – `pending`, `accept` – `accepted`, `move` – `move_requested`.
- Свои записи: `status = "own"`, `withName = "только ты"`, `shared` – видна ли наставнику.
- Прошла встреча или нет, фронт считает сам по `startsAt + durationMin`.

### Ручки

| Метод | Путь | Тело | Ответ |
|---|---|---|---|
| GET | `/api/me/calendar?from=ISO&to=ISO` | – | `{ "events": CalendarEvent[] }`, по возрастанию `startsAt`, только `from <= startsAt < to` |
| POST | `/api/me/calendar/entries` | `{ title, kind, startsAt, durationMin, shareWithMentor }` | `CalendarEvent`, 201 |
| DELETE | `/api/me/calendar/entries/{id}` | – | 204; чужая или несуществующая запись – 404 |
| POST | `/api/me/calendar/events/{meetingId}/answer` | `{ "answer": "accept" \| "move", "reason": "до 200 символов" }` | `CalendarEvent` |

Проверки при `POST /entries`:
- `title` 1–120 символов;
- `kind` – одно из `focus`, `study`, `personal`;
- конец записи не позже 18:00 по Москве (иначе 400 `validation_error`);
- запись пересекается с другим событием стажера – **409** `{"error":{"code":"conflict","message":"Пересекается с «Знакомство с DevOps»"}}`. Фронт покажет `message` как есть.

Побочные эффекты:
- `answer = move` пишет в тред стажер–наставник сообщение от стажера «Можно перенести «…»? {reason}» и увеличивает `unread` у наставника. Фронт после ответа сам обновит чат с наставником.
- `answer = accept`: можно записать уведомление наставнику (таблица `notifications`).
- Запись с `shared = true` наставник видит в `GET /api/mentor/meetings` как занятое время стажера (например, `kind: "busy"`, без названия для `personal`).
- Новая встреча, перенос или отмена в кабинете наставника (`/api/mentor/meetings`) сразу видны стажеру в календаре: это те же строки `meetings`.

## 2. Шаги целей стажировки

Прогресс цели больше не кликается: стажер не может сам выставить себе 100%. Цель состоит из шагов, а процент – это доля выполненных шагов. Считает его backend.

### Таблица

```sql
CREATE TABLE goal_steps (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  goal_id    uuid NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  position   int  NOT NULL,
  title      text NOT NULL,
  source     text NOT NULL CHECK (source IN ('task','mentor','lead')),
  task_id    uuid REFERENCES task_templates(id) ON DELETE SET NULL, -- для source = task
  done_at    timestamptz,
  done_by    uuid REFERENCES users(id)
);
```

### Что добавить в ответ `GET /api/goals` и `GET /api/interns/{id}/goals`

У каждой цели – массив `steps`:

```json
"steps": [
  { "id": "uuid", "title": "Доступ к GitLab", "source": "task", "stageCode": "02", "taskId": "uuid", "status": "in_progress" },
  { "id": "uuid", "title": "Merge в main", "source": "mentor", "stageCode": null, "taskId": null, "status": "todo" }
]
```

- `status` для `source = task` берется из статуса задачи стажера (`todo`, `in_progress`, `waiting` → `in_progress`, `done`). Для `mentor` и `lead`: `done`, если `done_at` заполнено, иначе `todo`.
- `progressPct = round(done / всего шагов * 100)`. Пересчитывать при смене статуса задачи (`POST /api/path/tasks/{id}/complete`, `PUT /api/interns/{id}/tasks/{taskId}`) и при подтверждении шага.
- Подтверждение шага (новая ручка, только наставник стажера или руководитель): `POST /api/goals/{goalId}/steps/{stepId}` `{ "done": true }` → `Goal`.
- `POST /api/goals/{goalId}/progress` от стажера теперь отвечает 403. Ручку можно оставить наставнику для заметок.
- Поле `steps` необязательное: без него фронт покажет просто полоску процента. Поэтому выкатывать можно по шагам.

## 3. Что не требует изменений на backend

- Новый «Мой путь»: карточки этапов, кольцо из штрихов, шкала недель. Используется тот же `GET /api/path`. Если наставник добавит пятый этап через `/api/console/stages`, он появится автоматически: стрелка у карточек листает этапы.
- Справочник: те же `GET /api/kb`, `/api/kb/search`, `/api/kb/articles/{id}`. Для подписей разделов нужны `sections[].hand` и `sections[].color` – они уже есть.
- Чаты (AI-buddy и наставник) теперь закреплены, прокручивается только лента. Это чисто фронт.

## 4. Проверка после подключения

1. `docker compose up --build -d`. В `.env` фронта: `VITE_PENDING_MOCKS=false`.
2. Войти `intern_anna` / `demo123` → «Календарь»:
   - видна встреча 1:1 из кабинета наставника;
   - «Приду» меняет статус;
   - «Попросить перенести» добавляет сообщение в чат наставника;
   - своя запись сохраняется и удаляется;
   - пересечение со встречей возвращает 409 с понятным текстом.
3. Войти `mentor_igor` → «Расписание»: назначить встречу Анне и проверить, что она появилась в ее календаре со статусом «ждет ответа».
4. «Мой путь» у Анны: у целей видны шаги. Отметить задачу «Доступ к GitLab» выполненной – процент первой цели должен вырасти.
