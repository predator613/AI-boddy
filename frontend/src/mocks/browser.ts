import { setupWorker } from 'msw/browser'
import { allHandlers } from './handlers'
import { pendingHandlers } from './pending'

// Полный режим (npm run dev:mock): все запросы обрабатывает MSW, backend не нужен.
export const fullWorker = () => setupWorker(...allHandlers)

// Режим «вместе с backend»: MSW отвечает только на эндпоинты, которых в backend еще нет.
// null – перехватывать нечего, MSW не запускается.
export const pendingWorker = () => (pendingHandlers.length ? setupWorker(...pendingHandlers) : null)
