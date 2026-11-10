import { Alert, Checkbox, Group, Paper, Stack, Text, UnstyledButton } from '@mantine/core'
import { useEffect, useRef, useState } from 'react'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import { MessageComposer } from '../components/mentor/MessageComposer'
import { ThreadMessageView } from '../components/mentor/ThreadMessageView'
import { timeLabel } from '../lib/mentorUi'
import { useMarkThreadRead, useReplyToIntern, useThread, useThreads } from '../hooks/useMentor'
import { errorText } from '../lib/errors'
import { toast } from '../store/toastStore'

const COLORS = ['#BADAFB', '#DFCDF3', '#C9FFBF', '#F7C1C5']
const QUICK = ['Обсудим на 1:1', 'Посмотрю сегодня', 'Да, все верно']

interface Props {
  selectedId?: string
  onSelect: (internId: string) => void
}

// «Сообщения» наставника: вопросы, которые AI-buddy передал, и переписка со стажерами.
export function InboxPage({ selectedId, onSelect }: Props) {
  const threads = useThreads()
  const list = threads.data ?? []
  const currentId = selectedId ?? list[0]?.internId ?? null
  const thread = useThread(currentId)
  const markRead = useMarkThreadRead()
  const reply = useReplyToIntern(currentId ?? '')
  const [draft, setDraft] = useState('')
  const [addToKb, setAddToKb] = useState(false)
  const endRef = useRef<HTMLDivElement>(null)
  const unread = list.find((t) => t.internId === currentId)?.unread ?? 0
  const count = thread.data?.messages.length ?? 0

  useEffect(() => {
    if (currentId && unread > 0) markRead.mutate(currentId)
    // отмечаем прочитанным, только когда открыли тред с новыми сообщениями
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [currentId, unread])

  useEffect(() => {
    endRef.current?.scrollIntoView({ block: 'end' })
  }, [count, currentId])

  const send = () => {
    const name = thread.data?.internName ?? 'Стажер'
    reply.mutate(
      { text: draft.trim(), addToKb },
      {
        onSuccess: () => {
          setDraft('')
          setAddToKb(false)
          toast(
            addToKb
              ? `${name} получит ответ, а AI-buddy добавил его в справочник`
              : `${name} получит ответ и уведомление`,
          )
        },
      },
    )
  }

  return (
    <Page maxWidth={1240}>
      <Stack gap="lg" style={{ flex: 1, minWidth: 0 }}>
        <PageHeader
          overline="Сообщения · вопросы, которые AI-buddy передал тебе"
          title={
            <>
              Вопросы <Marker color="#965EEB">стажеров</Marker>
            </>
          }
        />
        {threads.isPending && <LoadingBlocks label="Загрузка сообщений" />}
        {threads.isError && <ErrorBlock error={threads.error} onRetry={() => threads.refetch()} />}
        {threads.data && list.length === 0 && (
          <Paper radius="lg" p="lg">
            <Text>Пока вопросов нет: AI-buddy справляется сам.</Text>
          </Paper>
        )}
        {list.length > 0 && (
          <Paper
            radius="lg"
            style={{ overflow: 'hidden', display: 'flex', flexWrap: 'wrap', minHeight: 560 }}
          >
            <nav
              aria-label="Стажеры"
              style={{ flex: '0 1 300px', minWidth: 240, borderRight: '1px solid #D6D6D6' }}
            >
              {list.map((t, i) => {
                const active = t.internId === currentId
                return (
                  <UnstyledButton
                    key={t.internId}
                    onClick={() => onSelect(t.internId)}
                    aria-current={active ? 'true' : undefined}
                    aria-label={`${t.internName}${t.unread ? `, новых: ${t.unread}` : ''}`}
                    style={{
                      display: 'flex',
                      gap: 12,
                      width: '100%',
                      padding: '16px 18px',
                      borderBottom: '1px solid #D6D6D6',
                      background: active ? '#F4F4F4' : '#FFFFFF',
                    }}
                  >
                    <span
                      aria-hidden="true"
                      style={{
                        width: 40,
                        height: 40,
                        borderRadius: 20,
                        background: COLORS[i % COLORS.length],
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontWeight: 600,
                        flex: 'none',
                      }}
                    >
                      {t.internName.charAt(0)}
                    </span>
                    <span style={{ flex: 1, minWidth: 0 }}>
                      <Group justify="space-between" gap={8} wrap="nowrap">
                        <Text fw={600}>{t.internName}</Text>
                        <Text size="xs" c="ink.5">
                          {t.lastAt ? timeLabel(t.lastAt) : ''}
                        </Text>
                      </Group>
                      <Text size="sm" c="ink.5" truncate>
                        {t.lastText}
                      </Text>
                    </span>
                    {t.unread > 0 && (
                      <span
                        style={{
                          minWidth: 20,
                          height: 20,
                          borderRadius: 10,
                          background: '#965EEB',
                          color: '#FFFFFF',
                          fontSize: 11,
                          fontWeight: 600,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                        }}
                      >
                        {t.unread}
                      </span>
                    )}
                  </UnstyledButton>
                )
              })}
            </nav>

            <Stack gap={0} style={{ flex: '999 1 420px', minWidth: 0 }}>
              {thread.isPending && <LoadingBlocks label="Загрузка переписки" />}
              {thread.isError && (
                <ErrorBlock error={thread.error} onRetry={() => thread.refetch()} />
              )}
              {thread.data && (
                <>
                  <Group gap={12} px="lg" py="md" style={{ borderBottom: '1px solid #D6D6D6' }}>
                    <Text component="h2" fz={18} fw={500} m={0}>
                      {thread.data.internName}
                    </Text>
                    <StatusTag bg="#BADAFB">
                      Этап {thread.data.stageCode} · {thread.data.stageTitle}
                    </StatusTag>
                  </Group>
                  <Stack
                    gap={14}
                    p="lg"
                    role="log"
                    aria-label={`Переписка с ${thread.data.internName}`}
                    style={{ flex: 1, background: '#F4F4F4' }}
                  >
                    {thread.data.messages.map((m) => (
                      <ThreadMessageView key={m.id} message={m} viewer="mentor" />
                    ))}
                    <div ref={endRef} />
                  </Stack>
                  <Stack gap={10} p="lg" style={{ borderTop: '1px solid #D6D6D6' }}>
                    {reply.isError && (
                      <Alert color="red" variant="light" role="alert">
                        {errorText(reply.error)}
                      </Alert>
                    )}
                    <MessageComposer
                      value={draft}
                      onChange={setDraft}
                      onSend={send}
                      busy={reply.isPending}
                      label="Ответ стажеру"
                      placeholder="Ответь стажеру…"
                      quick={QUICK}
                      footer={
                        <Checkbox
                          color="green.6"
                          checked={addToKb}
                          onChange={(e) => setAddToKb(e.currentTarget.checked)}
                          label="Добавить ответ в справочник – AI-buddy будет отвечать так сам"
                        />
                      }
                    />
                  </Stack>
                </>
              )}
            </Stack>
          </Paper>
        )}
      </Stack>
    </Page>
  )
}
