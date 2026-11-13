import '@testing-library/jest-dom/vitest'
import { cleanup, configure } from '@testing-library/react'
import { afterAll, afterEach, beforeAll } from 'vitest'
import { server } from '../src/mocks/server'
import { useAuthStore } from '../src/store/authStore'
import { useSettingsStore } from '../src/store/settingsStore'
import { resetPathMock } from '../src/mocks/pathData'
import { resetPendingMocks } from '../src/mocks/pending'
import { resetCalendarMock, resetMentorMocks } from '../src/mocks/mentorData'
import { resetConsoleMock } from '../src/mocks/consoleData'
import { useToastStore } from '../src/store/toastStore'
import { usePlanLogStore } from '../src/store/planLogStore'
import { resetLeadMocks, resetSupportMocks } from '../src/mocks/leadData'
import { resetGoalsMock } from '../src/mocks/goalsData'

beforeAll(() => server.listen({ onUnhandledFrame: 'error' }))

afterEach(() => {
  cleanup()
  server.resetHandlers()
  useAuthStore.setState({ token: null, user: null })
  useSettingsStore.getState().reset()
  resetPathMock()
  resetPendingMocks()
  resetSupportMocks()
  resetMentorMocks()
  resetCalendarMock()
  resetConsoleMock()
  resetLeadMocks()
  usePlanLogStore.getState().reset()
  resetGoalsMock()
  useToastStore.setState({ text: null })
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

// ожидание появления элементов: с запасом для медленных компьютеров
configure({ asyncUtilTimeout: 5000 })
