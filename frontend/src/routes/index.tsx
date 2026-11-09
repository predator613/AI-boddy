import { createFileRoute } from '@tanstack/react-router'
import { HomePlaceholder } from '../components/HomePlaceholder'

export const Route = createFileRoute('/')({
  component: HomePlaceholder,
})
