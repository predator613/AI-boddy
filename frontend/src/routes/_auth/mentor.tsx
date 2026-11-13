import { createFileRoute } from '@tanstack/react-router'
import { internOnly } from '../../lib/guards'
import { MentorChatPage } from '../../pages/MentorChatPage'

interface MentorSearch {
  q?: string
}

export const Route = createFileRoute('/_auth/mentor')({
  // q – готовый текст вопроса наставнику (например, «свериться перед заявкой»).
  validateSearch: (search: Record<string, unknown>): MentorSearch =>
    typeof search.q === 'string' && search.q ? { q: search.q } : {},
  beforeLoad: internOnly,
  component: function MentorRoute() {
    const { q } = Route.useSearch()
    return <MentorChatPage initialDraft={q} />
  },
})
