import { ApiError } from '../api/client'

// Понятный текст ошибки запроса для пользователя.
export function errorText(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.code === 'network_error') return error.message
    if (error.status === 403) return 'Этот раздел недоступен для твоей роли.'
    // конфликт (например, запись пересекается со встречей): сервер объясняет причину сам
    if (error.status === 409 && error.message) return error.message
    if (error.status === 404) return 'Ничего не нашлось или у тебя нет доступа.'
    if (error.status === 503)
      return 'Сервис заявок сейчас не отвечает. Попробуй еще раз через минуту.'
  }
  return 'Не удалось загрузить данные. Попробуй еще раз.'
}
