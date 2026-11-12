import { Alert, Checkbox, Group, Stack, Text, UnstyledButton } from '@mantine/core'
import { useEffect, useRef, useState } from 'react'
import { MaterialsFold } from '../components/mentor/MaterialsFold'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import { MessageComposer } from '../components/mentor/MessageComposer'
import { ThreadMessageView } from '../components/mentor/ThreadMessageView'
import { timeLabel, foldMaterials } from '../lib/mentorUi'
import { useMarkThreadRead, useReplyToIntern, useThread, useThreads } from '../hooks/useMentor'
import { errorText } from '../lib/errors'
import { toast } from '../store/toastStore'
import classes from './ChatPage.module.css'

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
  // неизвестный id в ссылке (например, со старой закладки) – открываем первую переписку
  const known = list.some((t) => t.internId === selectedId)
  const currentId = (known ? selectedId : list[0]?.internId) ?? null
  const thread = useThread(currentId)
  const markRead = useMarkThreadRead()
  const reply = useReplyToIntern(currentId ?? '')
  const [draft, setDraft] = useState('')
  const [addToKb, setAddToKb] = useState(false)
  const endRef = useRef<HTMLDivElement>(null)
  const unread = list.find((t) => t.internId === currentId)?.unread ?? 0
  const count = thread.data?.messages.length ?? 0
  const [allMaterials, setAllMaterials] = useState(false)
  const fold = foldMaterials(thread.data?.messages ?? [])

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
    <div className={classes.shell}>
      <nav
        aria-label="Стажеры"
        className={classes.history}
        style={{ padding: 0, gap: 0, overflowY: 'auto' }}
      >
        <Text component="h1" fz={18} fw={600} m={0} px={18} py={20}>
          Сообщения
        </Text>
        {threads.isPending && <LoadingBlocks label="Загрузка сообщений" />}
        {threads.isError && <ErrorBlock error={threads.error} onRetry={() => threads.refetch()} />}
        {threads.data && list.length === 0 && (
          <Text px={18} c="ink.5">
            Пока вопросов нет: AI-buddy справляется сам.
          </Text>
        )}
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

      <div className={classes.chat}>
        {thread.isPending && <LoadingBlocks label="Загрузка переписки" />}
        {thread.isError && <ErrorBlock error={thread.error} onRetry={() => thread.refetch()} />}
        {thread.data && (
          <>
            <header className={classes.header}>
              <Text component="h2" fz={18} fw={500} m={0}>
                {thread.data.internName}
              </Text>
              <StatusTag bg="#BADAFB">
                Этап {thread.data.stageCode} · {thread.data.stageTitle}
              </StatusTag>
            </header>
            <div className={classes.scroll}>
              <Stack
                gap={14}
                className={classes.column}
                role="log"
                aria-label={`Переписка с ${thread.data.internName}`}
              >
                <MaterialsFold
                  hidden={fold.hidden}
                  open={allMaterials}
                  onToggle={() => setAllMaterials((v) => !v)}
                />
                {(allMaterials ? thread.data.messages : fold.visible).map((m) => (
                  <ThreadMessageView key={m.id} message={m} viewer="mentor" />
                ))}
                <div ref={endRef} />
              </Stack>
            </div>
            <div className={classes.composer}>
              <Stack gap={8} className={classes.column}>
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
            </div>
          </>
        )}
      </div>
    </div>
  )
}
