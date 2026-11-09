import { screen } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRouter } from '@tanstack/react-router'
import { routeTree } from '../src/routeTree.gen'
import { renderWithMantine } from './test-utils'

function renderAt(path: string) {
  const router = createRouter({
    routeTree,
    context: { queryClient: new QueryClient() },
    history: createMemoryHistory({ initialEntries: [path] }),
  })
  renderWithMantine(<RouterProvider router={router} />)
  return router
}

describe('роутер', () => {
  it('на / показывает главную', async () => {
    renderAt('/')
    expect(await screen.findByText('AI-buddy')).toBeInTheDocument()
  })
})
