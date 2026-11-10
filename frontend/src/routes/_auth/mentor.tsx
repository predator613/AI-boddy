import { createFileRoute } from '@tanstack/react-router'
import { internOnly } from '../../lib/guards'
import { MentorChatPage } from '../../pages/MentorChatPage'

export const Route = createFileRoute('/_auth/mentor')({
  beforeLoad: internOnly,
  component: MentorChatPage,
})
