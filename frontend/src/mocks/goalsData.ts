// Цели стажировки для режима без backend и для тестов. Повторяют seed backend (011_seed_demo.sql).
import type { TGoal, TGoalStep, TGoalsView } from '../schemas/goals'

const OLEG = { id: '33333333-3333-4333-8333-333333333333', name: 'Олег' }
const OKRS = [
  { id: 'okr-1', period: 'Q4 2026', title: 'Ускорить выпуск изменений в сервисе авторизации' },
  { id: 'okr-2', period: 'Q4 2026', title: 'Поднять покрытие тестами ключевых модулей до 70%' },
]

// Компетенции из матрицы роли, к которым привязаны цели демо-данных.
const COMPETENCY: Record<string, string> = {
  'Пройти путь задачи до релиза': 'Код и ревью · L1',
  'Разобраться в сервисе авторизации': 'Обработка данных · L1',
  'Самостоятельно закрывать задачи уровня junior': 'Продуктовый подход',
  'Разобраться в процессах команды': 'Обмен знаниями',
}

const goal = (
  n: number,
  title: string,
  criterion: string,
  dueWeek: number,
  pct: number,
  okr: number | null,
  steps: TGoalStep[] = [],
): TGoal => ({
  id: `g-${n}`,
  position: n,
  title,
  criterion,
  dueWeek,
  progressPct: pct,
  okr: okr === null ? null : OKRS[okr],
  lastUpdate: null,
  updatedAt: new Date().toISOString(),
  createdBy: OLEG,
  steps,
  competency: COMPETENCY[title] ?? null,
})

const step = (
  id: string,
  title: string,
  source: TGoalStep['source'],
  stageCode: string | null,
  status: TGoalStep['status'] = 'todo',
): TGoalStep => ({ id, title, source, stageCode, taskId: null, status })

const initial = () => [
  goal(1, 'Пройти путь задачи до релиза', 'первый merge в main до 4 недели', 4, 0, 0, [
    step('s-11', 'Доступ к GitLab', 'task', '02', 'in_progress'),
    step('s-12', 'Окружение', 'task', '03'),
    step('s-13', 'Первая задача', 'task', '04'),
    step('s-14', 'Merge в main', 'mentor', null),
  ]),
  goal(2, 'Разобраться в сервисе авторизации', 'провести демо команде на 8 неделе', 8, 25, 0, [
    step('s-21', 'Памятка и документация', 'task', '01', 'done'),
    step('s-22', 'Разбор схемы на 1:1', 'mentor', null),
    step('s-23', 'Тесты к модулю', 'task', '04'),
    step('s-24', 'Демо команде', 'lead', null),
  ]),
  goal(
    3,
    'Самостоятельно закрывать задачи уровня junior',
    '3 задачи без помощи к 12 неделе',
    12,
    0,
    null,
    [
      step('s-31', 'Задача 1', 'mentor', null),
      step('s-32', 'Задача 2', 'mentor', null),
      step('s-33', 'Задача 3', 'mentor', null),
    ],
  ),
]

// У Павла цели – черновик со встречи с Олегом, еще не согласованы.
const pavelDraft = () => [
  goal(4, 'Разобраться в процессах команды', 'провести стендап сам на 4 неделе', 4, 0, null, [
    step('s-41', 'Памятка стажера', 'task', '01'),
    step('s-42', 'Провести стендап', 'mentor', null),
  ]),
  goal(5, 'Пройти путь задачи до релиза', 'первый merge в main до 6 недели', 6, 0, 0, [
    step('s-51', 'Доступ к GitLab', 'task', '02'),
    step('s-52', 'Merge в main', 'mentor', null),
  ]),
]

const ANNA = ['u-anna', '11111111-1111-4111-8111-111111111111']
const key = (internId: string) => (ANNA.includes(internId) ? 'u-anna' : internId)

interface Store {
  goals: TGoal[]
  agreed: TGoalsView['agreed']
}
const initialStore = (): Record<string, Store> => ({
  'u-anna': {
    goals: initial(),
    agreed: { by: OLEG, at: new Date(Date.now() - 6 * 86_400_000).toISOString() },
  },
  'u-pavel': { goals: pavelDraft(), agreed: null },
})
let store = initialStore()
let seq = 100

