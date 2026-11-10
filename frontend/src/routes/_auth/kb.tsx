import { useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { pathKeys } from '../../hooks/usePath'
import { KnowledgePage } from '../../pages/KnowledgePage'
import type { TPathView } from '../../schemas/path'

export const Route = createFileRoute('/_auth/kb')({
  validateSearch: (s: Record<string, unknown>): { article?: string } =>
    typeof s.article === 'string' && s.article ? { article: s.article } : {},
  component: function KbRoute() {
    const { article } = Route.useSearch()
    const navigate = Route.useNavigate()
    const path = useQueryClient().getQueryData<TPathView>(pathKeys.my)
    const stageCode = path?.stages.find((s) => s.status === 'current')?.code ?? null
    return (
      <KnowledgePage
        articleId={article}
        stageCode={stageCode}
        onArticle={(id) => navigate({ search: id ? { article: id } : {}, replace: true })}
      />
    )
  },
})
