// API целей стажировки. Все запросы идут через общий client.ts.
import { apiFetch, apiRequest } from './client'
import {
  Goal,
  GoalInput,
  GoalPatch,
  GoalsView,
  ProgressInput,
  type TGoalInput,
  type TGoalPatch,
  type TProgressInput,
} from '../schemas/goals'

const id = (s: string) => encodeURIComponent(s)

/** GET /api/goals – цели текущего стажера. */
export function getMyGoals() {
  return apiRequest('/api/goals', GoalsView)
}

/** GET /api/interns/{id}/goals – цели стажера для наставника и руководителя. */
export function getInternGoals(internId: string) {
  return apiRequest(`/api/interns/${id(internId)}/goals`, GoalsView)
}

/** POST /api/goals/{id}/progress – отметить прогресс (стажер, его наставник, руководитель). */
export function setGoalProgress(goalId: string, input: TProgressInput) {
  return apiRequest(`/api/goals/${id(goalId)}/progress`, Goal, {
    method: 'POST',
    body: ProgressInput.parse(input),
  })
}

/** POST /api/interns/{id}/goals – добавить цель. Стажер может, пока цели не согласованы. */
export function createGoal(internId: string, input: TGoalInput) {
  return apiRequest(`/api/interns/${id(internId)}/goals`, Goal, {
    method: 'POST',
    body: GoalInput.parse(input),
  })
}

/** PATCH /api/goals/{id} – изменить цель (только переданные поля). */
export function updateGoal(goalId: string, input: TGoalPatch) {
  return apiRequest(`/api/goals/${id(goalId)}`, Goal, {
    method: 'PATCH',
    body: GoalPatch.parse(input),
  })
}

/** DELETE /api/goals/{id}. */
export async function deleteGoal(goalId: string) {
  await apiFetch(`/api/goals/${id(goalId)}`, { method: 'DELETE' })
}

/** POST /api/interns/{id}/goals/agree – руководитель согласовал цели. */
export function agreeGoals(internId: string) {
  return apiRequest(`/api/interns/${id(internId)}/goals/agree`, GoalsView, { method: 'POST' })
}