export function resetGoalsMock() {
  store = initialStore()
}

const pctOf = (g: TGoal) =>
  g.steps.length
    ? Math.round((g.steps.filter((x) => x.status === 'done').length / g.steps.length) * 100)
    : g.progressPct

function entry(internId: string): Store {
  const k = key(internId)
  store[k] ??= { goals: [], agreed: null }
  return store[k]
}

/** Цели стажера. Стажер правит свои цели, пока они не согласованы; согласует только руководитель. */
export function mockGoals(internId: string, viewerRole: 'intern' | 'mentor' | 'lead'): TGoalsView {
  const e = entry(internId)
  return {
    internId,
    agreed: e.agreed,
    fromOkr: e.goals.some((g) => g.okr !== null),
    goals: e.goals.map((g) => ({ ...g, progressPct: pctOf(g) })),
    okrs: OKRS,
    canEdit: viewerRole !== 'intern' || !e.agreed,
    canAgree: viewerRole === 'lead' && !e.agreed && e.goals.length > 0,
  }
}

function findGoal(goalId: string): { e: Store; g: TGoal } | null {
  for (const e of Object.values(store)) {
    const g = e.goals.find((x) => x.id === goalId)
    if (g) return { e, g }
  }
  return null
}

export function setMockGoalProgress(goalId: string, pct: number): TGoal | null {
  const f = findGoal(goalId)
  if (!f) return null
  f.g.progressPct = pct
  f.g.updatedAt = new Date().toISOString()
  return f.g
}

export interface MockGoalInput {
  title?: string
  criterion?: string
  dueWeek?: number
  okrId?: string
  competency?: string
  steps?: { title: string; source: TGoalStep['source'] }[]
}

const toSteps = (steps: MockGoalInput['steps'] = []): TGoalStep[] =>
  steps.map((x) => ({
    id: `s-${++seq}`,
    title: x.title,
    source: x.source,
    stageCode: null,
    taskId: null,
    status: 'todo',
  }))

export function createMockGoal(internId: string, input: MockGoalInput): TGoal | 'limit' {
  const e = entry(internId)
  if (e.goals.length >= 6) return 'limit'
  const n = ++seq
  const g: TGoal = {
    ...goal(
      n,
      input.title ?? '',
      input.criterion ?? '',
      input.dueWeek ?? 0,
      0,
      null,
      toSteps(input.steps),
    ),
    dueWeek: input.dueWeek || null,
    okr: OKRS.find((o) => o.id === input.okrId) ?? null,
    competency: input.competency || null,
  }
  e.goals.push(g)
  return g
}

export function updateMockGoal(goalId: string, input: MockGoalInput): TGoal | null {
  const f = findGoal(goalId)
  if (!f) return null
  const g = f.g
  if (input.title !== undefined) g.title = input.title
  if (input.criterion !== undefined) g.criterion = input.criterion
  if (input.dueWeek !== undefined) g.dueWeek = input.dueWeek || null
  if (input.okrId !== undefined) g.okr = OKRS.find((o) => o.id === input.okrId) ?? null
  if (input.competency !== undefined) g.competency = input.competency || null
  if (input.steps) g.steps = toSteps(input.steps)
  g.updatedAt = new Date().toISOString()
  return g
}

export function deleteMockGoal(goalId: string): boolean {
  const f = findGoal(goalId)
  if (!f) return false
  f.e.goals = f.e.goals.filter((x) => x.id !== goalId)
  return true
}

export function agreeMockGoals(internId: string) {
  entry(internId).agreed = { by: OLEG, at: new Date().toISOString() }
}

/** Наставник или руководитель подтверждает шаг. Шаг-задачу так отметить нельзя. */
export function setMockStep(goalId: string, stepId: string, done: boolean): TGoal | 'task' | null {
  const f = findGoal(goalId)
  const st = f?.g.steps.find((x) => x.id === stepId)
  if (!f || !st) return null
  if (st.source === 'task') return 'task'
  st.status = done ? 'done' : 'todo'
  return { ...f.g, progressPct: pctOf(f.g) }
}
