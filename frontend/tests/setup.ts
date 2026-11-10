import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterAll, afterEach, beforeAll } from 'vitest'
import { server } from '../src/mocks/server'
import { useAuthStore } from '../src/store/authStore'
import { useSettingsStore } from '../src/store/settingsStore'
import { resetPathMock } from '../src/mocks/pathData'
import { resetPendingMocks } from '../src/mocks/pending'

beforeAll(() => server.listen({ onUnhandledFrame: 'error' }))

afterEach(() => {
  cleanup()
  server.resetHandlers()
  useAuthStore.setState({ token: null, user: null })
  useSettingsStore.getState().reset()
  resetPathMock()
  resetPendingMocks()
  localStorage.clear()
})

afterAll(() => server.close())

// Mantine использует matchMedia, которого нет в jsdom
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
})

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}
window.ResizeObserver = ResizeObserverMock
window.scrollTo = () => {}

Element.prototype.scrollIntoView = function scrollIntoView() {}

// В jsdom нет document.fonts, а Mantine Textarea autosize его слушает
if (!('fonts' in document)) {
  Object.defineProperty(document, 'fonts', {
    value: { addEventListener: () => {}, removeEventListener: () => {}, ready: Promise.resolve() },
  })
}
