import { createFileRoute, redirect } from '@tanstack/react-router'
import { RequestsPage } from '../../pages/RequestsPage'
import { useAuthStore } from '../../store/authStore'

export const Route = createFileRoute('/_auth/requests')({
  // Заявки есть только у стажера: остальным раздел не показываем.
  beforeLoad: () => {
    if (useAuthStore.getState().user?.role !== 'intern') throw redirect({ to: '/' })
  },
  validateSearch: (s: Record<string, unknown>): { id?: string } =>
    typeof s.id === 'string' && s.id ? { id: s.id } : {},
  component: function RequestsRoute() {
    const { id } = Route.useSearch()
    const navigate = Route.useNavigate()
    return (
      <RequestsPage
        selectedId={id}
        onSelect={(next) => navigate({ search: { id: next }, replace: true })}
      />
    )
  },
})
