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

### AI-buddy без ключа и с моделью

По умолчанию `LLM_PROVIDER=mock` – офлайн-режим без ключа и без сети: помощник определяет намерение по ключевым словам и отвечает фрагментом справочника. Весь сценарий (права, подтверждение, заявка, статус) работает так же, как с моделью.

Подключить модель – любой OpenAI-совместимый API (Yandex AI Studio, Cloud.ru, Avito A-Vibe через Ollama; подробно – `docs/llm-setup.md`). В `.env`:

```
LLM_PROVIDER=openai_compat
LLM_BASE_URL=<адрес API, оканчивается на /v1>
LLM_API_KEY=<ключ>
LLM_MODEL=<имя модели>
LLM_PROJECT=<ID каталога, только для Yandex AI Studio>
```

и `docker compose up -d backend`. Модель должна поддерживать function calling (tools). Сравнение провайдеров и пошаговое получение ключа: `docs/llm-setup.md` (рекомендуется Yandex AI Studio). GPU не нужен: модель вызывается по API. Если модель не ответила за `LLM_TIMEOUT` (20 с), ответ собирается офлайн.

Остановить: `docker compose down`. Удалить и базу тоже (начать с чистого листа): `docker compose down -v`.

Файл `.env` не обязателен. Чтобы поменять порты или секреты: `cp .env.example .env` и отредактировать.

## Тестовые учетные данные

Пароль у всех: `demo123`

| Логин | Имя | Роль | Команда |
|---|---|---|---|
| intern_anna | Анна | стажер, 2-я неделя, этап «Доступы», цели согласованы | Backend |
| intern_dima | Дмитрий | стажер, 6-я неделя, этап «Первые задачи», есть просрочка | Backend |
| intern_liza | Лиза | стажер, 1-я неделя, цели еще не согласованы | Backend |
| mentor_igor | Игорь | наставник Анны, Дмитрия и Лизы | Backend |
| lead_oleg | Олег | руководитель, согласует цели и GitLab backend/auth-service | Backend |
| intern_pavel | Павел | стажер без наставника | Frontend |
| lead_nina | Нина | руководитель | Frontend |

Кирилл (DevOps, роль staff) – согласующий тестового стенда и логов, в приложение не входит.

## Порядок проверки (backend)

```bash
# 1. список тестовых пользователей (работает при DEMO_MODE=true)
curl http://localhost:8080/api/auth/test-users

# 2. вход
curl -X POST http://localhost:8080/api/auth/login -H "Content-Type: application/json" -d "{\"login\":\"intern_anna\",\"password\":\"demo123\"}"

# 3. текущий пользователь (подставить token из ответа)
curl http://localhost:8080/api/auth/me -H "Authorization: Bearer <token>"

# 4. цели Анны и отметка прогресса
curl http://localhost:8080/api/goals -H "Authorization: Bearer <token>"
curl -X POST http://localhost:8080/api/goals/f3000000-0000-4000-8000-000000000002/progress -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d "{\"progressPct\":35}"

# 5. справочник и поиск
curl http://localhost:8080/api/kb -H "Authorization: Bearer <token>"
curl -G http://localhost:8080/api/kb/search --data-urlencode "q=как взять отгул" -H "Authorization: Bearer <token>"

# 6. главный сценарий: доступ к GitLab через AI-buddy (actionId – из карточки confirm)
curl -X POST http://localhost:8080/api/chat/messages -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d "{\"text\":\"Запроси доступ к GitLab\"}"
curl -X POST http://localhost:8080/api/chat/actions/<actionId>/confirm -H "Authorization: Bearer <token>"
curl http://localhost:8080/api/requests -H "Authorization: Bearer <token>"   # через 45 с статус done
```

В интерфейсе то же самое: войти как `intern_anna` → «Мой путь» → «Оформить с AI-buddy» у задачи «Получить доступ к GitLab» → «Подтвердить». Через 45 секунд (`MOCK_DECISION_DELAY`) на странице «Заявки» статус «выполнена», а пункт пути закрыт. Отказ согласующего можно показать вручную: `POST http://localhost:8090/v1/tickets/<ACC-номер>/decision` с `{"approve":false,"reason":"…"}` (номер – в `backend/api.http`).

Или в VS Code: файл `backend/api.http` (расширение REST Client).

## Тесты и покрытие

```bash
docker compose --profile test run --rm backend-tests
```

Команда поднимает PostgreSQL, прогоняет unit- и интеграционные тесты и пишет отчет:
- `backend/coverage/coverage.out` – профиль покрытия,
- `backend/coverage/coverage.html` – отчет для браузера.

Покрытие считается по всему прикладному коду backend, кроме имитации корпоративных систем (`internal/mockservices`, `cmd/`). Текущее значение – около 88%.

Общий отчет по фронту и бэку одним числом (покрытые строки вместе, а не среднее двух процентов):

```bash
docker compose --profile test run --rm backend-tests
docker compose --profile test run --rm frontend-tests
python scripts/coverage_summary.py      # итог около 90%, отчет в coverage/summary.md
```

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
| GET | /api/chat/messages | последние сообщения |
| POST | /api/chat/actions/{id}/confirm | подтвердить заявку |
| POST | /api/chat/actions/{id}/cancel | отказаться |
| GET | /api/requests | мои заявки |

Помощник отвечает из справочника со ссылкой на статью, готовит заявки на доступ и отгул (создаются только после подтверждения), не оформляет доступ, запрещенный роли, показывает статусы и передает вопрос наставнику. Наставник переписку стажера с помощником не видит, только переданные ему вопросы.

## База данных

Миграции 005–012 приводят схему к 3НФ и заполняют демо-данные: справочники (роли, сервисы, уровни прав read/edit/admin, проекты с владельцами), справочник знаний с полнотекстовым поиском, цели и OKR, заявки, чат, уведомления, события для метрик, feature flags. Всего 55 таблиц. Даты в демо-данных считаются от дня применения миграций. Чтобы пересоздать демо-данные на сегодня: `docker compose down -v` и `docker compose up --build`.

## Структура backend

```
backend/
├── cmd/server/main.go        точка входа
├── internal/app              роутер, запуск и остановка сервера
├── internal/config           чтение переменных окружения
├── internal/database         подключение к БД и миграции
├── cmd/mockservices          имитация системы заявок (отдельный контейнер)
├── internal/handlers         HTTP-обработчики
├── internal/httpx            единый формат JSON-ответов и ошибок
├── internal/integrations     адаптер к системе заявок (заменяется на настоящую)
├── internal/llm              модель: mock и OpenAI-совместимый API
├── internal/mockservices     имитация системы заявок (не входит в покрытие)
├── internal/middleware       проверка JWT
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
