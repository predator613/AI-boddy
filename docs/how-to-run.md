# Как запустить AI-buddy

## Вариант 1. Все в Docker (как у жюри)

1. Открой Docker Desktop и дождись статуса «Engine running».
2. В корне репозитория: `docker compose up --build -d`.
3. Открой http://localhost:3000 (API бэка – http://localhost:8080).
4. Остановить: `docker compose down`.

Первая сборка идет 5–15 минут. Если шаг `npm ci` висит дольше 10 минут – используй вариант 2.

## Вариант 2. Бэк в Docker, фронт локально (быстрее при разработке)

1. В корне: `docker compose up --build -d backend mock-services` (база и Redis поднимутся сами).
2. В другой вкладке терминала:
   - `cd frontend`
   - `npm install` (только в первый раз или после изменения package.json)
   - `npm run dev`
3. Открой адрес из терминала, обычно http://localhost:5173.

## Вариант 3. Только фронт, без бэка (все данные – моки)

- `cd frontend`
- `npm run dev:mock`

Подходит, чтобы посмотреть интерфейс, когда Docker не запущен.

## Тестовые пользователи

Пароль у всех `demo123`. На экране входа их можно выбрать кнопкой.

- `intern_maria` – стажер, неделя 2 (главные сценарии)
- `intern_dima` – стажер, неделя 6
- `intern_liza` – стажер, первая неделя
- `mentor_igor` – наставник всех троих
- `lead_artemiy` – руководитель команды Backend

## Тесты и проверки фронта

Из папки `frontend`:

- `npm run test:run` – все тесты один раз. В конце должно быть `Tests  158 passed`.
- `npm run coverage` – тесты с отчетом о покрытии (папка `coverage`).
- `npx oxlint` – линтер, `npx tsc -b` – проверка типов.

Если терминал зависает или не хватает памяти: закрой Docker Desktop и браузер и запусти `npx vitest run --maxWorkers=1`.

## Живая модель вместо заготовок

Без ключа AI-buddy отвечает заготовками (`LLM_PROVIDER=mock`). Подключить любую OpenAI-совместимую модель:

1. `copy .env.example .env` (Linux/macOS: `cp`).
2. В `.env`: `LLM_PROVIDER=openai_compat`, `LLM_BASE_URL`, `LLM_API_KEY`, `LLM_MODEL` (для Яндекса еще `LLM_PROJECT`).
3. `docker compose up -d backend`.

Подробно, с примерами для Яндекса, Cloud.ru, OpenAI и локальных моделей – `docs/llm-setup.md`.
