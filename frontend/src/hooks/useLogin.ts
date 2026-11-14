import { useMutation } from '@tanstack/react-query'
import { login } from '../api/auth'
import { ApiError } from '../api/client'
import { useAuthStore } from '../store/authStore'

export function useLogin(onSuccess?: () => void) {
  const setAuth = useAuthStore((s) => s.setAuth)
  return useMutation({
    mutationFn: login,
    onSuccess: ({ token, user }) => {
      setAuth(token, user)
      onSuccess?.()
    },
  })
}

export function loginErrorText(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.code === 'invalid_credentials') return 'Неверный логин или пароль'
    if (error.code === 'too_many_attempts')
      return 'Слишком много неудачных попыток. Подождите 5 минут и попробуйте снова'
    if (error.code === 'network_error') return error.message
  }
  return 'Не удалось войти. Попробуйте еще раз'
}
