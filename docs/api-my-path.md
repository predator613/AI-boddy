# API страницы «Мой путь» и консоли наставника

Контракт для фронта. Все запросы с заголовком `Authorization: Bearer <token>`. Ошибки в общем формате `{ "error": { "code", "message" } }`: 400 `validation_error`, 401 `unauthorized`, 403 `forbidden` (действие запрещено роли), 404 `not_found` (нет объекта или он чужой).

Готовые файлы для фронта:
- `frontend/src/schemas/path.ts` – Zod-схемы ответов и форм.
- `frontend/src/api/path.ts` – функции запросов (через `apiFetch` из `client.ts`).
- `frontend/src/hooks/usePath.ts` – хуки TanStack Query.
- `frontend/src/lib/pathUi.ts` – подписи и цвета статусов из макета, кольцо прогресса, тексты сроков и встречи.

## 1. GET /api/path – экран стажера

Один запрос отдает все данные экрана. Ответ, сокращенный (стажер intern_anna):

```json
{
  "intern": { "id": "aaaa…0001", "name": "Анна", "teamName": "Backend", "mentor": { "id": "aaaa…0002", "name": "Игорь" } },
  "week": { "current": 2, "total": 24, "day": 10, "startDate": "2026-11-01",
            "milestones": [{ "week": 1, "title": "Старт" }, { "week": 4, "title": "Первые задачи" }, { "week": 12, "title": "Отзыв" }, { "week": 24, "title": "Итоги" }] },
  "progress": { "done": 5, "inProgress": 2, "todo": 8, "total": 15, "percent": 33 },
  "stages": [
    { "id": "b500…0002", "code": "02", "title": "Доступы", "color": "#BADAFB", "status": "current",
      "doneCount": 2, "totalCount": 4, "tip": "Доступы выдает руководитель команды…",
      "suggestions": ["Запросить GitLab", "Сколько ждать согласования?", "Кто выдает доступы?"],
      "tasks": [
        { "id": "b700…0203", "title": "Доступ к тестовому стенду", "note": "Ждет подтверждения Олега", "status": "waiting", "isNext": false, "dueDate": "2026-11-05", "action": { "kind": "access_request", "payload": { "service": "test_stand", "role": "read" } } },
        { "id": "b700…0204", "title": "Получить доступ к GitLab", "note": "Нужен для первой задачи…", "status": "in_progress", "isNext": true, "dueDate": "2026-11-10", "action": { "kind": "access_request", "payload": { "service": "gitlab", "group": "backend", "role": "developer" } } }
      ],
      "resources": [
        { "id": "…", "kind": "access", "name": "Тестовый стенд", "note": "роль: чтение", "tag": "на согласовании", "tagKind": "wait" },
        { "id": "…", "kind": "access", "name": "GitLab", "note": "группа backend", "tag": "не запрошен", "tagKind": "none" }
      ] }
  ],
  "nextStep": { "taskId": "b700…0204", "stageId": "b500…0002", "stageCode": "02", "title": "Получить доступ к GitLab",
                "status": "in_progress", "dueDate": "2026-11-10", "dueInDays": 0, "action": { "kind": "access_request", "payload": { … } } },
  "nextMeeting": { "id": "…", "title": "Встреча 1:1", "startsAt": "2026-11-12T15:00:00+03:00", "hostId": "aaaa…0002", "hostName": "Игорь" },
  "buddy": { "text": "Анна, ты на этапе «Доступы». Осталось 2 из 4. Следующий шаг – «Получить доступ к GitLab». Могу помочь сделать его прямо сейчас.",
             "suggestions": ["Запросить GitLab", "Сколько ждать согласования?", "Кто выдает доступы?"] }
}
```

Где что на макете:

