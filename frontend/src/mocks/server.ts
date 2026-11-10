import { setupServer } from 'msw/node'
import { allHandlers } from './handlers'

// MSW для тестов (Node.js)
export const server = setupServer(...allHandlers)
