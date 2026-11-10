import type { z } from 'zod'
import { ApiErrorSchema } from '../schemas/auth'
import { useAuthStore } from '../store/authStore'

export const API_URL = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/+$/, '')

export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

// Единственное место, через которое фронт ходит в backend.
// Подставляет адрес и токен, разбирает ошибки, при 401 выходит из аккаунта.
// Возвращает разобранный JSON или undefined для пустого ответа (204).
export async function apiFetch(path: string, init: RequestInit = {}): Promise<unknown> {
  const token = useAuthStore.getState().token
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (init.body !== undefined && init.body !== null && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  if (token) headers.set('Authorization', `Bearer ${token}`)

  let response: Response
  try {
    response = await fetch(`${API_URL}${path}`, { ...init, headers })
  } catch (error) {
    if (init.signal?.aborted) throw error // запрос отменили сами (например, ушли со страницы)
    throw new ApiError(0, 'network_error', 'Сервер недоступен. Проверьте подключение')
  }

  const text = await response.text().catch(() => '')
  let data: unknown = undefined
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = null
    }
  }

  if (!response.ok) {
    // токен истек или недействителен – выходим, защищенный layout сам отправит на /login
    if (response.status === 401 && token) useAuthStore.getState().logout()

    const parsed = ApiErrorSchema.safeParse(data)
    if (parsed.success) {
      throw new ApiError(response.status, parsed.data.error.code, parsed.data.error.message)
    }
    throw new ApiError(
      response.status,
      `http_${response.status}`,
      `Ошибка сервера (${response.status})`,
    )
  }

  return data
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  signal?: AbortSignal
}

// То же, что apiFetch, но сразу проверяет ответ Zod-схемой.
export async function apiRequest<T>(
  path: string,
  schema: z.ZodType<T>,
  { method = 'GET', body, signal }: RequestOptions = {},
): Promise<T> {
  const data = await apiFetch(path, {
    method,
    body: body === undefined ? undefined : JSON.stringify(body),
    signal,
  })
  const parsed = schema.safeParse(data)
  if (!parsed.success) {
    throw new ApiError(200, 'invalid_response', 'Сервер вернул данные в неожиданном формате')
  }
  return parsed.data
}
