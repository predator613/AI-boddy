# AI-buddy – ИИ-наставник для стажеров (прототип, Hack&Change 2026, кейс Авито)

Все данные в проекте синтетические. Тестовые инструкции не являются регламентами Авито.

## Быстрый запуск

Нужен только Docker (Docker Desktop 4.x на Windows/macOS или Docker Engine 24+ с плагином compose v2 на Linux).

```bash
docker compose up --build -d
docker compose ps          # backend, postgres, redis должны быть healthy
```

Проверка: http://localhost:8080/api/health → `{"status":"ok"}`

Интерфейс: **http://localhost:3000** – войти под любым тестовым пользователем (карточки на странице входа). Стажер видит «Мой путь», наставник и руководитель – своих стажеров. Подробнее про фронтенд: `frontend/README.md`.

Чат, заявки и справочник backend пока не реализует: на эти запросы в браузере отвечает имитация (MSW), контракт для backend – `docs/api-frontend-pending.md`.

Остановить: `docker compose down`. Удалить и базу тоже (начать с чистого листа): `docker compose down -v`.

Файл `.env` не обязателен. Чтобы поменять порты или секреты: `cp .env.example .env` и отредактировать.

## Тестовые учетные данные

Пароль у всех: `demo123`

| Логин | Имя | Роль | Команда |
|---|---|---|---|
| intern_anna | Анна | стажер (intern), наставник – Игорь | Backend |
| mentor_igor | Игорь | наставник (mentor) | Backend |
| lead_oleg | Олег | руководитель (lead) | Backend |
| intern_pavel | Павел | стажер (intern) | Frontend |

## Порядок проверки (backend)

```bash
# 1. список тестовых пользователей (работает при DEMO_MODE=true)
curl http://localhost:8080/api/auth/test-users

# 2. вход
curl -X POST http://localhost:8080/api/auth/login -H "Content-Type: application/json" -d "{\"login\":\"intern_anna\",\"password\":\"demo123\"}"

# 3. текущий пользователь (подставить token из ответа)
curl http://localhost:8080/api/auth/me -H "Authorization: Bearer <token>"
```

Или в VS Code: файл `backend/api.http` (расширение REST Client).

## Тесты и покрытие

```bash
docker compose --profile test run --rm backend-tests
```

Команда поднимает PostgreSQL, прогоняет unit- и интеграционные тесты и пишет отчет:
- `backend/coverage/coverage.out` – профиль покрытия,
- `backend/coverage/coverage.html` – отчет для браузера.

Покрытие считается по всему прикладному коду backend (`-coverpkg=./...`), тесты и зависимости не учитываются. Текущее значение – около 89%.

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

## Структура backend

```
backend/
├── cmd/server/main.go        точка входа
├── internal/app              роутер, запуск и остановка сервера
├── internal/config           чтение переменных окружения
├── internal/database         подключение к БД и миграции
├── internal/handlers         HTTP-обработчики
├── internal/httpx            единый формат JSON-ответов и ошибок
├── internal/middleware       проверка JWT
├── internal/models           структуры (User, TestUser)
├── internal/repository       SQL-запросы
├── internal/services         логика входа, bcrypt, JWT
├── migrations/               SQL-миграции (встраиваются в бинарник, применяются при старте)
└── Dockerfile
```

## Если что-то не работает

- `port is already allocated` – порт занят. Поменяйте `POSTGRES_PORT` / `BACKEND_PORT` в `.env`.
- Тесты пишут `database "ai_buddy_test" does not exist` – база создалась до появления init-скрипта. Выполните `docker compose down -v` и запустите снова.
- Backend в статусе `unhealthy` – смотрите логи: `docker compose logs backend`.
