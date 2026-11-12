// Лента изменений для наставника (колокольчик) и согласования руководителя.
// Контракт – docs/api-lead-feed.md. Пока backend их не реализовал, отвечают моки src/mocks/leadData.ts.
import { z } from 'zod'

export const FeedItem = z.object({
  id: z.string(),
  actor: z.object({
    name: z.string(),
    kind: z.enum(['intern', 'mentor', 'lead', 'buddy', 'other']),
  }),
  text: z.string(),
  at: z.string(),
  read: z.boolean(),
})

export const Feed = z.object({ items: z.array(FeedItem), unread: z.number() })

export const Approval = z.object({
  id: z.string(),
  number: z.string(),
  internId: z.string(),
  internName: z.string(),
  title: z.string(),
  fields: z.array(z.object({ label: z.string(), value: z.string() })),
  createdAt: z.string(),
  status: z.enum(['waiting', 'approved', 'rejected']),
  rejectReason: z.string().nullable(),
})

export const Decision = z.discriminatedUnion('decision', [
  z.object({ decision: z.literal('approve') }),
  z.object({
    decision: z.literal('reject'),
    reason: z.string().trim().min(1, 'Напиши причину: ее увидит стажер').max(300),
  }),
])

export type TFeed = z.infer<typeof Feed>
export type TFeedItem = z.infer<typeof FeedItem>
export type TApproval = z.infer<typeof Approval>
export type TDecision = z.input<typeof Decision>

// Чек-лист наставника по стажеру (по чек-листу стажировки Авито): что сделать и к какому сроку.
export const ChecklistItem = z.object({
  id: z.string(),
  title: z.string(),
  hint: z.string(),
  // С какой недели стажировки пункт актуален: 1, 2, 13 (экватор), 24 (итоги).
  fromWeek: z.number(),
  period: z.string(),
  done: z.boolean(),
})

export const MentorChecklist = z.object({
  internId: z.string(),
  items: z.array(ChecklistItem),
})

export type TChecklistItem = z.infer<typeof ChecklistItem>
export type TMentorChecklist = z.infer<typeof MentorChecklist>
