// Обращения в поддержку («Сообщить о проблеме»). Контракт – docs/api-support.md.
import { z } from 'zod'

export const SupportTicket = z.object({
  id: z.string(),
  number: z.string(), // SUP-1042
  type: z.string(),
  details: z.string(),
  status: z.enum(['new', 'in_progress', 'done']),
  createdAt: z.string(),
  answer: z.string().nullable(),
})

export const SupportInput = z.object({
  type: z.string().min(1),
  details: z.string().trim().min(1).max(2000),
  internId: z.string().nullable(),
  attachDialog: z.boolean(),
})

export type TSupportTicket = z.infer<typeof SupportTicket>
export type TSupportInput = z.infer<typeof SupportInput>
