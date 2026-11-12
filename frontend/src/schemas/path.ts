// Zod-схемы страницы «Мой путь» и консоли наставника. Точно повторяют ответы backend.
import { z } from 'zod'

export const TaskStatus = z.enum(['todo', 'in_progress', 'waiting', 'done'])
export const StageStatus = z.enum(['done', 'current', 'todo'])
export const TagKind = z.enum(['ok', 'wait', 'none', 'mentor'])
export const ResourceKind = z.enum(['person', 'access', 'tool', 'article', 'link'])
export const ActionKind = z.enum([
  'access_request',
  'leave_request',
  'ask_buddy',
  'open_article',
  'join_chats',
])

export const Action = z.object({
  kind: ActionKind,
  payload: z.record(z.string(), z.unknown()),
})

export const PersonRef = z.object({ id: z.string(), name: z.string() })

export const Milestone = z.object({ week: z.number(), title: z.string() })

export const Meeting = z.object({
  id: z.string(),
  title: z.string(),
  startsAt: z.string(), // ISO-дата, например 2026-11-12T15:00:00+03:00
  hostId: z.string(),
  hostName: z.string(),
})

// Подробности задачи для раскрытия по клику: что сделать, кто поможет, когда задача готова,
// в какую цель стажировки она идет. Поле необязательное: без него показываем только note.
export const TaskDetails = z.object({
  about: z.string(),
  helper: z.string(),
  doneWhen: z.string(),
  goal: z.string().nullable(),
})

export const TaskView = z.object({
  id: z.string(),
  title: z.string(),
  note: z.string(),
  status: TaskStatus,
  isNext: z.boolean(),
  dueDate: z.string().nullable(),
  action: Action.nullable(),
  details: TaskDetails.nullable().default(null),
})

export const ResourceView = z.object({
  id: z.string(),
  kind: ResourceKind,
  name: z.string(),
  note: z.string(),
  tag: z.string(),
  tagKind: TagKind,
})

export const StageView = z.object({
  id: z.string(),
  code: z.string(), // "01", "02"...
  title: z.string(),
  // Когда проходит этап по чек-листу: «неделя 1», «месяцы 1–2», «после 3 месяцев».
  period: z.string().nullable().default(null),
  color: z.string(),
  tip: z.string(),
  status: StageStatus,
  doneCount: z.number(),
  totalCount: z.number(),
  suggestions: z.array(z.string()),
  tasks: z.array(TaskView),
  resources: z.array(ResourceView),
})

export const Progress = z.object({
  done: z.number(),
  inProgress: z.number(),
  todo: z.number(),
  total: z.number(),
  percent: z.number(),
})

export const PathView = z.object({
  intern: z.object({
    id: z.string(),
    name: z.string(),
    teamName: z.string(),
    mentor: PersonRef.nullable(),
  }),
  week: z.object({
    current: z.number(),
    total: z.number(),
    day: z.number(),
    startDate: z.string(),
    milestones: z.array(Milestone),
  }),
  progress: Progress,
  stages: z.array(StageView),
  nextStep: z
    .object({
      taskId: z.string(),
      stageId: z.string(),
      stageCode: z.string(),
      title: z.string(),
      status: TaskStatus,
      dueDate: z.string().nullable(),
      dueInDays: z.number().nullable(),
      action: Action.nullable(),
    })
    .nullable(),
  nextMeeting: Meeting.nullable(),
  buddy: z.object({ text: z.string(), suggestions: z.array(z.string()) }),
})

export const InternSummary = z.object({
  id: z.string(),
  login: z.string(),
  name: z.string(),
  teamName: z.string(),
  mentor: PersonRef.nullable(),
  week: z.number(),
  progress: Progress,
  currentStageCode: z.string(),
  currentStage: z.string(),
})

// ---------- консоль наставника ----------

export const TaskTemplate = z.object({
  id: z.string(),
  stageId: z.string(),
  position: z.number(),
  title: z.string(),
  note: z.string(),
  dueDay: z.number().nullable(),
  action: Action.nullable(),
})

export const ResourceTemplate = z.object({
  id: z.string(),
  stageId: z.string(),
  position: z.number(),
  kind: ResourceKind,
  name: z.string(),
  note: z.string(),
  tag: z.string(),
  tagKind: TagKind,
  taskId: z.string().nullable(),
})

export const StageTemplate = z.object({
  id: z.string(),
  teamId: z.string(),
  position: z.number(),
  title: z.string(),
  color: z.string(),
  tip: z.string(),
  suggestions: z.array(z.string()),
  tasks: z.array(TaskTemplate).optional(),
  resources: z.array(ResourceTemplate).optional(),
})

// Формы консоли: та же валидация, что на backend, чтобы ошибки ловились до отправки.
export const StageInput = z.object({
  title: z.string().trim().min(1, 'Введите название').max(80),
  color: z
    .string()
    .regex(/^#[0-9A-Fa-f]{6}$/, 'Цвет в формате #RRGGBB')
    .default('#C9FFBF'),
  tip: z.string().max(300).default(''),
  suggestions: z.array(z.string().max(80)).max(6).default([]),
  position: z.number().int().min(1).optional(),
})

export const TaskInput = z.object({
  title: z.string().trim().min(1, 'Введите название').max(120),
  note: z.string().max(300).default(''),
  dueDay: z.number().int().min(1).max(730).nullable().default(null),
  action: Action.nullable().default(null),
  position: z.number().int().min(1).optional(),
})

export const ResourceInput = z.object({
  kind: ResourceKind,
  name: z.string().trim().min(1, 'Введите название').max(80),
  note: z.string().max(120).default(''),
  tag: z.string().max(40).default(''),
  tagKind: TagKind.default('none'),
  taskId: z.string().nullable().default(null),
})

export const SetStatusInput = z.object({
  status: TaskStatus,
  note: z.string().max(200).nullable().optional(),
})

export const MeetingInput = z.object({
  title: z.string().max(120).optional(),
  startsAt: z.string(), // ISO
})

export type TPathView = z.infer<typeof PathView>
export type TStageView = z.infer<typeof StageView>
export type TTaskView = z.infer<typeof TaskView>
export type TInternSummary = z.infer<typeof InternSummary>
export type TStageTemplate = z.infer<typeof StageTemplate>
export type TTaskTemplate = z.infer<typeof TaskTemplate>
export type TResourceTemplate = z.infer<typeof ResourceTemplate>
export type TMeeting = z.infer<typeof Meeting>
export type TStageInput = z.input<typeof StageInput>
export type TTaskInput = z.input<typeof TaskInput>
export type TResourceInput = z.input<typeof ResourceInput>
export type TSetStatusInput = z.infer<typeof SetStatusInput>
export type TMeetingInput = z.infer<typeof MeetingInput>
