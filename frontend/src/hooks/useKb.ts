// Хуки справочника: статья целиком и поиск на сервере (русская морфология, только доступные статьи).
import { useDebouncedValue } from '@mantine/hooks'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { getKbArticle, getKnowledgeBase, searchKb } from '../api/assistant'

export const kbKeys = {
  all: ['kb'] as const,
  article: (id: string) => ['kb', 'article', id] as const,
  search: (q: string) => ['kb', 'search', q] as const,
}

export function useKnowledgeBase() {
  return useQuery({ queryKey: kbKeys.all, queryFn: getKnowledgeBase, staleTime: 5 * 60_000 })
}

export function useKbArticle(articleId: string | undefined) {
  return useQuery({
    queryKey: kbKeys.article(articleId ?? ''),
    queryFn: ({ signal }) => getKbArticle(articleId!, signal),
    enabled: Boolean(articleId),
    staleTime: 5 * 60_000,
  })
}

/** Поиск запускается через 250 мс после ввода и только от 2 символов. */
export function useKbSearch(raw: string) {
  const [q] = useDebouncedValue(raw.trim(), 250)
  return useQuery({
    queryKey: kbKeys.search(q),
    queryFn: ({ signal }) => searchKb(q, signal),
    enabled: q.length >= 2,
    placeholderData: keepPreviousData,
    staleTime: 60_000,
  })
}
