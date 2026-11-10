import { HttpResponse, http } from 'msw'
import { z } from 'zod'
import { API_URL, ApiError, apiRequest } from '../src/api/client'
import { getMe, getTestUsers, login } from '../src/api/auth'
import { server } from '../src/mocks/server'
import { mockToken, mockUsers } from '../src/mocks/data'
import { useAuthStore } from '../src/store/authStore'

const Ok = z.object({ ok: z.boolean() })

describe('api/client', () => {
  it('подставляет токен в заголовок Authorization', async () => {
    let header: string | null = null
    server.use(
      http.get(`${API_URL}/api/ping`, ({ request }) => {
        header = request.headers.get('Authorization')
        return HttpResponse.json({ ok: true })
      }),
    )
    useAuthStore.setState({ token: 'abc' })
    await expect(apiRequest('/api/ping', Ok)).resolves.toEqual({ ok: true })
    expect(header).toBe('Bearer abc')
  })

  it('без токена заголовок Authorization не отправляется', async () => {
    let header: string | null = 'x'
    server.use(
      http.get(`${API_URL}/api/ping`, ({ request }) => {
        header = request.headers.get('Authorization')
        return HttpResponse.json({ ok: true })
      }),
    )
    await apiRequest('/api/ping', Ok)
    expect(header).toBeNull()
  })

  it('при 401 вызывает logout и бросает ApiError с кодом из ответа', async () => {
    useAuthStore.setState({ token: 'expired', user: mockUsers[0] })
    const error = await getMe().catch((e: unknown) => e)
    expect(error).toBeInstanceOf(ApiError)
    expect((error as ApiError).code).toBe('unauthorized')
    expect(useAuthStore.getState().token).toBeNull()
  })

  it('при 401 на вход (без токена) не трогает store', async () => {
    const error = await login({ login: 'intern_anna', password: 'wrong' }).catch((e: unknown) => e)
    expect((error as ApiError).status).toBe(401)
    expect((error as ApiError).code).toBe('invalid_credentials')
  })

  it('ошибка без JSON-тела превращается в http_<код>', async () => {
    server.use(http.get(`${API_URL}/api/ping`, () => new HttpResponse('oops', { status: 500 })))
    const error = (await apiRequest('/api/ping', Ok).catch((e: unknown) => e)) as ApiError
    expect(error.status).toBe(500)
    expect(error.code).toBe('http_500')
  })

  it('ответ неожиданного формата – invalid_response', async () => {
    server.use(http.get(`${API_URL}/api/ping`, () => HttpResponse.json({ ok: 'yes' })))
    const error = (await apiRequest('/api/ping', Ok).catch((e: unknown) => e)) as ApiError
    expect(error.code).toBe('invalid_response')
  })

  it('сервер недоступен – network_error', async () => {
    server.use(http.get(`${API_URL}/api/ping`, () => HttpResponse.error()))
    const error = (await apiRequest('/api/ping', Ok).catch((e: unknown) => e)) as ApiError
    expect(error.code).toBe('network_error')
    expect(error.status).toBe(0)
  })

  it('отмена запроса пробрасывается как есть', async () => {
    server.use(http.get(`${API_URL}/api/ping`, () => new Promise(() => {})))
    const controller = new AbortController()
    const promise = apiRequest('/api/ping', Ok, { signal: controller.signal })
    controller.abort()
    await expect(promise).rejects.toMatchObject({ name: 'AbortError' })
  })
})

describe('api/auth', () => {
  it('login возвращает токен и пользователя', async () => {
    const res = await login({ login: 'intern_anna', password: 'demo123' })
    expect(res.token).toBe(mockToken('intern_anna'))
    expect(res.user.name).toBe('Анна')
  })

  it('getMe возвращает пользователя по токену', async () => {
    useAuthStore.setState({ token: mockToken('lead_oleg') })
    await expect(getMe()).resolves.toMatchObject({ login: 'lead_oleg', role: 'lead' })
  })

  it('getTestUsers возвращает список', async () => {
    const users = await getTestUsers()
    expect(users).toHaveLength(4)
  })

  it('getTestUsers при 404 (не демо-режим) возвращает пустой список', async () => {
    server.use(
      http.get(`${API_URL}/api/auth/test-users`, () =>
        HttpResponse.json({ error: { code: 'not_found', message: 'not found' } }, { status: 404 }),
      ),
    )
    await expect(getTestUsers()).resolves.toEqual([])
  })

  it('getTestUsers пробрасывает другие ошибки', async () => {
    server.use(
      http.get(`${API_URL}/api/auth/test-users`, () => new HttpResponse(null, { status: 500 })),
    )
    await expect(getTestUsers()).rejects.toBeInstanceOf(ApiError)
  })
})
