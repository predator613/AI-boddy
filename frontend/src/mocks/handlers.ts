import { HttpResponse, delay, http } from 'msw'
import { API_URL } from '../api/client'
import { MOCK_DEPARTMENTS, MOCK_PASSWORD, mockTestUsers, mockToken, mockUsers } from './data'
import {
  agreeMockGoals,
  createMockGoal,
  deleteMockGoal,
  mockGoals,
  setMockGoalProgress,
  setMockStep,
  updateMockGoal,
  type MockGoalInput,
} from './goalsData'
import {
  PAVEL_ID,
  buildMockPath,
  buildMockPavelPath,
  mockInterns,
  setMockTaskStatus,
} from './pathData'
import { KB, assistantHandlers, pendingHandlers } from './pending'
import { consoleHandlers } from './consoleData'
import { calendarHandlers } from './mentorData'
import { leadHandlers } from './leadData'

const apiError = (status: number, code: string, message: string) =>
  HttpResponse.json({ error: { code, message } }, { status })

function userFrom(request: Request) {
  const token = request.headers.get('Authorization')?.replace(/^Bearer\s+/i, '')
  return mockUsers.find((u) => mockToken(u.login) === token) ?? null
}

// Имитация backend по контракту из брифа.
export const handlers = [
  http.post(`${API_URL}/api/auth/login`, async ({ request }) => {
    await delay(300)
    const body = (await request.json().catch(() => null)) as {
      login?: string
      password?: string
    } | null
    const login = body?.login?.trim()
    if (!login || !body?.password) {
      return apiError(400, 'validation_error', 'login and password are required')
    }
    const user = mockUsers.find((u) => u.login === login)
    if (!user || body.password !== MOCK_PASSWORD) {
      return apiError(401, 'invalid_credentials', 'invalid login or password')
    }
    return HttpResponse.json({ token: mockToken(user.login), user })
  }),

  http.get(`${API_URL}/api/auth/me`, ({ request }) => {
    const token = request.headers.get('Authorization')?.replace(/^Bearer\s+/i, '')
    const user = mockUsers.find((u) => mockToken(u.login) === token)
    if (!user) return apiError(401, 'unauthorized', 'invalid or expired token')
    return HttpResponse.json(user)
  }),

  http.get(`${API_URL}/api/auth/test-users`, () => HttpResponse.json(mockTestUsers)),

  // Наставник и руководитель – из связей пользователя, как в backend.
  http.get(`${API_URL}/api/me/contacts`, ({ request }) => {
    const user = userFrom(request)
    if (!user) return apiError(401, 'unauthorized', 'invalid or expired token')
    const ref = (u?: (typeof mockUsers)[number]) => (u ? { id: u.id, name: u.name } : null)
    return HttpResponse.json({
      team: user.teamName,
      mentor: ref(mockUsers.find((u) => u.id === user.mentorId)),
      lead: ref(mockUsers.find((u) => u.teamId === user.teamId && u.role === 'lead')),
      departments: MOCK_DEPARTMENTS,
    })
  }),
]

// «Мой путь» и стажеры наставника – по контракту docs/api-my-path.md.
export const pathHandlers = [
  http.get(`${API_URL}/api/path`, ({ request }) => {
    const user = userFrom(request)
    if (!user) return apiError(401, 'unauthorized', 'invalid or expired token')
    if (user.role !== 'intern') return apiError(403, 'forbidden', 'only interns have a path')
    return HttpResponse.json(buildMockPath(user))
  }),

  http.post(`${API_URL}/api/path/tasks/:taskId/complete`, ({ request, params }) => {
    const user = userFrom(request)
    if (!user) return apiError(401, 'unauthorized', 'invalid or expired token')
    if (!setMockTaskStatus(String(params.taskId), 'done'))
      return apiError(404, 'not_found', 'task not found')
    return HttpResponse.json(buildMockPath(user))
  }),

  http.get(`${API_URL}/api/interns`, ({ request }) => {
    const user = userFrom(request)
    if (!user) return apiError(401, 'unauthorized', 'invalid or expired token')
    if (user.role === 'intern') return apiError(403, 'forbidden', 'forbidden')
    return HttpResponse.json(mockInterns())
  }),

  http.get(`${API_URL}/api/interns/:internId/path`, ({ request, params }) => {
    const user = userFrom(request)
    if (!user || user.role === 'intern') return apiError(403, 'forbidden', 'forbidden')
    if (params.internId === PAVEL_ID) return HttpResponse.json(buildMockPavelPath())
    if (params.internId !== 'u-anna') return apiError(404, 'not_found', 'not found')
    return HttpResponse.json(buildMockPath())
  }),

  http.put(`${API_URL}/api/interns/:internId/tasks/:taskId`, async ({ request, params }) => {
    const user = userFrom(request)
    if (!user || user.role === 'intern') return apiError(403, 'forbidden', 'forbidden')
    const body = (await request.json().catch(() => null)) as {
      status?: 'todo' | 'in_progress' | 'waiting' | 'done'
    } | null
    const internId = String(params.internId)
    if (!body?.status || !setMockTaskStatus(String(params.taskId), body.status, internId))
      return apiError(404, 'not_found', 'not found')
    return HttpResponse.json(internId === PAVEL_ID ? buildMockPavelPath() : buildMockPath())
  }),
]

