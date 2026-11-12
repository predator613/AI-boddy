// Календарь стажера: встречи от наставника и команды + собственные записи стажера.
// Контракт – docs/api-intern-calendar.md. Пока backend его не реализовал, отвечают моки
// (src/mocks/mentorData.ts, режим VITE_PENDING_MOCKS).
import { z } from 'zod'
import { msk } from '../lib/msk'

export const EventKind = z.enum([
  'mentor', // встреча с наставником (1:1, разбор)
  'team', // командная встреча
  'lead', // с руководителем
  'intro', // знакомство с коллегой
  'focus', // своя запись: фокус-время
  'study', // своя запись: учеба
  'personal', // своя запись: личное
])

export const OWN_KINDS = ['focus', 'study', 'personal'] as const

export const CalendarEvent = z.object({
  id: z.string(),
  title: z.string(),
  kind: EventKind,
  startsAt: z.string(),
  durationMin: z.number(),
  withName: z.string(),
  place: z.string(),
  note: z.string(),
  // pending – ждет ответа стажера, accepted – придет, move_requested – попросил перенести,
  // own – собственная запись
  status: z.enum(['pending', 'accepted', 'move_requested', 'own']),
  shared: z.boolean(), // своя запись видна наставнику
})

export const CalendarView = z.object({ events: z.array(CalendarEvent) })

export const EntryInput = z
  .object({
    title: z.string().trim().min(1, 'Напиши, что планируешь').max(120, 'Не длиннее 120 символов'),
    kind: z.enum(OWN_KINDS),
    startsAt: z.string(),
    durationMin: z.number().int().min(15).max(480),
    shareWithMentor: z.boolean(),
  })
  .refine(
    (v) => {
      const s = msk(v.startsAt)
      return s.hour * 60 + s.minute + v.durationMin <= 18 * 60
    },
    { message: 'Запись выходит за рабочий день (до 18:00)', path: ['durationMin'] },
  )

export const AnswerInput = z.object({
  answer: z.enum(['accept', 'move']),
  reason: z.string().max(200).optional(),
})

export type TEventKind = z.infer<typeof EventKind>
export type TCalendarEvent = z.infer<typeof CalendarEvent>
export type TEntryInput = z.input<typeof EntryInput>
export type TAnswerInput = z.infer<typeof AnswerInput>

// Конспект встречи (чек-лист: «заведи конспект встреч с наставником»). Виден только стажеру.
export const MeetingNote = z.object({
  eventId: z.string(),
  text: z.string(),
  updatedAt: z.string(),
})
export const NoteInput = z.object({ text: z.string().max(4000, 'Не больше 4000 символов') })
export type TMeetingNote = z.infer<typeof MeetingNote>
