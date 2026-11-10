# AI-buddy – frontend

React + TypeScript, TanStack Router и Query, Zustand, Zod, Mantine. Шрифты Onest и Caveat, иконки Tabler. Тесты – Vitest.

## Экраны

| Адрес | Экран | Кто видит |
|---|---|---|
| `/login` | вход, быстрый вход под тестовыми пользователями | все |
| `/` | «Мой путь»: кольцо прогресса, этапы (раскрываются), следующий шаг, наставник, панель AI-buddy | стажер |
| `/` | «Мои стажеры»: прогресс каждого стажера | наставник, руководитель |
| `/interns/:id` | путь стажера, смена статусов задач | наставник, руководитель |
| `/chat` | чат с помощником: ответы с источником, подтверждение действия, созданная заявка | все |
| `/kb` | справочник: разделы, поиск, статьи текущего этапа, словарик | все |
| `/requests` | мои заявки: фильтры, детали, история | стажер |
| `/support` | поддержка: кому написать, частые вопросы, сообщить о проблеме | все |
| `/profile` | профиль и настройки (уведомления, подробность ответов) | все |

## Локальный запуск

```bash
cd frontend
npm install
npm run dev        # http://localhost:5173, нужен backend: docker compose up -d backend
npm run dev:mock   # без backend: все запросы обрабатывают моки MSW
```

## Как фронт работает с backend

- Все запросы идут через один модуль `src/api/client.ts` (`apiFetch` / `apiRequest`).
- «Мой путь», стажеры и вход – настоящий backend (контракт: `docs/api-my-path.md`).
- Чат, заявки и справочник backend пока не умеет. На них в браузере отвечает MSW (`src/mocks/pending.ts`), остальные запросы проходят в backend. Контракт для backend: `docs/api-frontend-pending.md`. Выключить: `VITE_PENDING_MOCKS=false`.

## Команды

| Команда | Что делает |
|---|---|
| `npm run dev` | dev-сервер с горячей перезагрузкой |
| `npm run dev:mock` | dev-сервер на моках, backend не нужен |
| `npm run build` | проверка типов и сборка в `dist/` |
| `npm test` | тесты в режиме наблюдения |
| `npm run coverage` | тесты + отчет о покрытии в `coverage/` (порог 80% строк) |
| `npm run lint` | проверка кода (oxlint) |
| `npm run format` | форматирование (Prettier) |

## Структура

```
src/
├── api/          client.ts – единый клиент; auth.ts, path.ts, assistant.ts – запросы по разделам
├── schemas/      Zod-схемы ответов backend и форм
├── hooks/        хуки TanStack Query (usePath, useLogin)
├── store/        Zustand: authStore (токен), settingsStore (настройки)
├── lib/          pathUi (цвета и подписи статусов), nav (меню по ролям), errors
├── components/   AppRail (меню-иконки), Mascot (5 состояний), path/* (кольцо, этапы, задачи), chat/*
├── pages/        экраны
├── routes/       маршруты TanStack Router (file-based)
├── content/      статичные тексты (поддержка)
├── mocks/        MSW: handlers (все), pending (то, чего нет в backend), pathData
├── styles/       анимации и общие стили
└── theme.ts      тема Mantine: палитра и шрифты кейса
```

## Переменные окружения

См. `.env.example`: `VITE_API_URL` – адрес backend, `VITE_USE_MOCKS=true` – только моки, `VITE_PENDING_MOCKS=false` – выключить моки недостающих эндпоинтов.
