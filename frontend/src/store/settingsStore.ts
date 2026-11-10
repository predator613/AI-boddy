import { create } from 'zustand'
import { persist } from 'zustand/middleware'

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
    { name: 'ai-buddy-settings' },
  ),
)
