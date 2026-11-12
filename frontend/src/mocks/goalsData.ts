// Цели стажировки для режима без backend и для тестов. Повторяют seed backend (011_seed_demo.sql).
import type { TGoal, TGoalStep, TGoalsView } from '../schemas/goals'

const OLEG = { id: '33333333-3333-4333-8333-333333333333', name: 'Олег' }
const OKRS = [
  { id: 'okr-1', period: 'Q4 2026', title: 'Ускорить выпуск изменений в сервисе авторизации' },
  { id: 'okr-2', period: 'Q4 2026', title: 'Поднять покрытие тестами ключевых модулей до 70%' },
]

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

let annaGoals = initial()

export function resetGoalsMock() {
  annaGoals = initial()
}

/** Цели стажера: у Анны три согласованные цели, у остальных целей пока нет. */
export function mockGoals(internId: string, viewerRole: 'intern' | 'mentor' | 'lead'): TGoalsView {
  const isAnna = internId === 'u-anna' || internId === '11111111-1111-4111-8111-111111111111'
  return {
    internId,
    agreed: isAnna ? { by: OLEG, at: new Date(Date.now() - 6 * 86_400_000).toISOString() } : null,
    fromOkr: isAnna,
    goals: isAnna ? annaGoals : [],
    okrs: OKRS,
    canEdit: viewerRole !== 'intern' || !isAnna,
    canAgree: viewerRole === 'lead' && isAnna,
  }
}

export function setMockGoalProgress(goalId: string, pct: number): TGoal | null {
  const g = annaGoals.find((x) => x.id === goalId)
  if (!g) return null
  g.progressPct = pct
  g.updatedAt = new Date().toISOString()
  return g
}
