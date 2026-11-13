import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { useAuthStore } from './authStore'

export type AnswerStyle = 'short' | 'normal' | 'detailed'
export type NotificationKey = 'tasks' | 'requests' | 'meetings' | 'tips'

interface SettingsState {
  notifications: Record<NotificationKey, boolean>
  answerStyle: AnswerStyle
  toggle: (key: NotificationKey) => void
  setAnswerStyle: (style: AnswerStyle) => void
  reset: () => void
}

const DEFAULTS = {
  notifications: { tasks: true, requests: true, meetings: true, tips: false },
  answerStyle: 'normal' as AnswerStyle,
}

// У каждого аккаунта свои настройки: ключ в браузере зависит от id пользователя.
const keyFor = (name: string) => `${name}:${useAuthStore.getState().user?.id ?? 'guest'}`

// Личные настройки помощника. Хранятся в браузере, на сервер не уходят.
export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      ...DEFAULTS,
      toggle: (key) =>
        set((s) => ({ notifications: { ...s.notifications, [key]: !s.notifications[key] } })),
      setAnswerStyle: (answerStyle) => set({ answerStyle }),
      reset: () => set(DEFAULTS),
    }),
    {
      name: 'ai-buddy-settings',
      storage: createJSONStorage(() => ({
        getItem: (name) => localStorage.getItem(keyFor(name)),
        setItem: (name, value) => localStorage.setItem(keyFor(name), value),
        removeItem: (name) => localStorage.removeItem(keyFor(name)),
      })),
      // Сохраненное кладем поверх значений по умолчанию, а не поверх настроек прошлого аккаунта.
      merge: (saved, current) => ({
        ...current,
        ...DEFAULTS,
        ...(saved as Partial<SettingsState> | undefined),
      }),
    },
  ),
)

// Сменился пользователь – подгружаем его настройки.
useAuthStore.subscribe((s, prev) => {
  if (s.user?.id !== prev.user?.id) void useSettingsStore.persist.rehydrate()
})
