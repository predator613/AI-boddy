import { createFileRoute } from '@tanstack/react-router'
import { staffOnly } from '../../lib/guards'
import { PlanEditorPage } from '../../pages/PlanEditorPage'

export const Route = createFileRoute('/_auth/plan')({
  beforeLoad: staffOnly,
  component: PlanEditorPage,
})
