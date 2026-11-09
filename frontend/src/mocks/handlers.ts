import { HttpResponse, delay, http } from 'msw'
import { API_URL } from '../api/client'
import { MOCK_PASSWORD, mockTestUsers, mockToken, mockUsers } from './data'

const apiError = (status: number, code: string, message: string) =>
  HttpResponse.json({ error: { code, message } }, { status })

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
