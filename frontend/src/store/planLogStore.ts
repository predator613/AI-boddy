import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface PlanLogEntry {
  id: string
  text: string
  at: string
}

interface PlanLogState {
  entries: PlanLogEntry[]
  add: (text: string) => void
  reset: () => void
}

// Журнал изменений плана адаптации. Пока backend не отдает историю шаблона,
// записи хранятся в браузере наставника (последние 20).
export const usePlanLogStore = create<PlanLogState>()(
  persist(
    (set) => ({
      entries: [],
      add: (text) =>
        set((s) => ({
          entries: [
            { id: `${Date.now()}-${s.entries.length}`, text, at: new Date().toISOString() },
            ...s.entries,
          ].slice(0, 20),
        })),
      reset: () => set({ entries: [] }),
    }),
    { name: 'ai-buddy-plan-log' },
  ),
)
