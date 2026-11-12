// Zod-схемы кабинета наставника: переписка, материалы, расписание, автоматизации, статистика.
// Контракт описан в docs/api-mentor-console.md. Пока backend их не реализовал, отвечают моки
// из src/mocks/mentorData.ts (режим VITE_PENDING_MOCKS).
import { z } from 'zod'

export const MentorStats = z.object({
  periodDays: z.number(),
  answeredByBuddy: z.number(),
  handedOff: z.number(),
  savedMinutes: z.number(),
  mentorHoursSpent: z.number(),
  mentorHoursPlan: z.number(),
})

export const UpdateType = z.enum(['plan', 'material', 'meeting'])

export const ThreadMessage = z.object({
  id: z.string(),
  // text – обычное сообщение, handoff – AI-buddy передал вопрос наставнику,
  // update – карточка изменения от наставника (план, материал, встреча)
  kind: z.enum(['text', 'handoff', 'update']),
  author: z.enum(['intern', 'mentor', 'buddy']),
  authorName: z.string(),
  text: z.string(),
  at: z.string(),
  update: z
    .object({
      type: UpdateType,
      title: z.string(),
      detail: z.string(),
      link: z.string().nullable(),
      acked: z.boolean(),
    })
    .nullable(),
})

export const Thread = z.object({
  internId: z.string(),
  internName: z.string(),
  mentor: z.object({ id: z.string(), name: z.string() }),
  stageCode: z.string(),
  stageTitle: z.string(),
  messages: z.array(ThreadMessage),
})

export const ThreadSummary = z.object({
  internId: z.string(),
  internName: z.string(),
  stageCode: z.string(),
  stageTitle: z.string(),
  lastText: z.string(),
  lastAt: z.string(),
  unread: z.number(),
})

export const ReplyInput = z.object({
  text: z.string().trim().min(1, 'Напиши сообщение').max(2000),
  addToKb: z.boolean().default(false),
})

export const MaterialKind = z.enum(['pdf', 'docx', 'md', 'link', 'answer'])

export const Material = z.object({
  id: z.string(),
  title: z.string(),
  kind: MaterialKind,
  stageCode: z.string().nullable(),
  // Раздел справочника, в котором стажеры увидят материал.
  section: z.string().nullable().default(null),
  useInAi: z.boolean(),
  authorName: z.string(),
  createdAt: z.string(),
  sizeLabel: z.string(),
})

export const MaterialInput = z
  .object({
    title: z.string().trim().min(1, 'Введи название').max(120),
    stageCode: z.string().default(''),
    section: z.string().trim().max(60).default(''),
    useInAi: z.boolean().default(true),
    notify: z.boolean().default(true),
    url: z.string().trim().default(''),
    file: z.instanceof(File).nullable().default(null),
  })
  .refine((v) => v.file !== null || /^https?:\/\/\S+$/.test(v.url), {
    message: 'Выбери файл или вставь ссылку',
    path: ['file'],
  })

export const MeetingKind = z.enum(['one_on_one', 'review', 'team'])

export const MentorMeeting = z.object({
  id: z.string(),
  title: z.string(),
  internId: z.string().nullable(), // null – встреча со всеми стажерами
  withName: z.string(),
  kind: MeetingKind,
  startsAt: z.string(),
  durationMin: z.number(),
})

export const MeetingCreateInput = z.object({
  internId: z.string().min(1),
  title: z.string().trim().min(1, 'Введи тему').max(120),
  startsAt: z.string(),
  durationMin: z.number().int().min(15).max(240),
  notify: z.boolean(),
})

export const Automation = z.object({
  id: z.string(),
  kind: z.enum(['deadline_reminder', 'weekly_goal_checkin', 'overdue_alert', 'approver_reminder']),
  title: z.string(),
  note: z.string(),
  enabled: z.boolean(),
})

export type TMentorStats = z.infer<typeof MentorStats>
export type TThreadMessage = z.infer<typeof ThreadMessage>
export type TThread = z.infer<typeof Thread>
export type TThreadSummary = z.infer<typeof ThreadSummary>
export type TReplyInput = z.input<typeof ReplyInput>
export type TMaterial = z.infer<typeof Material>
export type TMaterialInput = z.input<typeof MaterialInput>
export type TMentorMeeting = z.infer<typeof MentorMeeting>
export type TMeetingCreateInput = z.infer<typeof MeetingCreateInput>
export type TAutomation = z.infer<typeof Automation>
