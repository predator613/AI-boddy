import { HttpResponse, delay, http } from 'msw'
import { API_URL } from '../api/client'
import { MOCK_PASSWORD, mockTestUsers, mockToken, mockUsers } from './data'
import { buildMockPath, mockInterns, setMockTaskStatus } from './pathData'
import { pendingHandlers } from './pending'

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
    if (params.internId !== 'u-anna') return apiError(404, 'not_found', 'not found')
    return HttpResponse.json(buildMockPath())
  }),

  http.put(`${API_URL}/api/interns/:internId/tasks/:taskId`, async ({ request, params }) => {
    const user = userFrom(request)
    if (!user || user.role === 'intern') return apiError(403, 'forbidden', 'forbidden')
    const body = (await request.json().catch(() => null)) as {
      status?: 'todo' | 'in_progress' | 'waiting' | 'done'
    } | null
    if (!body?.status || !setMockTaskStatus(String(params.taskId), body.status))
      return apiError(404, 'not_found', 'not found')
    return HttpResponse.json(buildMockPath())
  }),
]

// Все обработчики: полный режим без backend и тесты.
export const allHandlers = [...handlers, ...pathHandlers, ...pendingHandlers]
