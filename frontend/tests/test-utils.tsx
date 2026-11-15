import type { ReactElement } from 'react'
import { render } from '@testing-library/react'
import { MantineProvider } from '@mantine/core'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRouter } from '@tanstack/react-router'
import { theme } from '../src/theme'
import { routeTree } from '../src/routeTree.gen'

export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
}

// Рендер компонента с Mantine и TanStack Query
export function renderWithProviders(ui: ReactElement) {
  const queryClient = createTestQueryClient()
  return render(
    <MantineProvider theme={theme}>
      <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
    </MantineProvider>,
  )
}

// Рендер всего приложения с роутером, начиная с адреса path
export function renderApp(path: string) {
  const queryClient = createTestQueryClient()
  const router = createRouter({
    routeTree,
    context: { queryClient },
    history: createMemoryHistory({ initialEntries: [path] }),
  })
  render(
    <MantineProvider theme={theme}>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </MantineProvider>,
  )
  return router
}

import { mockToken, mockUsers } from '../src/mocks/data'
import { useAuthStore } from '../src/store/authStore'

/** Войти тестовым пользователем без формы: intern_maria, mentor_igor, lead_artemiy, intern_dima. */
export function loginAs(login: string) {
  const user = mockUsers.find((u) => u.login === login)!
  useAuthStore.setState({ token: mockToken(login), user })
  return user
}
