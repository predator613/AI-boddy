import { setupWorker } from 'msw/browser'
import { allHandlers } from './handlers'
import { pendingHandlers } from './pending'

// Полный режим (npm run dev:mock): все запросы обрабатывает MSW, backend не нужен.
export const fullWorker = () => setupWorker(...allHandlers)

// Режим «вместе с backend»: MSW отвечает только на эндпоинты, которых в backend еще нет.
export const pendingWorker = () => setupWorker(...pendingHandlers)
