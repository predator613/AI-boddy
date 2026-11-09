import { setupServer } from 'msw/node'
import { handlers } from './handlers'

// MSW для тестов (Node.js)
export const server = setupServer(...handlers)
