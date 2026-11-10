import { create } from 'zustand'

interface ToastState {
  text: string | null
  show: (text: string) => void
  hide: () => void
}

// Короткое уведомление внизу экрана («Анна получит уведомление»).
export const useToastStore = create<ToastState>((set) => ({
  text: null,
  show: (text) => set({ text }),
  hide: () => set({ text: null }),
}))

export const toast = (text: string) => useToastStore.getState().show(text)
