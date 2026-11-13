import { createFileRoute } from '@tanstack/react-router'
import { LeadGoalsPage } from '../../pages/LeadGoalsPage'
import { MentorHomePage } from '../../pages/MentorHomePage'
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

// Главная зависит от роли: стажеру – «Мой путь», наставнику – его стажеры, руководителю – цели стажеров.
function HomePage() {
  const role = useAuthStore((s) => s.user?.role)
  const { stage } = Route.useSearch()
  const navigate = Route.useNavigate()
  if (role === 'mentor') return <MentorHomePage />
  if (role === 'lead') return <LeadGoalsPage />
  return (
    <MyPathPage
      stage={stage ?? null}
      onStage={(code) =>
        navigate({ search: code ? { stage: code } : {}, replace: true, resetScroll: false })
      }
    />
  )
}
