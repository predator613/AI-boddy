import { createFileRoute } from '@tanstack/react-router'
import { leadOnly } from '../../lib/guards'
import { ApprovalsPage } from '../../pages/ApprovalsPage'

export const Route = createFileRoute('/_auth/approvals')({
  beforeLoad: leadOnly,
  component: ApprovalsPage,
})