// Цели стажировки и справочник – по контракту docs/api-goals-kb.md.
export const goalsKbHandlers = [
  http.get(`${API_URL}/api/goals`, ({ request }) => {
    const user = userFrom(request)
    if (!user) return apiError(401, 'unauthorized', 'invalid or expired token')
    if (user.role !== 'intern') return apiError(404, 'not_found', 'not found')
    return HttpResponse.json(mockGoals(user.id, 'intern'))
  }),

  http.get(`${API_URL}/api/interns/:internId/goals`, ({ request, params }) => {
    const user = userFrom(request)
    if (!user || user.role === 'intern') return apiError(404, 'not_found', 'not found')
    return HttpResponse.json(mockGoals(String(params.internId), user.role))
  }),

  http.post(`${API_URL}/api/goals/:goalId/progress`, async ({ request, params }) => {
    const user = userFrom(request)
    if (!user) return apiError(401, 'unauthorized', 'invalid or expired token')
    const body = (await request.json().catch(() => null)) as { progressPct?: number } | null
    const pct = body?.progressPct
    if (typeof pct !== 'number' || pct < 0 || pct > 100)
      return apiError(400, 'validation_error', 'progressPct must be between 0 and 100')
    const goal = setMockGoalProgress(String(params.goalId), pct)
    return goal ? HttpResponse.json(goal) : apiError(404, 'not_found', 'not found')
  }),

  http.post(`${API_URL}/api/interns/:internId/goals`, async ({ request, params }) => {
    const user = userFrom(request)
    if (!user) return apiError(401, 'unauthorized', 'invalid or expired token')
    const body = ((await request.json().catch(() => null)) ?? {}) as MockGoalInput
    const g = createMockGoal(String(params.internId), body)
    if (g === 'limit') return apiError(400, 'validation_error', 'Не больше 6 целей')
    return HttpResponse.json(g, { status: 201 })
  }),

  http.patch(`${API_URL}/api/goals/:goalId`, async ({ request, params }) => {
    const body = ((await request.json().catch(() => null)) ?? {}) as MockGoalInput
    const g = updateMockGoal(String(params.goalId), body)
    return g ? HttpResponse.json(g) : apiError(404, 'not_found', 'not found')
  }),

  http.delete(`${API_URL}/api/goals/:goalId`, ({ params }) =>
    deleteMockGoal(String(params.goalId))
      ? new HttpResponse(null, { status: 204 })
      : apiError(404, 'not_found', 'not found'),
  ),

  http.post(`${API_URL}/api/interns/:internId/goals/agree`, ({ request, params }) => {
    const user = userFrom(request)
    if (user?.role !== 'lead') return apiError(403, 'forbidden', 'Согласует только руководитель')
    agreeMockGoals(String(params.internId))
    return HttpResponse.json(mockGoals(String(params.internId), 'lead'))
  }),

  http.post(`${API_URL}/api/goals/:goalId/steps/:stepId`, async ({ request, params }) => {
    const user = userFrom(request)
    if (!user || user.role === 'intern') return apiError(403, 'forbidden', 'forbidden')
    const body = (await request.json().catch(() => null)) as { done?: boolean } | null
    const g = setMockStep(String(params.goalId), String(params.stepId), body?.done !== false)
    if (g === 'task')
      return apiError(400, 'validation_error', 'Шаг-задача закрывается вместе с задачей')
    return g ? HttpResponse.json(g) : apiError(404, 'not_found', 'not found')
  }),

  http.get(`${API_URL}/api/kb`, () => HttpResponse.json(KB)),

  http.get(`${API_URL}/api/kb/articles/:articleId`, ({ params }) => {
    const a = KB.articles.find((x) => x.id === params.articleId)
    if (!a) return apiError(404, 'not_found', 'not found')
    return HttpResponse.json({
      ...a,
      body: `${a.summary}\n\nДемо-текст статьи: в режиме без backend показывается только аннотация.`,
      source: 'seed',
      url: null,
      authorName: null,
      useInAi: true,
      updatedAt: new Date().toISOString(),
    })
  }),

  http.get(`${API_URL}/api/kb/search`, ({ request }) => {
    const q = new URL(request.url).searchParams.get('q')?.trim().toLowerCase() ?? ''
    if (!q) return apiError(400, 'validation_error', 'q is required')
    const words = q.split(/\s+/).filter((w) => w.length > 2)
    const results = KB.articles
      .map((a) => {
        const text = `${a.title} ${a.summary}`.toLowerCase()
        const score = words.filter((w) => text.includes(w.slice(0, 5))).length
        return { a, score }
      })
      .filter((x) => x.score > 0)
      .sort((x, y) => y.score - x.score)
      .slice(0, 10)
      .map(({ a, score }) => ({
        articleId: a.id,
        sectionId: a.sectionId,
        title: a.title,
        summary: a.summary,
        snippet: a.summary,
        stageCode: a.stageCode,
        score,
      }))
    return HttpResponse.json({ query: q, results })
  }),
]

// Все обработчики: полный режим без backend и тесты.
export const allHandlers = [
  ...handlers,
  ...pathHandlers,
  ...goalsKbHandlers,
  ...consoleHandlers,
  ...assistantHandlers,
  ...calendarHandlers,
  ...leadHandlers,
  ...pendingHandlers,
]
