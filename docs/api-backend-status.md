# Backend: что готово по запросам фронта (13.11.2026, вечер)

Ответ на `docs/api-jira.md`, `docs/api-support.md`, `docs/api-lead-feed.md` и `docs/api-mentor-console.md`. Миграции `018_jira_support_lead.sql` и `019_mentor_console.sql`, тесты `backend/internal/app/staff_test.go` и `mentor_test.go`.

Теперь backend реализует все ручки, которые вызывает фронт, поэтому `pendingHandlers` пуст. Моки кабинета наставника, руководителя и календаря работают только в полном режиме без backend (`npm run dev:mock`) и в тестах.

## Jira Service Desk (api-jira.md)

- Ключи: доступы `ACCESS-…`, заявки в отделы `SD-…`, отгулы `HR-…`, обращения `SUP-…`.
- Поле `external: { system, key, url } | null` в `GET /api/requests` и в `result` сообщения чата. В схеме фронта у `ChatMessage.result` поля `external` пока нет – добавьте, чтобы показывать ключ в чате.
- Страница заявки в имитации: `http://localhost:8090/browse/{ref}`.
- Обычные доступы (стенд, логи) решает команда доступов в сервисе, GitLab – владелец проекта (попадает ему в «Согласования»), отгулы – руководитель. В истории заявки шаг «Передана в Jira Service Desk».

## Поддержка (api-support.md)

- `POST /api/support/tickets` → 201 `{ id, number, type, details, status, createdAt, answer }`; `GET /api/support/tickets` – мои обращения.
- Статусы `new` → `in_progress` → `done` с ответом поддержки, автору приходит уведомление.

## Руководитель и наставник (api-lead-feed.md)

- 2: шаги и компетенция в POST/PATCH цели; `canAgree = false` после согласования.
- 3: `GET /api/approvals`, `POST /api/approvals/{id}`.
- 4: `GET /api/mentor/feed`, `POST /api/mentor/feed/read`.
- 5: `details` у задачи в `GET /api/path`.
- 7: `period` у этапа (редактируется и в консоли плана: поле `period` в `POST/PUT /api/console/stages`); этапы 05 «Безопасность и учеба», 06 «Экватор», 07 «Итоги» у команды Backend; задачи этапа 01 «Добавиться в чаты и каналы команды» (`join_chats`) и «Завести конспект встреч». `join_chats`: карточка «Добавить в чаты и каналы?» с полями «Чаты», «Каналы», «Закрытые», после подтверждения заявка сразу `done`, задача закрывается. У `ask_buddy` в payload есть `prompt`.
- 5, редактор плана: у задачи в `GET /api/console/templates` поля `about`, `helper`, `doneWhen`; их же принимают `POST /api/console/stages/{stageId}/tasks` и `PUT /api/console/tasks/{taskId}` (до 600, 300 и 300 символов, можно `{{mentor}}` и `{{lead}}`). Если в PUT нет ни одного из трех полей, подробности не меняются.
- 8: чек-лист наставника, id пунктов – UUID.
- 9: `GET /api/me/notes`, `PUT /api/me/notes/{eventId}`.
- 10: `competency` у цели; `section` у материала.
- 11: `GET /api/mentor/metrics`. `accessHours` – дробное число часов.

Демо: у Анны и Дмитрия новые задачи этапа 01 уже выполнены, у Лизы – нет, на ней видно действие `join_chats`.

## Кабинет наставника (api-mentor-console.md)

- `GET /api/mentor/stats` (наставник и руководитель).
- Переписка: `GET /api/mentor/threads`, `GET /api/mentor/threads/{internId}`, `POST …/messages` `{ text, addToKb }`, `POST …/read` – только наставник своих стажеров (руководителю 403, чужой стажер 404). `GET /api/me/mentor-thread`, `POST /api/me/mentor-thread/messages`, `POST /api/me/mentor-thread/updates/{messageId}/ack` – стажер; без наставника 404.
- Переданный AI-buddy вопрос в переписке – `kind: handoff`, `author: buddy`. Ответ наставника на него приходит стажеру и в чат AI-buddy («Игорь ответил на твой вопрос: …»). С `addToKb: true` ответ сохраняется в справочник (раздел «Ответы наставников») и AI-buddy дальше находит его сам.
- Материалы: `GET /api/materials`, `POST /api/materials` (multipart: `title`, `stageCode`, `section`, `useInAi`, `notify`, `file` или `url`), `DELETE /api/materials/{id}`. Файлы PDF, DOCX, MD, TXT до 10 МБ; текст MD и TXT попадает в поиск AI-buddy, у PDF и DOCX ищется только название. Новый раздел создается автоматически. С `notify` стажерам приходит карточка `update` с `type: material`.
- Расписание: `GET /api/mentor/meetings?from&to`, `POST /api/mentor/meetings` (`internId` или `"all"`), `PATCH …/{id}` `{ startsAt }`, `DELETE …/{id}`. Создание (при `notify`), перенос и отмена – карточки `update` с `type: meeting`. Встреча попадает в календарь стажера и в `nextMeeting`.
- Автоматизации: `GET /api/mentor/automations`, `PATCH /api/mentor/automations/{id}` `{ enabled }`; `id` равен `kind`, по умолчанию все включены. Воркер раз в минуту: напоминание стажеру за день до срока, пятничный чек-ин по целям после 17:00, сигнал наставнику о просрочке больше 2 дней, напоминание согласующему о заявке старше 3 дней. Каждое уведомление – один раз на объект (и на день для напоминаний).
