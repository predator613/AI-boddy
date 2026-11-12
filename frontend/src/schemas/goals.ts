// Цели стажировки: экран «Мой путь» (карточка «Мои цели на стажировку») и карточка стажера у наставника.
// Точно повторяют ответы backend, контракт – docs/api-goals-kb.md.
import { z } from 'zod'
import { PersonRef } from './path'

export const TeamOkr = z.object({ id: z.string(), period: z.string(), title: z.string() })

export const GoalNote = z.object({
  author: PersonRef,
  progressPct: z.number(),
  note: z.string(),
  at: z.string(),
})

// Шаг цели: связан с задачей плана (закрывается вместе с ней) или подтверждается наставником
// или руководителем. Процент цели = доля выполненных шагов, считает backend.
export const GoalStep = z.object({
  id: z.string(),
  title: z.string(),
  source: z.enum(['task', 'mentor', 'lead']),
  stageCode: z.string().nullable(),
  taskId: z.string().nullable(),
  status: z.enum(['todo', 'in_progress', 'done']),
})

export const Goal = z.object({
  id: z.string(),
  position: z.number(),
  title: z.string(),
  criterion: z.string(),
  dueWeek: z.number().nullable(),
  progressPct: z.number().min(0).max(100),
  okr: TeamOkr.nullable(),
  lastUpdate: GoalNote.nullable(),
  updatedAt: z.string(),
  createdBy: PersonRef.nullable(),
  steps: z.array(GoalStep).default([]),
  // Компетенция из матрицы команды, которую развивает цель: «Обработка данных · L1».
  competency: z.string().nullable().default(null),
})

export const GoalsView = z.object({
  internId: z.string(),
  agreed: z.object({ by: PersonRef, at: z.string() }).nullable(),
  fromOkr: z.boolean(),
  goals: z.array(Goal),
  okrs: z.array(TeamOkr),
  canEdit: z.boolean(),
  canAgree: z.boolean(),
})

// Форма цели. dueWeek = 0 и okrId = '' при изменении очищают значение.
export const GoalInput = z.object({
  title: z.string().trim().min(1, 'Назови цель').max(120),
  criterion: z.string().trim().min(1, 'Как поймем, что цель достигнута?').max(160),
  dueWeek: z.number().int().min(0).max(52).optional(),
  okrId: z.string().optional(),
  competency: z.string().max(80).optional(),
  // Шаги цели: из них считается прогресс. При изменении список заменяется целиком.
  steps: z
    .array(
      z.object({
        title: z.string().trim().min(1).max(120),
        source: z.enum(['task', 'mentor', 'lead']),
      }),
    )
    .max(8)
    .optional(),
})
export const GoalPatch = GoalInput.partial()

export const ProgressInput = z.object({
  progressPct: z.number().int().min(0).max(100),
  note: z.string().max(500).optional(),
})

export type TGoal = z.infer<typeof Goal>
export type TGoalStep = z.infer<typeof GoalStep>
export type TGoalsView = z.infer<typeof GoalsView>
export type TTeamOkr = z.infer<typeof TeamOkr>
// Компетенции из матрицы роли (AvitoTech Playbook): к какой компетенции и уровню относится цель.
export const COMPETENCIES = [
  'Обработка данных · L1',
  'Визуализация и отчетность · L1',
  'Автоматизация · L1',
  'Код и ревью · L1',
  'Продуктовый подход',
  'Обмен знаниями',
  'Анализ потребностей',
]

export type TGoalInput = z.input<typeof GoalInput>
export type TGoalPatch = z.input<typeof GoalPatch>
export type TProgressInput = z.input<typeof ProgressInput>

/** Имя в творительном падеже для подписи «с Олегом», «с Ниной», «с Игорем». */
export function withName(name: string): string {
  const last = name.slice(-1)
  if (last === 'ь' || last === 'й') return `${name.slice(0, -1)}ем`
  if (last === 'а') return `${name.slice(0, -1)}ой`
  if (last === 'я') return `${name.slice(0, -1)}ей`
  if (/[а-яё]/i.test(last)) return `${name}ом`
  return name
}

/** Подпись бейджа: «согласованы с Олегом 4 нояб · от OKR команды». */
export function agreedLabel(view: TGoalsView): string {
  if (!view.agreed) return 'еще не согласованы с руководителем'
  const date = new Date(view.agreed.at)
    .toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })
    .replace('.', '')
  return `согласованы с ${withName(view.agreed.by.name)} ${date}${view.fromOkr ? ' · от OKR команды' : ''}`
}

/** Следующий шаг по клику на цель в макете: +25%, после 100% – снова 0. */
export const nextPct = (pct: number) => (pct >= 100 ? 0 : Math.min(100, pct + 25))
