import { ApiError } from '../api/client'

// Понятный текст ошибки запроса для пользователя.
export function errorText(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.code === 'network_error') return error.message
    if (error.status === 403) return 'Этот раздел недоступен для твоей роли.'
    if (error.status === 404) return 'Ничего не нашлось или у тебя нет доступа.'
  }
  return 'Не удалось загрузить данные. Попробуй еще раз.'
}
