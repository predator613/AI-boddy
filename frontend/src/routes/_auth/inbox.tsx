import { createFileRoute } from '@tanstack/react-router'
import { staffOnly } from '../../lib/guards'
import { InboxPage } from '../../pages/InboxPage'

export const Route = createFileRoute('/_auth/inbox')({
  beforeLoad: staffOnly,
  validateSearch: (s: Record<string, unknown>): { intern?: string } =>
    typeof s.intern === 'string' && s.intern ? { intern: s.intern } : {},
  component: function InboxRoute() {
    const { intern } = Route.useSearch()
    const navigate = Route.useNavigate()
    return (
      <InboxPage
        selectedId={intern}
        onSelect={(id) => navigate({ search: { intern: id }, replace: true })}
      />
    )
  },
})
