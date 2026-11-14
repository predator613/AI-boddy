# API: кабинет наставника и чат стажера с наставником

Фронт уже вызывает эти адреса. Пока backend их не реализовал, отвечают моки `frontend/src/mocks/mentorData.ts` (режим `VITE_PENDING_MOCKS`, включен по умолчанию). Когда ручка готова – удалите ее обработчик из `mentorHandlers`, код страниц менять не нужно. Схемы ответов – `frontend/src/schemas/mentor.ts`.

Все запросы с `Authorization: Bearer <token>`. Ошибки: `{ "error": { "code", "message" } }`. Ручки `/api/mentor/*` и `/api/materials` (запись) – только `mentor` и `lead` (стажеру 403), наставник видит только своих стажеров (чужие – 404). Ручки `/api/me/mentor-thread*` – только стажер.

План адаптации уже работает через существующие `/api/console/*` (docs/api-my-path.md), отдельного API не нужно.

## Статистика (плашки на «Моих стажерах»)

`GET /api/mentor/stats`

```json
{ "periodDays": 7, "answeredByBuddy": 23, "handedOff": 3, "savedMinutes": 240,
  "mentorHoursSpent": 9, "mentorHoursPlan": 44 }
```

`answeredByBuddy` – вопросы стажеров, закрытые AI-buddy без наставника; `handedOff` – переданные наставнику; `mentorHoursPlan` – ориентир Авито (40–48 ч на 6-месячную программу).

## Переписка

Сообщение треда (`ThreadMessage`):

```json
{ "id": "…", "kind": "text|handoff|update", "author": "intern|mentor|buddy", "authorName": "Мария",
  "text": "…", "at": "2026-11-10T14:02:00+03:00",
  "update": null }
```

- `handoff` – AI-buddy не нашел ответ и передал вопрос наставнику (пишет agent).
- `update` – карточка изменения от наставника, `update = { "type": "plan|material|meeting", "title", "detail", "link": "/kb" | null, "acked": false }`. Создается автоматически при `notify=true` (материалы, встречи; позже – изменения плана).

Тред (`Thread`): `{ internId, internName, mentor: {id, name}, stageCode, stageTitle, messages: ThreadMessage[] }`.

| Метод | Путь | Кто | Тело | Ответ |
|---|---|---|---|---|
| GET | /api/mentor/threads | наставник | – | `[{ internId, internName, stageCode, stageTitle, lastText, lastAt, unread }]`, новые сверху |
| GET | /api/mentor/threads/{internId} | наставник | – | Thread |
| POST | /api/mentor/threads/{internId}/messages | наставник | `{ text, addToKb }` | Thread |
| POST | /api/mentor/threads/{internId}/read | наставник | – | 204 |
| GET | /api/me/mentor-thread | стажер | – | Thread (404 `no_mentor`, если наставника нет) |
| POST | /api/me/mentor-thread/messages | стажер | `{ text }` | Thread, наставнику +1 unread |
| POST | /api/me/mentor-thread/updates/{messageId}/ack | стажер | – | Thread (у карточки `acked: true`) |

`addToKb: true` – ответ сохраняется статьей справочника (`source = mentor_answer`, этап стажера) и попадает в материалы. Фронт опрашивает треды раз в 15 секунд.

## Материалы

`Material`: `{ id, title, kind: "pdf|docx|md|link|answer", stageCode: "02" | null, useInAi, authorName, createdAt, sizeLabel }`

| Метод | Путь | Тело | Ответ |
|---|---|---|---|
| GET | /api/materials | – | Material[] |
| POST | /api/materials | multipart: `title`, `stageCode` (пусто = все этапы), `useInAi` (`true/false`), `notify`, и `file` или `url` | Material, 201 |
| DELETE | /api/materials/{id} | – | 204 |

После загрузки с `useInAi=true` материал индексируется для поиска AI-buddy (pgvector или full-text). При `notify=true` всем стажерам наставника уходит карточка `update` с `type: material`.

## Расписание

`MentorMeeting`: `{ id, title, internId | null, withName, kind: "one_on_one|review|team", startsAt, durationMin }` (`internId: null` – встреча со всеми стажерами).

| Метод | Путь | Тело | Ответ |
|---|---|---|---|
| GET | /api/mentor/meetings?from=ISO&to=ISO | – | MentorMeeting[] по возрастанию |
| POST | /api/mentor/meetings | `{ internId \| "all", title, startsAt, durationMin, notify }` | MentorMeeting, 201 |
| PATCH | /api/mentor/meetings/{id} | `{ startsAt }` | MentorMeeting |
| DELETE | /api/mentor/meetings/{id} | – | 204 |

Создание, перенос и отмена добавляют стажеру карточку `update` с `type: meeting`. Ближайшая встреча должна попадать в `nextMeeting` у `GET /api/path`.

## Автоматизации (имитация ИИ-крона и feature flags)

`Automation`: `{ id, kind: "deadline_reminder|weekly_goal_checkin|overdue_alert|approver_reminder", title, note, enabled }`

- `GET /api/mentor/automations` → Automation[]
- `PATCH /api/mentor/automations/{id}` `{ enabled }` → Automation

Воркер по расписанию проверяет включенные правила: напоминание о сроке задачи за день, еженедельный чек-ин по целям (пятница 17:00), сигнал наставнику о просрочке больше 2 дней, напоминание согласующему о заявке старше 3 дней.

## Заявки на доступ (по ответам организаторов)

Поля заявки: `resource`, `project`, `accessLevel: read|edit|admin`, `justification`, согласующий зависит от типа ресурса (`owner`, `lead`, `devops`). Статусы моделируем сами: `created → pending_approval → approved|rejected → granted`.
