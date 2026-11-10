import { createFileRoute } from '@tanstack/react-router'
import { staffOnly } from '../../lib/guards'
import { SchedulePage } from '../../pages/SchedulePage'

export const Route = createFileRoute('/_auth/schedule')({
  beforeLoad: staffOnly,
  component: SchedulePage,
})
