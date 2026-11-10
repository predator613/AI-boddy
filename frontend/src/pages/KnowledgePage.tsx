import {
  Anchor,
  Button,
  Group,
  Modal,
  Paper,
  SimpleGrid,
  Stack,
  Text,
  TextInput,
  Title,
  UnstyledButton,
} from '@mantine/core'
import { IconChevronRight, IconSearch } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useKbArticle, useKbSearch, useKnowledgeBase } from '../hooks/useKb'
import { Mascot } from '../components/Mascot'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import type { TKbArticle } from '../schemas/assistant'

const TILTS = [-1.2, 0, 1, 0, -0.8, 0]

interface KnowledgePageProps {
  articleId?: string
  onArticle: (id: string | null) => void
  stageCode?: string | null
}

export function KnowledgePage({ articleId, onArticle, stageCode }: KnowledgePageProps) {
  const kb = useKnowledgeBase()
  const [search, setSearch] = useState('')
  const found = useKbSearch(search)
  const detail = useKbArticle(articleId)
  const [section, setSection] = useState<string | null>(null)

  return (
    <Page maxWidth={1400}>
      <Stack gap={28} style={{ flex: '999 1 560px', minWidth: 0 }}>
        <Stack gap="md">
          <PageHeader
            overline="Справочник · материалы для стажеров"
            title={
              <>
                Как у нас <Marker color="#00AAFF">все устроено</Marker>
              </>
            }
          />
          <TextInput
            aria-label="Поиск по справочнику"
            placeholder="Найди статью: доступы, отгул, код-ревью…"
            size="lg"
            radius="md"
            maw={720}
            leftSection={<IconSearch size={20} />}
            value={search}
            onChange={(e) => setSearch(e.currentTarget.value)}
          />
        </Stack>
        {kb.isPending && <LoadingBlocks label="Загрузка справочника" />}
        {kb.isError && <ErrorBlock error={kb.error} onRetry={() => kb.refetch()} />}
        {kb.data &&
          (() => {
            const q = search.trim().toLowerCase()
            // Поиск на сервере (морфология, порядок по релевантности); пока ответа нет – фильтр по названию.
            const byId = new Map(kb.data.articles.map((a) => [a.id, a]))
            const serverHits = found.data?.results
              .map((r) => byId.get(r.articleId))
              .filter((a): a is TKbArticle => Boolean(a))
            const filtered =
              q && serverHits
                ? serverHits.filter((a) => !section || a.sectionId === section)
                : kb.data.articles.filter(
                    (a) =>
                      (!section || a.sectionId === section) &&
                      (!q || `${a.title} ${a.summary}`.toLowerCase().includes(q)),
                  )
            const forStage = stageCode
              ? kb.data.articles.filter((a) => a.stageCode === stageCode)
              : []
            const list =
              q || section ? filtered : forStage.length ? forStage : kb.data.articles.slice(0, 4)
            const listTitle = q
              ? `Найдено: ${filtered.length}`
              : section
                ? kb.data.sections.find((s) => s.id === section)?.title
                : forStage.length
                  ? 'Нужно на твоем этапе'
                  : 'Популярное'
            const open = kb.data.articles.find((a) => a.id === articleId) ?? null
            return (
              <>
                {!q && (
                  <Stack gap="sm">
                    <Group justify="space-between">
                      <Title order={2}>Разделы</Title>
                      {section && (
                        <Anchor component="button" c="ink.6" onClick={() => setSection(null)}>
                          Все разделы
                        </Anchor>
                      )}
                    </Group>
                    <SimpleGrid cols={{ base: 1, xs: 2, md: 3 }} spacing="md">
                      {kb.data.sections.map((s, i) => (
                        <UnstyledButton
                          key={s.id}
                          className="ab-lift"
                          onClick={() => setSection(section === s.id ? null : s.id)}
                          aria-pressed={section === s.id}
                          style={{
                            background: s.color,
                            padding: 18,
                            minHeight: 140,
                            display: 'flex',
                            flexDirection: 'column',
                            gap: 8,
                            transform: `rotate(${TILTS[i % TILTS.length]}deg)`,
                            outline: section === s.id ? '2px solid #141414' : 0,
                            outlineOffset: -2,
                          }}
                        >
                          <span className="ab-hand" style={{ fontSize: 22, lineHeight: 1 }}>
                            {s.hand}
                          </span>
                          <span
                            style={{ fontSize: 18, fontWeight: 500, lineHeight: 1.15, flex: 1 }}
                          >
                            {s.title}
                          </span>
                          <span style={{ fontSize: 12 }}>
                            {s.count} {s.count === 1 ? 'статья' : s.count < 5 ? 'статьи' : 'статей'}
                          </span>
                        </UnstyledButton>
                      ))}
                    </SimpleGrid>
                  </Stack>
                )}
                <Paper radius="lg" p="lg">
                  <Group justify="space-between" mb="xs">
                    <Title order={2}>{listTitle}</Title>
                    {stageCode && !q && !section && forStage.length > 0 && (
                      <StatusTag bg="#BADAFB">Этап {stageCode}</StatusTag>
                    )}
                  </Group>
                  {list.length === 0 ? (
                    <Text c="ink.5" py="md">
                      Ничего не нашлось. Попробуй спросить AI-buddy.
                    </Text>
                  ) : (
                    list.map((a) => (
                      <ArticleRow key={a.id} article={a} onOpen={() => onArticle(a.id)} />
                    ))
                  )}
                </Paper>
                <Text size="xs" c="ink.5">
                  Демо-материалы: тексты синтетические и не являются регламентами Авито.
                </Text>
                <Modal
                  opened={Boolean(open)}
                  onClose={() => onArticle(null)}
                  title={open?.title}
                  size="lg"
                  radius="lg"
                >
                  {open && (
                    <Stack>
                      <Text fw={500}>{open.summary}</Text>
                      {detail.data?.body
                        .split(/\n{2,}/)
                        .filter(Boolean)
                        .map((p, i) => (
                          <Text key={i}>{p}</Text>
                        ))}
                      {detail.isPending && <Text c="ink.5">Загружаем текст…</Text>}
                      <Text size="sm" c="ink.5">
                        {open.minutes} мин чтения
                        {detail.data?.authorName ? ` · ${detail.data.authorName}` : ''}
                      </Text>
                      <Button
                        renderRoot={(p) => (
                          <Link
                            {...p}
                            to="/chat"
                            search={{ q: `Расскажи подробнее: ${open.title}` }}
                          />
                        )}
                        color="ink"
                        style={{ alignSelf: 'flex-start' }}
                      >
                        Спросить AI-buddy
                      </Button>
                    </Stack>
                  )}
                </Modal>
              </>
            )
          })()}
      </Stack>

      <Stack gap="md" style={{ flex: '1 1 300px', maxWidth: 400, minWidth: 0 }}>
        <Paper radius="lg" p="lg">
          <Stack align="center" gap="sm" ta="center">
            <Mascot size={96} glow />
            <Text fz={18} fw={500}>
              Не нашла ответ?
            </Text>
            <Text size="sm" c="ink.5">
              Спроси AI-buddy: он поищет по всем разделам и покажет, откуда взял ответ.
            </Text>
            <Button component={Link} to="/chat" color="ink">
              Спросить AI-buddy
            </Button>
          </Stack>
        </Paper>
        {kb.data && (
          <Paper radius="lg" p="md">
            <Title order={3} mb={8}>
              Словарик
            </Title>
            {kb.data.terms.map((t) => (
              <div key={t.term} style={{ padding: '10px 0', borderTop: '1px solid #D6D6D6' }}>
                <Text size="sm" fw={500}>
                  {t.term}
                </Text>
                <Text size="sm" c="ink.5">
                  {t.def}
                </Text>
              </div>
            ))}
          </Paper>
        )}
      </Stack>
    </Page>
  )
}

function ArticleRow({ article, onOpen }: { article: TKbArticle; onOpen: () => void }) {
  return (
    <UnstyledButton
      onClick={onOpen}
      style={{
        width: '100%',
        display: 'flex',
        alignItems: 'center',
        gap: 14,
        padding: '14px 0',
        borderTop: '1px solid #D6D6D6',
      }}
    >
      <span style={{ flex: 1, minWidth: 0 }}>
        <Text fw={500} size="15px">
          {article.title}
        </Text>
        <Text size="sm" c="ink.5">
          {article.summary} · {article.minutes} мин
        </Text>
      </span>
      <IconChevronRight size={18} color="#5C5C5C" aria-hidden="true" />
    </UnstyledButton>
  )
}
