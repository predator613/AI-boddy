import { redirect } from '@tanstack/react-router'
import { useAuthStore } from '../store/authStore'

// Проверки роли для beforeLoad маршрутов: недоступный раздел просто ведет на главную.
export function staffOnly() {
  if (useAuthStore.getState().user?.role === 'intern') throw redirect({ to: '/' })
}

export function internOnly() {
  if (useAuthStore.getState().user?.role !== 'intern') throw redirect({ to: '/' })
}

export function leadOnly() {
  if (useAuthStore.getState().user?.role !== 'lead') throw redirect({ to: '/' })
}
