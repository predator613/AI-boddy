# Backend: что готово по запросам фронта (13.11.2026)

Ответ на `docs/api-jira.md`, `docs/api-support.md` и `docs/api-lead-feed.md`. Миграция `018_jira_support_lead.sql`, тесты `backend/internal/app/staff_test.go`.

## Готово

### Заявки через Jira Service Desk (api-jira.md)

- Ключи имитации: доступы `ACCESS-…`, заявки в отделы `SD-…`, отгулы `HR-…`, обращения `SUP-…`. Старые `ACC-…` в базе переименованы.
- `GET /api/requests`: у каждой заявки поле `external: { system, key, url } | null`. `url` приходит, когда задан `MOCK_SERVICES_PUBLIC_URL` (в docker-compose уже задан: `http://localhost:8090`). У отгулов `external = null`.
- То же поле `external` в `result` сообщения чата («Заявка создана»). Во фронте в схеме `ChatMessage.result` его пока нет, поэтому Zod его отбрасывает – добавьте `external` в схему, если хотите показывать ключ в чате.
- `GET http://localhost:8090/browse/{ref}` – страница заявки в имитации: ключ, что запрошено, автор, статус.
- Кто решает: `auto` – выдается сразу; `devops` (тестовый стенд, логи) – сервис заявок, в полях заявки «Куда ушла: Jira Service Desk, команда доступов»; `project_owner` (GitLab) – владелец проекта, заявка попадает ему в «Согласования»; отгулы – руководителю.
- В истории заявки после «Создана AI-buddy» идет шаг «Передана в Jira Service Desk».
- Заявку, которую решает человек, имитация сама «согласует» через 6 × `MOCK_DECISION_DELAY` (по умолчанию 4,5 минуты), если руководитель не успел. Обычные заявки – через 1 × задержку, как раньше.

### Обращения в поддержку (api-support.md)

- `POST /api/support/tickets` `{ type, details, internId, attachDialog }` → 201 `{ id, number: "SUP-8003", type, details, status: "new", createdAt, answer: null }`. В ответе полей больше, чем в документе (добавлены `type`, `details`, `answer`), лишнее можно не читать.
- `GET /api/support/tickets` – мои обращения, новые сверху.
- Статусы: `new` → `in_progress` (через `MOCK_DECISION_DELAY`) → `done` с ответом «Исправили, проверь, пожалуйста» (еще через одну задержку). Автору приходит уведомление.
- Ошибки: 400 `validation_error` (пустой тип или описание, описание длиннее 2000, `internId` от стажера или чужой стажер), 503, если сервис заявок недоступен.

### Руководитель и наставник (api-lead-feed.md)

- Раздел 2: `POST /api/interns/{id}/goals` и `PATCH /api/goals/{id}` принимают `steps` и `competency`. Шаг `task` привязывается к задаче плана по названию; отметки наставника у шагов с прежним названием при замене списка сохраняются. `canAgree = false` после согласования.
- Раздел 3: `GET /api/approvals`, `POST /api/approvals/{id}`. Коды как в документе: 400, 403, 404, 409. После решения меняется статус заявки у стажера, закрывается задача плана, стажер получает уведомление и сообщение в чате, наставник – запись в ленте.
- Раздел 4: `GET /api/mentor/feed`, `POST /api/mentor/feed/read`. В ленту пишется: согласование целей, подтверждение шага руководителем, решение по заявке (руководителя и сервиса), ответ стажера на приглашение, закрытие задачи стажером, отметка прогресса цели.
- Раздел 5: `details` у задачи в `GET /api/path` и `GET /api/interns/{id}/path` (`about`, `helper`, `doneWhen`, `goal`). Имена наставника и руководителя подставляются из связей стажера.
- Раздел 8: `GET /api/mentor/interns/{internId}/checklist`, `POST …/checklist/{itemId}`. id пунктов – UUID, не `c-chats`. Руководитель видит и меняет отметки наставника стажера.
- Раздел 9: `GET /api/me/notes`, `PUT /api/me/notes/{eventId}`. `eventId` – id события из `GET /api/me/calendar` (встреча или своя запись), чужое событие – 404.
- Раздел 10, цели: поле `competency` у цели.

Моки этих ручек убраны из `pendingHandlers` (остались только в полном режиме без backend и в тестах): `frontend/src/mocks/leadData.ts`, `pending.ts`, `handlers.ts`.

## Еще не сделано

- Раздел 7: `period` у этапа, этапы 05–07, действие `join_chats`, `payload.prompt` у `ask_buddy`.
- Раздел 10, материалы: поле `section` и сами ручки `/api/materials`.
- Раздел 11: `GET /api/mentor/metrics` (пока отвечает мок `leadPendingHandlers`).
- `docs/api-mentor-console.md`: `/api/mentor/stats`, переписка (`/api/mentor/threads*`, `/api/me/mentor-thread*`), материалы, расписание наставника (`/api/mentor/meetings*`), автоматизации.
- В редакторе плана поля `about`, `helper`, `doneWhen` пока не редактируются, только seed.
