import { createFileRoute } from '@tanstack/react-router'
import { staffOnly } from '../../lib/guards'
import { MaterialsPage } from '../../pages/MaterialsPage'

export const Route = createFileRoute('/_auth/materials')({
  beforeLoad: staffOnly,
  component: MaterialsPage,
})
