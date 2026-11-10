import { createFileRoute } from '@tanstack/react-router'
import { ChatPage } from '../../pages/ChatPage'

interface ChatSearch {
  q?: string
  task?: string
}

export const Route = createFileRoute('/_auth/chat')({
  validateSearch: (s: Record<string, unknown>): ChatSearch => ({
    ...(typeof s.q === 'string' && s.q ? { q: s.q } : {}),
    ...(typeof s.task === 'string' && s.task ? { task: s.task } : {}),
  }),
  component: function ChatRoute() {
    const { q, task } = Route.useSearch()
    return <ChatPage initialQuestion={q} taskId={task} />
  },
})
