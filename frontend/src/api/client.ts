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

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  signal?: AbortSignal
}

// Единственное место, через которое фронт ходит в backend.
export async function apiRequest<T>(
  path: string,
  schema: z.ZodType<T>,
  { method = 'GET', body, signal }: RequestOptions = {},
): Promise<T> {
  const token = useAuthStore.getState().token
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (token) headers.Authorization = `Bearer ${token}`

  let response: Response
  try {
    response = await fetch(`${API_URL}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
    })
  } catch (error) {
    if (signal?.aborted) throw error // запрос отменили сами (например, ушли со страницы)
    throw new ApiError(0, 'network_error', 'Сервер недоступен. Проверьте подключение')
  }

  const data: unknown = await response.json().catch(() => null)

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

  const parsed = schema.safeParse(data)
  if (!parsed.success) {
    throw new ApiError(
      response.status,
      'invalid_response',
      'Сервер вернул данные в неожиданном формате',
    )
  }
  return parsed.data
}
