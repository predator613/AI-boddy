import {
  Anchor,
  Button,
  Group,
  Modal,
  Paper,
  Stack,
  Text,
  TextInput,
  Title,
  UnstyledButton,
} from '@mantine/core'
import { IconArrowLeft, IconChevronRight, IconFileText, IconSearch } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useKbArticle, useKbSearch, useKnowledgeBase } from '../hooks/useKb'
import { Mascot } from '../components/Mascot'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import type { TKbArticle } from '../schemas/assistant'

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
            const sectionColor = new Map(kb.data.sections.map((x) => [x.id, x.color]))
            const listTitleFinal =
              q || section ? listTitle : forStage.length ? 'Часто задаваемые вопросы' : 'Популярное'
            return (
              <>
                <Paper radius={20} p="lg" component="section" aria-label={listTitleFinal}>
                  <Group justify="space-between" mb="xs" gap={8}>
                    <Title order={2}>{listTitleFinal}</Title>
                    <Group gap={8}>
                      {stageCode && !q && !section && forStage.length > 0 && (
                        <StatusTag bg="#BADAFB">Этап {stageCode}</StatusTag>
                      )}
                      {(q || section) && (
                        <Button
                          size="xs"
                          radius="xl"
                          variant="outline"
                          color="ink"
                          leftSection={<IconArrowLeft size={14} />}
                          onClick={() => {
                            setSection(null)
                            setSearch('')
                          }}
                        >
                          Назад
                        </Button>
                      )}
                    </Group>
                  </Group>
                  {list.length === 0 ? (
                    <Group
                      gap="md"
                      py="md"
                      wrap="nowrap"
                      style={{ borderTop: '1px solid #EDEDED' }}
                    >
                      <Mascot size={52} mood="think" />
                      <Text size="sm">
                        Ничего не нашлось. Попробуй спросить{' '}
                        <Anchor renderRoot={(p) => <Link {...p} to="/chat" />} c="ink.9" fw={500}>
                          AI-buddy
                        </Anchor>
                        : он поищет по всем разделам и покажет источник.
                      </Text>
                    </Group>
                  ) : (
                    list.map((a) => (
                      <ArticleRow
                        key={a.id}
                        article={a}
                        color={sectionColor.get(a.sectionId) ?? '#F4F4F4'}
                        forStage={
                          Boolean(stageCode) &&
                          a.stageCode === stageCode &&
                          (Boolean(q) || Boolean(section))
                        }
                        onOpen={() => onArticle(a.id)}
                      />
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
                          <Text key={i} style={{ whiteSpace: 'pre-line' }}>
                            {p}
                          </Text>
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

      <Stack
        gap={24}
        pt={{ base: 0, md: 70 }}
        style={{ flex: '1 1 300px', maxWidth: 340, minWidth: 0 }}
      >
        {kb.data && (
          <nav aria-label="Разделы справочника">
            <Title order={3} fz={18} mb={10}>
              Поиск по разделам
            </Title>
            <Stack gap={4}>
              {kb.data.sections.map((s) => {
                const on = section === s.id
                return (
                  <UnstyledButton
                    key={s.id}
                    onClick={() => {
                      setSection(on ? null : s.id)
                      setSearch('')
                    }}
                    aria-pressed={on}
                    className="ab-nudge"
                    style={{
                      display: 'grid',
                      gridTemplateColumns: '1fr auto',
                      alignItems: 'end',
                      columnGap: 12,
                      padding: '6px 0',
                    }}
                  >
                    <span
                      className="ab-hand"
                      style={{
                        gridColumn: '1 / -1',
                        fontSize: 17,
                        lineHeight: 1,
                        color: '#5C5C5C',
                      }}
                    >
                      {s.hand}
                    </span>
                    <span style={{ fontSize: 18, lineHeight: 1.3, fontWeight: on ? 500 : 400 }}>
                      <span
                        style={{
                          background: on ? '#00AAFF' : 'transparent',
                          padding: '0 4px',
                          marginLeft: -4,
                        }}
                      >
                        {s.title}
                      </span>
                    </span>
                    <Text span size="xs" c="ink.5" pb={3}>
                      {s.count}
                    </Text>
                  </UnstyledButton>
                )
              })}
            </Stack>
          </nav>
        )}
        <Paper radius={20} p="xl" style={{ background: '#141414', color: '#FFFFFF' }}>
          <Stack align="center" gap="sm" ta="center">
            <Mascot size={84} glow float />
            <Text fz={20} fw={500}>
              Не нашла ответ?
            </Text>
            <Text size="sm" c="#D6D6D6" lh={1.45}>
              Спроси AI-buddy: он поищет по всем разделам и покажет, откуда взял ответ.
            </Text>
            <Button
              renderRoot={(p) => <Link {...p} to="/chat" />}
              variant="white"
              color="ink"
              radius="md"
            >
              Спросить AI-buddy
            </Button>
          </Stack>
        </Paper>
      </Stack>
    </Page>
  )
}

function ArticleRow({
  article,
  color,
  forStage,
  onOpen,
}: {
  article: TKbArticle
  color: string
  forStage: boolean
  onOpen: () => void
}) {
  return (
    <UnstyledButton
      onClick={onOpen}
      className="ab-row"
      style={{
        width: '100%',
        display: 'flex',
        alignItems: 'center',
        gap: 14,
        padding: '14px 0',
        borderTop: '1px solid #EDEDED',
      }}
    >
      <span
        style={{
          width: 40,
          height: 40,
          flex: 'none',
          borderRadius: 10,
          background: color,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <IconFileText size={18} stroke={1.8} aria-hidden="true" />
      </span>
      <span style={{ flex: 1, minWidth: 0 }}>
        <Text fw={500} size="15px">
          {article.title}
        </Text>
        <Text size="sm" c="ink.5" truncate>
          {article.summary} · {article.minutes} мин
        </Text>
      </span>
      {forStage && <StatusTag bg="#BADAFB">твой этап</StatusTag>}
      <IconChevronRight size={18} color="#5C5C5C" aria-hidden="true" />
    </UnstyledButton>
  )
}
