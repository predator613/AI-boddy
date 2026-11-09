# AI-buddy – frontend

React + TypeScript, TanStack Router и Query, Zustand, Zod, Mantine. Тесты – Vitest.

## Локальный запуск (разработка)

```bash
cd frontend
npm install
npm run dev        # http://localhost:5173
```

Backend для этого должен быть запущен (`docker compose up -d backend` из корня репозитория). Без backend: `npm run dev:mock`.

## Команды

| Команда            | Что делает                                               |
| ------------------ | -------------------------------------------------------- |
| `npm run dev`      | dev-сервер с горячей перезагрузкой                       |
| `npm run build`    | проверка типов и сборка в `dist/`                        |
| `npm test`         | тесты в режиме наблюдения                                |
| `npm run coverage` | тесты + отчет о покрытии в `coverage/` (порог 80% строк) |
| `npm run lint`     | проверка кода (oxlint)                                   |
| `npm run format`   | форматирование (Prettier)                                |

## Переменные окружения

См. `.env.example`. `VITE_API_URL` – адрес backend, `VITE_USE_MOCKS=true` – работать на моках MSW без backend.

## Структура

```
src/
├── api/client.ts      единый API-клиент: адрес из VITE_API_URL, токен, разбор ошибок, 401 → выход
├── api/auth.ts        login(), getMe(), getTestUsers()
├── schemas/auth.ts    Zod-схемы ответов backend и типы
├── store/authStore.ts Zustand: токен и пользователь (сохраняются в localStorage)
├── hooks/useLogin.ts  вход через TanStack Query + тексты ошибок
├── components/        LoginForm, TestUserPicker, RoleBadge
├── routes/            страницы (TanStack Router, file-based)
│   ├── login.tsx      вход
│   ├── _auth.tsx      layout защищенных страниц (без токена → /login)
│   └── _auth/index.tsx главная
└── mocks/             MSW: имитация backend по контракту API
```
