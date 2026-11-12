// Расчеты главного экрана наставника: что нужно от наставника по каждому стажеру.
import type { TGoalsView } from '../schemas/goals'
import type { TThreadSummary } from '../schemas/mentor'
import type { TPathView } from '../schemas/path'

export const glass = ([a, b]: [string, string]) =>
  `radial-gradient(60% 120% at 82% 30%, ${b} 0%, transparent 70%), ` +
  `radial-gradient(50% 140% at 30% 90%, ${a} 0%, transparent 72%), ` +
  `linear-gradient(100deg, rgba(255,255,255,.92) 0%, rgba(255,255,255,.55) 100%), ${a}`

export const DAY = 86_400_000

/** Сколько полных дней прошло после срока (0 – срок сегодня или еще не наступил). */
export function daysLate(due: string | null, now = new Date()): number {
  if (!due) return 0
  const d = new Date(`${due.slice(0, 10)}T23:59:59`)
  return Math.max(0, Math.floor((now.getTime() - d.getTime()) / DAY) + 1)
}

export const plural = (n: number, one: string, few: string, many: string) => {
  const a = n % 10
  const b = n % 100
  if (a === 1 && b !== 11) return one
  if (a >= 2 && a <= 4 && (b < 12 || b > 14)) return few
  return many
}

export type TodoKind = 'question' | 'late' | 'waiting' | 'goals'

export interface Todo {
  kind: TodoKind
  chip: string
  title: string
  text: string
  cta: 'chat' | 'path' | null
}

export const DOT: Record<TodoKind, string> = {
  question: '#965EEB',
  late: '#FF4053',
  waiting: '#00AAFF',
  goals: '#00AAFF',
}

/** Что от наставника нужно по стажеру: вопросы без ответа, просрочки, долгие согласования, цели. */
export function buildTodos(
  thread: TThreadSummary | undefined,
  path: TPathView | undefined,
  goals: TGoalsView | undefined,
  now = new Date(),
): Todo[] {
  const out: Todo[] = []
  if (thread && thread.unread > 0) {
    const n = thread.unread
    out.push({
      kind: 'question',
      chip: `${n} ${plural(n, 'вопрос', 'вопроса', 'вопросов')}`,
      title: `${n} ${plural(n, 'сообщение ждет', 'сообщения ждут', 'сообщений ждут')} ответа`,
      text: `«${thread.lastText}»`,
      cta: 'chat',
    })
  }
  // Просрочки и долгие согласования собираем в одну строку на вид: иначе список не прочитать.
  const late: { title: string; days: number }[] = []
  const waiting: { title: string; days: number }[] = []
  for (const t of path?.stages.flatMap((s) => s.tasks) ?? []) {
    if (t.status === 'done') continue
    const days = daysLate(t.dueDate, now)
    if (!days) continue
    ;(t.status === 'waiting' ? waiting : late).push({ title: t.title, days })
  }
  const names = (list: { title: string }[]) =>
    list
      .slice(0, 2)
      .map((x) => `«${x.title}»`)
      .join(', ') + (list.length > 2 ? ` и еще ${list.length - 2}` : '')
  if (late.length === 1) {
    const d = late[0]!.days
    out.push({
      kind: 'late',
      chip: `просрочка ${d} дн`,
      title: `«${late[0]!.title}» просрочена на ${d} ${plural(d, 'день', 'дня', 'дней')}`,
      text: 'Спроси, что мешает: возможно, стажер застрял.',
      cta: 'chat',
    })
  } else if (late.length > 1) {
    out.push({
      kind: 'late',
      chip: `просрочено: ${late.length}`,
      title: `Просрочено ${late.length} ${plural(late.length, 'задача', 'задачи', 'задач')}`,
      text: names(late),
      cta: 'chat',
    })
  }
  if (waiting.length > 0) {
    const d = Math.max(...waiting.map((w) => w.days))
    out.push({
      kind: 'waiting',
      chip: `согласование ${d} дн`,
      title:
        waiting.length === 1
          ? `«${waiting[0]!.title}» ждет согласования`
          : `Ждут согласования: ${waiting.length}`,
      text:
        waiting.length === 1
          ? `Срок прошел ${d} ${plural(d, 'день', 'дня', 'дней')} назад.`
          : names(waiting),
      cta: 'path',
    })
  }
  if (goals && !goals.agreed) {
    out.push({
      kind: 'goals',
      chip: 'цели не согласованы',
      title: 'Цели еще не согласованы',
      text: 'Их согласует руководитель команды на первой неделе. Без целей трекер прогресса пустой.',
      cta: null,
    })
  }
  return out
}
