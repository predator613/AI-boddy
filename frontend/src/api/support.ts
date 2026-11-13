// API обращений в поддержку (docs/api-support.md).
import { z } from 'zod'
import { apiRequest } from './client'
import { SupportInput, SupportTicket, type TSupportInput } from '../schemas/support'

/** GET /api/support/tickets – мои обращения, новые сверху. */
export const getTickets = () => apiRequest('/api/support/tickets', z.array(SupportTicket))

/** POST /api/support/tickets – новое обращение, номер выдает сервис заявок. */
export const createTicket = (input: TSupportInput) =>
  apiRequest('/api/support/tickets', SupportTicket, {
    method: 'POST',
    body: SupportInput.parse(input),
  })