| Элемент макета | Поле |
|---|---|
| «Привет, Анна!», «наставник Игорь», «Стажер · Backend» | `intern` |
| «Неделя 2 из 24», шкала из 24 недель, подписи вех | `week` |
| Кольцо 33 %, плашки 5 / 2 / 8 | `progress`; деления кольца – `ringSegments(stages)` |
| Карточки этапов 01–04 | `stages[]`: `code`, `title`, `status`, `doneCount/totalCount`, `color`; оформление – `stageCard()` |
| Раскрытый этап: задачи | `stages[i].tasks[]`; бейдж – `taskBadge()`, кружок – `taskDot()` |
| Раскрытый этап: боковая панель | `stages[i].resources[]`; тег – `resourceTag(tagKind)` |
| «Совет» | `stages[i].tip` |
| Карточка «Следующий шаг», «срок сегодня» | `nextStep`; текст срока – `dueText(dueInDays)` |
| Карточка наставника, «чт 12, Встреча 1:1, 15:00, через 2 дня» | `intern.mentor`, `nextMeeting`; части даты – `meetingParts()` |
| Реплика AI-buddy и кнопки-подсказки | `buddy.text`, `buddy.suggestions` (в раскрытом этапе – `stages[i].suggestions`) |

Статусы задач: `todo` (впереди), `in_progress` (в работе), `waiting` (на согласовании), `done` (готово). Плашка «в работе» на кольце = `in_progress + waiting`. Статус этапа: `done` – все задачи готовы, `current` – первый незавершенный этап, `todo` – следующие.

Кнопки:
- «Уже сделано» → `POST /api/path/tasks/{taskId}/complete`, ответ – обновленный путь целиком (хук `useCompleteTask` сам кладет его в кеш).
- «Сделать с AI-buddy» / «Оформить с AI-buddy» → берите `task.action` и открывайте чат с этим действием. Действия выполнит чат на следующем этапе (страница «Заявки»).

## 2. Наставник и руководитель

- `GET /api/interns` → `[{ id, login, name, teamName, mentor, week, progress, currentStageCode, currentStage }]`.
- `GET /api/interns/{internId}/path` → тот же формат, что `/api/path`.
- `PUT /api/interns/{internId}/tasks/{taskId}` с `{ "status": "waiting", "note": "Заявка у Олега" }` → обновленный путь стажера. Пустая `note` возвращает текст из шаблона.
- `POST /api/interns/{internId}/meetings` с `{ "title": "Разбор недели", "startsAt": "2026-11-13T11:00:00+03:00" }` → встреча (201). Без `title` будет «Встреча 1:1».

## 3. Консоль наставника: универсальные шаблоны

Наставник и руководитель редактируют шаблон своей команды. Изменения сразу видны всем стажерам команды. Номер этапа (01, 02…) считается из порядка, хранить его не нужно.

| Запрос | Тело | Ответ |
|---|---|---|
| GET /api/console/templates | – | этапы с `tasks[]` и `resources[]` |
| GET /api/console/action-kinds | – | `[{ kind, description }]` для выпадающего списка |
| POST /api/console/stages | `{ title, color?, tip?, suggestions? }` | этап (201), добавляется в конец |
| PUT /api/console/stages/{id} | `{ title, color, tip, suggestions, position }` | этап; `position` меняет порядок |
| DELETE /api/console/stages/{id} | – | 204, удаляет с задачами |
| POST /api/console/stages/{id}/tasks | `{ title, note?, dueDay?, action? }` | задача (201) |
| PUT /api/console/tasks/{id} | `{ title, note, dueDay, action, position }` | задача; `action: null` убирает действие |
| DELETE /api/console/tasks/{id} | – | 204 |
| POST /api/console/stages/{id}/resources | `{ kind, name, note?, tag?, tagKind?, taskId? }` | карточка (201) |
| DELETE /api/console/resources/{id} | – | 204 |

Ограничения (те же проверки есть в Zod-схемах форм): `title` этапа до 80 символов, задачи до 120; `color` вида `#RRGGBB`; до 6 подсказок по 80 символов; `dueDay` – день стажировки 1–730; `action.kind` – одно из `access_request`, `leave_request`, `ask_buddy`, `open_article`, `payload` – JSON-объект; `kind` карточки – `person`, `access`, `tool`, `article`, `link`; `tagKind` – `ok`, `wait`, `none`, `mentor`. Если у карточки указан `taskId`, ее тег у стажера считается сам: «активен», «на согласовании» или «не запрошен».

## 4. Права

| | стажер | наставник | руководитель |
|---|---|---|---|
| свой путь | да | – | – |
| путь стажера | только свой | своих подопечных | всех стажеров команды |
| статусы задач стажера | только «Уже сделано» у себя | своих подопечных | команды |
| шаблоны | нет (403) | своей команды | своей команды |

Чужие стажеры и шаблоны другой команды отвечают 404, чтобы не раскрывать, что они существуют.
