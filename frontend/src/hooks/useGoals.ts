// Хуки TanStack Query для целей стажировки.
// Прогресс по клику обновляется сразу (оптимистично), при ошибке откатывается.
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/goals'
import type { TGoal, TGoalInput, TGoalPatch, TGoalsView } from '../schemas/goals'

export const goalKeys = {
  my: ['goals', 'me'] as const,
  intern: (id: string) => ['goals', 'intern', id] as const,
}

/** internId не задан – цели текущего стажера, иначе – стажера из кабинета наставника. */
const keyFor = (internId?: string) => (internId ? goalKeys.intern(internId) : goalKeys.my)

export function useMyGoals() {
  return useQuery({ queryKey: goalKeys.my, queryFn: api.getMyGoals })
}

export function useInternGoals(internId: string) {
  return useQuery({
    queryKey: goalKeys.intern(internId),
    queryFn: () => api.getInternGoals(internId),
    enabled: Boolean(internId),
  })
}

const replaceGoal = (view: TGoalsView | undefined, goal: TGoal) =>
  view && { ...view, goals: view.goals.map((g) => (g.id === goal.id ? goal : g)) }

export function useGoalProgress(internId?: string) {
  const qc = useQueryClient()
  const key = keyFor(internId)
  return useMutation({
    mutationFn: ({
      goalId,
      progressPct,
      note,
    }: {
      goalId: string
      progressPct: number
      note?: string
    }) => api.setGoalProgress(goalId, { progressPct, note }),
    onMutate: async ({ goalId, progressPct }) => {
      await qc.cancelQueries({ queryKey: key })
      const prev = qc.getQueryData<TGoalsView>(key)
      qc.setQueryData<TGoalsView>(
        key,
        (v) =>
          v && { ...v, goals: v.goals.map((g) => (g.id === goalId ? { ...g, progressPct } : g)) },
      )
      return { prev }
    },
    onError: (_err, _vars, ctx) => qc.setQueryData(key, ctx?.prev),
    onSuccess: (goal) => qc.setQueryData<TGoalsView>(key, (v) => replaceGoal(v, goal)),
  })
}

/** Добавить, изменить, удалить цель – после ответа перечитываем список. */
export function useGoalEditor(internId: string, own = false) {
  const qc = useQueryClient()
  const key = own ? goalKeys.my : goalKeys.intern(internId)
  const refresh = () => qc.invalidateQueries({ queryKey: key })
  return {
    create: useMutation({
      mutationFn: (input: TGoalInput) => api.createGoal(internId, input),
      onSuccess: refresh,
    }),
    update: useMutation({
      mutationFn: ({ goalId, ...input }: TGoalPatch & { goalId: string }) =>
        api.updateGoal(goalId, input),
      onSuccess: refresh,
    }),
    remove: useMutation({
      mutationFn: (goalId: string) => api.deleteGoal(goalId),
      onSuccess: refresh,
    }),
  }
}

export function useAgreeGoals(internId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.agreeGoals(internId),
    onSuccess: (view) => qc.setQueryData(goalKeys.intern(internId), view),
  })
}

/** Подтвердить шаг цели (наставник – свои шаги, руководитель – свои). */
export function useGoalStep(internId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ goalId, stepId, done }: { goalId: string; stepId: string; done: boolean }) =>
      api.setGoalStep(goalId, stepId, done),
    onSuccess: (goal) =>
      qc.setQueryData<TGoalsView>(goalKeys.intern(internId), (v) => replaceGoal(v, goal)),
  })
}
