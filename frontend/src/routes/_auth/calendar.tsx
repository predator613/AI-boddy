import { createFileRoute } from '@tanstack/react-router'
import { internOnly } from '../../lib/guards'
import { CalendarPage } from '../../pages/CalendarPage'

export const Route = createFileRoute('/_auth/calendar')({
  beforeLoad: internOnly,
  component: () => <CalendarPage />,
})
