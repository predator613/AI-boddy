// Подсказки AI-buddy по состоянию стажера: из незакрытых задач, заявок на согласовании и следующего шага.
// Статичные подсказки этапа из плана остаются, но те, что про уже сделанное, скрываются.
import type { TRequestItem } from '../schemas/assistant'
import type { TPathView } from '../schemas/path'

type Stage = TPathView['stages'][number]
type Task = Stage['tasks'][number]

const MAX = 4

// Частые слова, по которым нельзя судить, о чем подсказка.
const STOP = new Set([
  'досту',
  'получ',
  'задач',
  'этап',
  'запро',
  'заявк',
  'помог',
  'какой',
  'когда',
  'нужно',
])

/** Корни слов: первые 5 букв слов от 4 букв («gitlab» → «gitla»). */
const roots = (s: string) =>
  (s.toLowerCase().match(/[a-zа-яё0-9]{4,}/g) ?? [])
    .map((w) => w.slice(0, 5))
    .filter((w) => !STOP.has(w))

const lowerFirst = (s: string) => s.charAt(0).toLowerCase() + s.slice(1)

const uniq = (list: (string | null | undefined)[]) => [
  ...new Set(list.filter((s): s is string => Boolean(s && s.trim()))),
]

/** Подсказка по задаче: что спросить или попросить, чтобы сдвинуть ее. */
export function taskHint(t: Task): string | null {
  if (t.status === 'done') return null
  if (t.status === 'waiting') return `Что с задачей «${t.title}»?`
  const prompt = t.action?.payload?.prompt
  if (t.action?.kind === 'ask_buddy' && typeof prompt === 'string' && prompt.trim()) return prompt
  if (t.action) return `Помоги: ${lowerFirst(t.title)}`
  return null
}

/** Подсказки этапа: сначала по незакрытым задачам, потом из плана, кроме тех, что про сделанное. */
export function stageHints(stage: Stage): string[] {
  const open = stage.tasks.filter((t) => t.status !== 'done')
  const order = (t: Task) => (t.isNext ? 0 : t.status === 'waiting' ? 1 : 2)
  const dynamic = [...open].sort((a, b) => order(a) - order(b)).map(taskHint)
  const openRoots = new Set(open.flatMap((t) => roots(t.title)))
  const doneRoots = new Set(
    stage.tasks.filter((t) => t.status === 'done').flatMap((t) => roots(t.title)),
  )
  const fromPlan = stage.suggestions.filter(
    (s) => !roots(s).some((r) => doneRoots.has(r) && !openRoots.has(r)),
  )
  return uniq([...dynamic, ...fromPlan]).slice(0, MAX)
}

/** Подсказки на «Моем пути»: открытый этап или текущий. */
export function pathHints(path: TPathView, openCode?: string | null): string[] {
  const stage =
    (openCode ? path.stages.find((s) => s.code === openCode) : null) ??
    path.stages.find((s) => s.status === 'current')
  return stage ? stageHints(stage) : path.buddy.suggestions.slice(0, MAX)
}

const GENERIC = [
  'Какой мой следующий шаг?',
  'Как оформить отгул?',
  'Кто мой наставник?',
  'Как устроено код-ревью?',
]

/** Подсказки в чате: заявка на согласовании, следующий шаг, затем общие вопросы. */
export function chatHints(path?: TPathView | null, requests?: TRequestItem[] | null): string[] {
  const waiting = (requests ?? []).filter((r) => r.status === 'waiting')
  const next = path?.nextStep
  const nextTask = next
    ? path?.stages.flatMap((s) => s.tasks).find((t) => t.id === next.taskId)
    : undefined
  const nextHint =
    next && !(next.status === 'waiting' && waiting.length > 0)
      ? ((nextTask ? taskHint(nextTask) : null) ?? `Помоги с задачей «${next.title}»`)
      : null
  return uniq([
    waiting[0] ? `Что с заявкой №${waiting[0].number}?` : null,
    nextHint,
    ...GENERIC,
  ]).slice(0, MAX)
}
