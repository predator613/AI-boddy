import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { MantineProvider } from '@mantine/core'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createRouter } from '@tanstack/react-router'
import '@fontsource/onest/400.css'
import '@fontsource/onest/500.css'
import '@fontsource/onest/600.css'
import '@fontsource/caveat/500.css'
import '@mantine/core/styles.css'
import './styles/global.css'
import { theme } from './theme'
import { routeTree } from './routeTree.gen'

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false } },
})

const router = createRouter({ routeTree, context: { queryClient } })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

// Моки MSW:
// - npm run dev:mock или VITE_USE_MOCKS=true – все запросы, backend не нужен;
// - по умолчанию – только эндпоинты, которых еще нет в backend (чат, заявки, справочник).
//   Выключить: VITE_PENDING_MOCKS=false.
async function enableMocks() {
  const env = import.meta.env
  const full = env.MODE === 'mock' || env.VITE_USE_MOCKS === 'true'
  if (!full && env.VITE_PENDING_MOCKS === 'false') return
  const mocks = await import('./mocks/browser')
  const worker = full ? mocks.fullWorker() : mocks.pendingWorker()
  await worker.start({ onUnhandledFrame: 'bypass', quiet: !full })
}

enableMocks()
  .catch(() => undefined) // без моков приложение все равно работает с backend
  .then(() => {
    createRoot(document.getElementById('root')!).render(
      <StrictMode>
        <MantineProvider theme={theme}>
          <QueryClientProvider client={queryClient}>
            <RouterProvider router={router} />
          </QueryClientProvider>
        </MantineProvider>
      </StrictMode>,
    )
  })
