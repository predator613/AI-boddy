# API, которого фронт ждет от backend (чат, заявки)

Эти эндпоинты фронт уже вызывает. Пока их нет в backend, на них отвечает MSW прямо в браузере (`frontend/src/mocks/pending.ts`), остальные запросы идут в настоящий backend. Когда эндпоинт появится в backend – удалите его обработчик из `pendingHandlers`, фронт менять не нужно. Схемы ответов: `frontend/src/schemas/assistant.ts`.

Все запросы с `Authorization: Bearer <token>`, ошибки в общем формате `{ "error": { "code", "message" } }`.

## Чат с помощником

### POST /api/chat/messages
Запрос: `{ "text": "Как получить доступ к GitLab?", "action": null }`. `action` – действие задачи из «Моего пути» (`task.action`), например `{ "kind": "access_request", "payload": { "service": "gitlab", "role": "developer", "project": "backend/auth-service" } }`.

Ответ 200: `{ "messages": ChatMessage[] }` – новые сообщения помощника.

```json
{
  "id": "m-1",
  "role": "assistant",
  "kind": "text | confirm | result | error",
  "text": "Доступ выдает руководитель команды…",
  "source": { "title": "Памятка стажера → Доступы", "articleId": "a-access" },
  "confirm": { "actionId": "act-1", "title": "Создать заявку на доступ?", "fields": [{ "label": "Ресурс", "value": "GitLab" }, { "label": "Проект", "value": "backend / auth-service" }, { "label": "Уровень прав", "value": "редактирование (Developer)" }, { "label": "Обоснование", "value": "Нужен для первой задачи" }, { "label": "Согласует", "value": "Олег, владелец проекта" }], "state": "pending" },
  "result": { "requestId": "req-1042", "number": "1042", "title": "Доступ к GitLab", "status": "waiting" },
  "mood": "calm | happy | think | mentor | sad"
}
```

- `kind: "confirm"` – помощник предлагает действие и ждет подтверждения (поле `confirm`).
- `kind: "result"` – действие выполнено, заявка создана (поле `result`).
- `kind: "error"` – отказ, например действие недоступно роли.
- `source` – откуда ответ (статья справочника), может быть `null`.
- `mood` – состояние маскота на фронте.

### POST /api/chat/actions/{actionId}/confirm
Выполняет действие (создает заявку через mock-сервис) и возвращает `{ "messages": [ … kind: "result" … ] }`. Если действие связано с задачей пути, задача переходит в `waiting`. 404 `not_found` – действие уже выполнено или чужое.

### POST /api/chat/actions/{actionId}/cancel
Отменяет действие, возвращает `{ "messages": [ … ] }`.

## Заявки

### GET /api/requests
Заявки текущего стажера, новые сверху:

```json
[{
  "id": "req-1042", "number": "1042", "kind": "access | leave | other",
  "title": "Доступ к GitLab", "status": "waiting | approved | done | rejected",
  "createdAt": "2026-11-10T13:20:00+03:00", "via": "buddy | auto | manual",
  "fields": [{ "label": "Сервис", "value": "GitLab" }],
  "history": [{ "title": "Создана AI-buddy", "at": "10 нояб, 13:20", "state": "done | now | rejected" }],
  "rejectReason": null
}]
```

## Справочник

Реализован в backend: `GET /api/kb`, `GET /api/kb/articles/{id}`, `GET /api/kb/search?q=`. Контракт – `docs/api-goals-kb.md`, MSW-обработчик убран из `pendingHandlers`.

## Права
- Заявки и чат-действия – только свои. Наставник и руководитель раздела «Заявки» не видят (фронт его не показывает).
- Действие, недоступное роли (например, доступ к продакшену для стажера), помощник не выполняет: ответ `kind: "error"`.
