# AI-buddy – ИИ-наставник для стажеров (прототип, Hack&Change 2026, кейс Авито)

Все данные в проекте синтетические. Тестовые инструкции не являются регламентами Авито.

## Быстрый запуск

Нужен только Docker (Docker Desktop 4.x на Windows/macOS или Docker Engine 24+ с плагином compose v2 на Linux).

```bash
docker compose up --build -d
docker compose ps          # backend, mock-services, postgres, redis должны быть healthy
```

Проверка: http://localhost:8080/api/health → `{"status":"ok"}`

Интерфейс: **http://localhost:3000** – войти под любым тестовым пользователем (карточки на странице входа). Стажер видит «Мой путь», наставник и руководитель – своих стажеров. Подробнее про фронтенд: `frontend/README.md`.

Все экраны работают с backend: «Мой путь», цели, справочник, чат с AI-buddy и заявки. Корпоративную систему заявок имитирует контейнер `mock-services` (http://localhost:8090/openapi.yaml).

Помощник работает и без ключа модели (офлайн-режим), все сценарии проходят. Как подключить языковую модель – раздел «Языковая модель» ниже.

Остановить: `docker compose down`. Удалить и базу тоже (начать с чистого листа): `docker compose down -v`.

Файл `.env` не обязателен. Чтобы подключить модель, поменять порты или секреты: `cp .env.example .env` и отредактировать (пример без секретов лежит в корне).

## Тестовые учетные данные

Пароль у всех: `demo123`

| Логин | Имя | Роль | Команда |
|---|---|---|---|
| intern_maria | Мария | стажер, 2-я неделя, этап «Доступы», цели согласованы | Backend |
| intern_dima | Дмитрий | стажер, 6-я неделя, этап «Первые задачи», есть просрочка | Backend |
| intern_liza | Лиза | стажер, 1-я неделя, цели еще не согласованы | Backend |
| mentor_igor | Игорь | наставник Марии, Дмитрия и Лизы | Backend |
| lead_artemiy | Артемий | руководитель, согласует цели и GitLab backend/auth-service | Backend |

Кирилл (DevOps, роль staff) – согласующий тестового стенда и логов, в приложение не входит.

## Заявленные сценарии и порядок проверки в интерфейсе

Откройте http://localhost:3000 и входите карточками на странице входа. Ниже пять сценариев, которые заявлены для проверки; остальное (заявки в отделы, поддержка, кабинет наставника) – дополнительная помощь наставнику и руководителю.

1. **Доступ к сервису (главный сценарий).** Войти как Мария (`intern_maria`) → «Мой путь» → у задачи «Получить доступ к GitLab» нажать «Подать заявку» → в чате проверить карточку и нажать «Подтвердить» → заявка получает ключ `ACCESS-…` в Jira Service Desk (имитация, ссылка открывает страницу заявки). Выйти, войти как Артемий (`lead_artemiy`) → «Согласования» → «Согласовать». Снова Мария: в чате «доступ к GitLab выдан», задача на «Моем пути» закрыта, кольцо прогресса выросло. Без руководителя имитация одобрит заявку сама через 6 × `MOCK_DECISION_DELAY`.
2. **Вопрос о процессах.** Мария → «Чат с помощником» → «как устроено код-ревью?» – ответ со ссылкой на статью справочника. Вопрос, которого нет в справочнике (например, «можно ли работать месяц из другого города?»), → помощник предлагает передать его наставнику, показывает формулировку и ждет «да». Войти как Игорь (`mentor_igor`) → «Сообщения» → ответить с галочкой «Добавить ответ в справочник». Мария видит ответ в чате.
3. **Отгул.** Мария → чат → «хочу взять отгул в пятницу» → карточка с датой → «Подтвердить» → Артемий согласует в «Согласованиях» → статус на странице «Мои заявки».
4. **Путь и цели адаптации.** Мария → «Мой путь»: 7 этапов по чек-листу Авито (от первой недели до выпускного), статусы задач, следующий шаг, цели. Лиза (`intern_liza`): цели еще не согласованы → Артемий → «Цели стажеров» → добавить шаги и «Согласовать цели».
5. **Первая неделя.** Лиза → «Мой путь» → «Добавиться в чаты и каналы команды» → помощник оформляет заявку, задача закрывается. «Календарь» → встреча 1:1 → конспект встречи.

Ошибочную заявку можно отозвать, пока она на согласовании: «Мои заявки» → «Отозвать заявку» или в чате «отзови заявку №…» и «да».

Права: стажер не видит экраны наставника и руководителя, наставник не пишет в чат стажера, руководитель не читает переписку наставника со стажером, запрещенный роли доступ («дай доступ к логам продакшена») помощник не оформляет.

Без ключа модели (`LLM_PROVIDER=mock`) все пять сценариев проходят, ответы помощника проще. Метрики пользы и способ проверки эффекта – `docs/metrics.md`, спецификации AI-функций – `specs/`.

## Порядок проверки через API (backend)

```bash
# 1. список тестовых пользователей (работает при DEMO_MODE=true)
curl http://localhost:8080/api/auth/test-users

# 2. вход
curl -X POST http://localhost:8080/api/auth/login -H "Content-Type: application/json" -d "{\"login\":\"intern_maria\",\"password\":\"demo123\"}"

# 3. текущий пользователь (подставить token из ответа)
curl http://localhost:8080/api/auth/me -H "Authorization: Bearer <token>"

# 4. цели Марии и отметка прогресса
curl http://localhost:8080/api/goals -H "Authorization: Bearer <token>"
curl -X POST http://localhost:8080/api/goals/f3000000-0000-4000-8000-000000000002/progress -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d "{\"progressPct\":35}"

# 5. справочник и поиск
curl http://localhost:8080/api/kb -H "Authorization: Bearer <token>"
curl -G http://localhost:8080/api/kb/search --data-urlencode "q=как взять отгул" -H "Authorization: Bearer <token>"

# 6. главный сценарий: доступ к GitLab через AI-buddy (actionId – из карточки confirm)
curl -X POST http://localhost:8080/api/chat/messages -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d "{\"text\":\"Запроси доступ к GitLab\"}"
curl -X POST http://localhost:8080/api/chat/actions/<actionId>/confirm -H "Authorization: Bearer <token>"
curl http://localhost:8080/api/requests -H "Authorization: Bearer <token>"   # done после согласования Артемием (или сам через 6 × MOCK_DECISION_DELAY)

# 7. заявка в отдел: ноутбук уходит в отдел оборудования, наставник получает уведомление
curl -X POST http://localhost:8080/api/chat/messages -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d "{\"text\":\"мне не выдали ноутбук, нужен\"}"

# 8. свои люди и отделы (имена – из связей пользователя)
curl http://localhost:8080/api/me/contacts -H "Authorization: Bearer <token>"
```

В интерфейсе – раздел «Заявленные сценарии и порядок проверки в интерфейсе» выше. Отказ согласующего можно показать вручную: `POST http://localhost:8090/v1/tickets/<ACCESS-номер>/decision` с `{"approve":false,"reason":"…"}`.

Или в VS Code: файл `backend/api.http` (расширение REST Client).

## Языковая модель

По умолчанию `LLM_PROVIDER=mock` – офлайн-режим без ключа и без сети: помощник определяет намерение по ключевым словам и отвечает фрагментом справочника. Весь сценарий (права, подтверждение, заявка, статус) работает так же, как с моделью.

Подключить модель – любой OpenAI-совместимый API (Yandex AI Studio, Cloud.ru, Avito A-Vibe через Ollama; подробно – `docs/llm-setup.md`). В `.env`:

```
LLM_PROVIDER=openai_compat
LLM_BASE_URL=<адрес API, оканчивается на /v1>
LLM_API_KEY=<ключ>
LLM_MODEL=<имя модели>
LLM_PROJECT=<ID каталога, только для Yandex AI Studio>
```

и `docker compose up -d backend` (пересобирать не нужно). Проверка: в `docker compose logs backend` строка `assistant ready` с `"llm":"openai_compat"`. Примеры адресов и имен моделей для Yandex AI Studio, Cloud.ru, OpenAI, vLLM и Ollama – в начале `docs/llm-setup.md`. Модель должна поддерживать function calling (tools). Сравнение провайдеров и пошаговое получение ключа: `docs/llm-setup.md` (рекомендуется Yandex AI Studio). GPU не нужен: модель вызывается по API. Если модель не ответила за `LLM_TIMEOUT` (20 с), ответ собирается офлайн.

## Тесты и покрытие

```bash
docker compose --profile test run --rm backend-tests
```

Команда поднимает PostgreSQL, прогоняет unit- и интеграционные тесты и пишет отчет:
- `backend/coverage/coverage.out` – профиль покрытия,
- `backend/coverage/coverage.html` – отчет для браузера.

Покрытие считается по всему прикладному коду backend, кроме имитации корпоративных систем (`internal/mockservices`, `cmd/`). Текущее значение – около 86%.

Общий отчет по фронту и бэку одним числом (покрытые строки вместе, а не среднее двух процентов):

```bash
docker compose --profile test run --rm backend-tests
docker compose --profile test run --rm frontend-tests
python scripts/coverage_summary.py      # итог около 89%, отчет в coverage/summary.md
```

Сквозная проверка сценариев и прав через API на запущенном проекте (нужен Node 18+, лучше на чистой базе):

```bash
node scripts/qa/scenarios.mjs   # пять сценариев из README и спецификации
node scripts/qa/security.mjs    # вход, чужие данные, инъекции, CORS
```

Результаты ручной проверки и найденные исправления – `docs/qa-report.md`.

Локально без Docker (нужен Go и запущенный postgres из compose): `cd backend && go test -p 1 -cover -coverpkg=./... ./...` с переменной `TEST_DATABASE_URL=postgres://buddy:buddy@localhost:5432/ai_buddy_test?sslmode=disable`. Без этой переменной интеграционные тесты пропускаются.

## Версии

| Компонент | Версия |
|---|---|
| Go | 1.25 (образ golang:1.25-alpine, GOTOOLCHAIN=auto) |
| PostgreSQL | 16 + pgvector (образ pgvector/pgvector:pg16) |
| Redis | 7 (redis:7-alpine) |
| chi | v5.2.2, go-chi/cors v1.2.2 |
| golang-jwt/jwt | v5.3.0 |
| pgx | v5.7.5 |
| golang.org/x/crypto (bcrypt) | v0.40.0 |
| testify | v1.10.0 |
| Node.js (сборка фронта) | 24 (образ node:24-alpine), раздача – nginx 1.27 |
| React | 19.3 |
| TypeScript | 7.0 |
| Vite | 8.3 |
| TanStack Router / Query | 1.170 / 5.107 |
| Zustand | 5.1 |
| Zod | 4.6 |
| Mantine | 9.8 |
| Vitest / MSW | 5.1 / 3.0 |

## Переменные окружения backend

| Переменная | По умолчанию | Назначение |
|---|---|---|
| PORT | 8080 | порт HTTP |
| DATABASE_URL | – (обязательна) | строка подключения к PostgreSQL |
| JWT_SECRET | – (обязательна, ≥16 символов) | ключ подписи JWT (HS256) |
| JWT_TTL | 24h | срок жизни токена |
| DEMO_MODE | false (в compose – true) | включает GET /api/auth/test-users |
| CORS_ORIGINS | http://localhost:5173 | разрешенные адреса фронта через запятую |
| LLM_PROVIDER | mock | mock (без ключа) или openai_compat |
| LLM_BASE_URL, LLM_API_KEY, LLM_MODEL | – | подключение OpenAI-совместимой модели |
| LLM_PROJECT | – | ID каталога для Yandex AI Studio (заголовок OpenAI-Project) |
| LLM_TIMEOUT | 20s | сколько ждать модель, потом офлайн-ответ |
| MOCK_SERVICES_URL | – (в compose – http://mock-services:8090) | система заявок; пусто – имитация внутри backend |
| MOCK_DECISION_DELAY | 45s | через сколько имитация «согласует» заявку (контейнер mock-services) |
| REQUEST_SYNC_INTERVAL | 5s | как часто забирать решения по заявкам |
| MOCK_SERVICES_PUBLIC_URL | – (в compose – http://localhost:8090) | адрес имитации для ссылок «открыть заявку в Jira» |

## API (этап 1 – авторизация)

Ошибки всегда в виде `{ "error": { "code": "...", "message": "..." } }`.

| Метод | Путь | Ответы |
|---|---|---|
| GET | /api/health | 200 `{status:"ok"}` |
| POST | /api/auth/login | 200 `{token, user}`; 400 validation_error; 401 invalid_credentials |
| GET | /api/auth/me | 200 User; 401 unauthorized |
| GET | /api/auth/test-users | 200 список; 404 not_found при DEMO_MODE=false |

## API (этап 2 – страница «Мой путь» и консоль наставника)

Подробный контракт с примерами: `docs/api-my-path.md`. Все запросы с заголовком `Authorization: Bearer <token>`.

| Метод | Путь | Кто | Что делает |
|---|---|---|---|
| GET | /api/path | стажер | весь экран «Мой путь» |
| POST | /api/path/tasks/{taskId}/complete | стажер | «Уже сделано» |
| GET | /api/interns | наставник, руководитель | стажеры с прогрессом |
| GET | /api/interns/{internId}/path | наставник, руководитель | путь стажера |
| PUT | /api/interns/{internId}/tasks/{taskId} | наставник, руководитель | статус задачи стажера |
| POST | /api/interns/{internId}/meetings | наставник, руководитель | назначить встречу |
| GET | /api/console/templates | наставник, руководитель | шаблон адаптации команды |
| GET | /api/console/action-kinds | все | какие действия умеет AI-buddy |
| POST/PUT/DELETE | /api/console/stages[/{id}] | наставник, руководитель | этапы |
| POST | /api/console/stages/{id}/tasks | наставник, руководитель | задача в этапе |
| PUT/DELETE | /api/console/tasks/{id} | наставник, руководитель | задача |
| POST | /api/console/stages/{id}/resources | наставник, руководитель | карточка боковой панели |
| DELETE | /api/console/resources/{id} | наставник, руководитель | карточка |

Права: стажер видит только себя; наставник – своих подопечных; руководитель – всех стажеров своей команды. Чужие данные отвечают 404 (не раскрываем, что они есть), запрещенные для роли действия – 403.

## API (этап 3 – справочник и цели стажировки)

Контракт с примерами: `docs/api-goals-kb.md`, спецификации: `specs/001-knowledge-search`, `specs/002-intern-goals`.

| Метод | Путь | Кто | Что делает |
|---|---|---|---|
| GET | /api/kb | все | разделы, статьи своей команды и общие, словарик |
| GET | /api/kb/articles/{id} | все | статья с текстом |
| GET | /api/kb/search?q= | все | поиск с русской морфологией |
| GET | /api/goals | стажер | свои цели |
| GET | /api/interns/{id}/goals | наставник, руководитель | цели стажера |
| POST | /api/goals/{id}/progress | стажер, наставник, руководитель | отметить прогресс |
| POST | /api/interns/{id}/goals | наставник, руководитель; стажер до согласования | добавить цель |
| PATCH/DELETE | /api/goals/{id} | так же | изменить, удалить |
| POST | /api/interns/{id}/goals/agree | руководитель | согласовать цели |

## API (этап 4 – чат с AI-buddy и заявки)

Контракт: `docs/api-chat-requests.md`, спецификация со схемой последовательности: `specs/003-access-request`.

| Метод | Путь | Что делает |
|---|---|---|
| POST | /api/chat/messages | вопрос помощнику (`text`, `action` и `taskId` задачи пути) |
| GET | /api/chat/messages | переписка текущего диалога (чат восстанавливается после перезагрузки) |
| POST | /api/chat/reset | новый диалог, старая переписка остается в базе |
| POST | /api/chat/actions/{id}/confirm | подтвердить заявку |
| POST | /api/chat/actions/{id}/cancel | отказаться |
| GET | /api/requests | мои заявки |
| GET | /api/me/contacts | наставник и руководитель пользователя, отделы компании |

Помощник отвечает из справочника со ссылкой на статью, готовит заявки на доступ, отгул и в отделы компании (оборудование, ИТ-поддержка, HR, бухгалтерия; создаются только после подтверждения), не оформляет доступ, запрещенный роли, показывает статусы и передает вопрос наставнику. Наставник переписку стажера с помощником не видит, только переданные ему вопросы.

## API (этап 5 – наставник, руководитель, поддержка)

Что готово на backend и контракты: `docs/api-backend-status.md`, спецификации: `specs/004-mentor-handoff`, `specs/005-service-requests`.

| Метод | Путь | Кто | Что делает |
|---|---|---|---|
| POST | /api/requests/{id}/withdraw | стажер | отозвать свою заявку на согласовании |
| GET, POST | /api/approvals, /api/approvals/{id} | руководитель, владелец ресурса | согласования |
| GET, POST | /api/mentor/feed, /api/mentor/feed/read | наставник | лента событий |
| GET, POST | /api/mentor/interns/{id}/checklist[/{itemId}] | наставник, руководитель | чек-лист наставника |
| GET | /api/mentor/metrics, /api/mentor/stats | наставник, руководитель | польза AI-buddy за неделю |
| GET, POST | /api/mentor/threads[/{internId}[/messages]] | наставник | переписка со стажерами, ответ в справочник |
| GET, POST | /api/me/mentor-thread[/messages] | стажер | переписка с наставником |
| GET, POST, DELETE | /api/materials[/{id}] | наставник | материалы и справочник команды |
| GET, POST, PATCH, DELETE | /api/mentor/meetings[/{id}] | наставник | расписание встреч |
| GET, PATCH | /api/mentor/automations[/{id}] | наставник | напоминания и сигналы |
| GET, PUT | /api/me/notes[/{eventId}] | стажер | конспекты встреч |
| GET, POST | /api/support/tickets | все | обращения в поддержку |

## База данных

Миграции 005–025 приводят схему к 3НФ и заполняют демо-данные: справочники (роли, сервисы, уровни прав read/edit/admin, проекты с владельцами), справочник знаний с полнотекстовым поиском, цели и OKR, заявки, чат, уведомления, события для метрик, feature flags. Всего 68 таблиц. В статьях справочника нет имен: метки `{{mentor}}`, `{{lead}}`, `{{team}}` заменяются на наставника, руководителя и команду того, кто читает. Даты в демо-данных считаются от дня применения миграций. Чтобы пересоздать демо-данные на сегодня: `docker compose down -v` и `docker compose up --build`.

## Структура backend

```
backend/
├── cmd/server/main.go        точка входа
├── internal/app              роутер, запуск и остановка сервера
├── internal/config           чтение переменных окружения
├── internal/database         подключение к БД и миграции
├── cmd/mockservices          имитация системы заявок (отдельный контейнер)
├── internal/handlers         HTTP-обработчики (вход: не больше 10 неверных паролей за 5 минут)
├── internal/httpx            единый формат JSON-ответов и ошибок
├── internal/integrations     адаптер к системе заявок (заменяется на настоящую)
├── internal/llm              модель: mock и OpenAI-совместимый API
├── internal/mockservices     имитация системы заявок (не входит в покрытие)
├── internal/middleware       проверка JWT, заголовки безопасности
├── internal/models           структуры ответов API
├── internal/repository       SQL-запросы
├── internal/services         бизнес-логика и права доступа
├── migrations/               SQL-миграции (встраиваются в бинарник, применяются при старте)
└── Dockerfile
```

## Если что-то не работает

- `port is already allocated` – порт занят. Поменяйте `POSTGRES_PORT` / `BACKEND_PORT` в `.env`.
- Тесты пишут `database "ai_buddy_test" does not exist` – база создалась до появления init-скрипта. Выполните `docker compose down -v` и запустите снова.
- Backend в статусе `unhealthy` – смотрите логи: `docker compose logs backend`.
- При входе «Слишком много неудачных попыток» – 10 неверных паролей подряд для этого логина. Подождите 5 минут или войдите под другим пользователем.
- Заявка навсегда «на согласовании» – контейнер `mock-services` перезапускали посреди сценария: имитация хранит заявки в памяти и забывает их. Начните с чистой базы: `docker compose down -v`, затем `docker compose up --build -d`.
