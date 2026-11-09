# AI-buddy – frontend

React + TypeScript, TanStack Router и Query, Zustand, Zod, Mantine. Тесты – Vitest.

## Локальный запуск (разработка)

```bash
cd frontend
npm install
npm run dev        # http://localhost:5173
```

Backend для этого должен быть запущен (`docker compose up -d backend` из корня репозитория).

## Команды

| Команда | Что делает |
|---|---|
| `npm run dev` | dev-сервер с горячей перезагрузкой |
| `npm run build` | проверка типов и сборка в `dist/` |
| `npm test` | тесты в режиме наблюдения |
| `npm run coverage` | тесты + отчет о покрытии в `coverage/` (порог 80% строк) |
| `npm run lint` | проверка кода (oxlint) |
| `npm run format` | форматирование (Prettier) |

## Переменные окружения

См. `.env.example`. `VITE_API_URL` – адрес backend, `VITE_USE_MOCKS=true` – работать на моках MSW без backend.
