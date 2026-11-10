import { createFileRoute } from '@tanstack/react-router'
import { InternsPage } from '../../pages/InternsPage'
import { MyPathPage } from '../../pages/MyPathPage'
import { useAuthStore } from '../../store/authStore'

interface HomeSearch {
  stage?: string
}

export const Route = createFileRoute('/_auth/')({
  validateSearch: (search: Record<string, unknown>): HomeSearch =>
    typeof search.stage === 'string' && search.stage ? { stage: search.stage } : {},
  component: HomePage,
})

// Главная зависит от роли: стажеру – «Мой путь», наставнику и руководителю – их стажеры.
function HomePage() {
  const role = useAuthStore((s) => s.user?.role)
  const { stage } = Route.useSearch()
  const navigate = Route.useNavigate()
  if (role !== 'intern') return <InternsPage />
  return (
    <MyPathPage
      stage={stage ?? null}
      onStage={(code) => navigate({ search: code ? { stage: code } : {}, replace: true })}
    />
  )
}
