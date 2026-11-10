// Цели стажировки для режима без backend и для тестов. Повторяют seed backend (011_seed_demo.sql).
import type { TGoal, TGoalsView } from '../schemas/goals'

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
})

const initial = () => [
  goal(1, 'Пройти путь задачи до релиза', 'первый merge в main до 4 недели', 4, 0, 0),
  goal(2, 'Разобраться в сервисе авторизации', 'провести демо команде на 8 неделе', 8, 10, 0),
  goal(
    3,
    'Самостоятельно закрывать задачи уровня junior',
    '3 задачи без помощи к 12 неделе',
    12,
    0,
    null,
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
