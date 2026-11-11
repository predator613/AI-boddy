# API, которого фронт ждет от backend

Эндпоинты из этого списка фронт уже вызывает, но в backend их пока нет. На них отвечает MSW прямо в браузере, остальные запросы идут в настоящий backend (`frontend/src/main.tsx`, переменная `VITE_PENDING_MOCKS`, по умолчанию включена). Список заглушек – `pendingHandlers` в `frontend/src/mocks/pending.ts`.

Сейчас в списке кабинет наставника и чат стажера с наставником (`mentorHandlers` из `frontend/src/mocks/mentorData.ts`), контракт – `docs/api-mentor-console.md`. Когда backend реализует ручку, удалите ее обработчик из `mentorHandlers`, код страниц менять не нужно.

Уже реализовано в backend:
- «Мой путь» и консоль плана адаптации – `docs/api-my-path.md`;
- цели стажировки и справочник – `docs/api-goals-kb.md`;
- чат с AI-buddy и заявки – `docs/api-chat-requests.md`.

Имитация этих разделов (`handlers.ts`, `consoleData.ts`, `assistantHandlers` в `pending.ts`) работает только в полном режиме без backend (`npm run dev:mock`) и в тестах.
