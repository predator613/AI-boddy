import {
  LoginResponseSchema,
  TestUsersSchema,
  UserSchema,
  type LoginRequest,
  type TestUser,
} from '../schemas/auth'
import { ApiError, apiRequest } from './client'

export function login(data: LoginRequest) {
  return apiRequest('/api/auth/login', LoginResponseSchema, { method: 'POST', body: data })
}

export function getMe() {
  return apiRequest('/api/auth/me', UserSchema)
}

// В обычном (не демо) режиме backend отвечает 404 – тогда просто нет тестовых пользователей.
export async function getTestUsers(): Promise<TestUser[]> {
  try {
    return await apiRequest('/api/auth/test-users', TestUsersSchema)
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return []
    throw error
  }
}
