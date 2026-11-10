import { Alert, Group, Paper, Stack, Text } from '@mantine/core'
import { Link } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { Mascot } from '../components/Mascot'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import { MessageComposer } from '../components/mentor/MessageComposer'
import { ThreadMessageView } from '../components/mentor/ThreadMessageView'
import { useAckUpdate, useMyMentorThread, useSendToMentor } from '../hooks/useMentor'
import { errorText } from '../lib/errors'
import { toast } from '../store/toastStore'

const QUICK = ['Можно перенести 1:1?', 'Посмотришь мое ревью?', 'Спасибо!']

// Чат стажера с наставником: ответы на вопросы, которые AI-buddy передал наставнику,
// и карточки изменений (план, материалы, встречи).
export function MentorChatPage() {
  const thread = useMyMentorThread()
  const send = useSendToMentor()
  const ack = useAckUpdate()
  const [draft, setDraft] = useState('')
  const [onlyUpdates, setOnlyUpdates] = useState(false)
  const endRef = useRef<HTMLDivElement>(null)
  const count = thread.data?.messages.length ?? 0

  useEffect(() => {
    endRef.current?.scrollIntoView({ block: 'end' })
  }, [count])

  if (thread.isPending)
    return (
      <Page>
        <LoadingBlocks label="Загрузка переписки" />
      </Page>
    )
  if (thread.isError)
    return (
      <Page>
        <ErrorBlock error={thread.error} onRetry={() => thread.refetch()} />
      </Page>
    )

  const t = thread.data
  const messages = onlyUpdates ? t.messages.filter((m) => m.kind === 'update') : t.messages
  const sendDraft = () =>
    send.mutate(draft.trim(), {
      onSuccess: () => {
        setDraft('')
        toast(`${t.mentor.name} получит уведомление о новом сообщении`)
      },
    })

  return (
    <Page maxWidth={1100}>
      <Stack gap="lg" style={{ flex: 1, minWidth: 0 }}>
        <Paper radius="lg" p="md">
          <Group justify="space-between" wrap="wrap" gap="sm">
            <Group gap={12} wrap="nowrap">
              <span
                aria-hidden="true"
                style={{
                  width: 44,
                  height: 44,
                  borderRadius: 22,
                  background: '#DFCDF3',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontWeight: 600,
                }}
              >
                {t.mentor.name.charAt(0)}
              </span>
              <div>
                <Group gap={8}>
                  <Text component="h1" fz={18} fw={600} m={0}>
                    {t.mentor.name}
                  </Text>
                  <StatusTag bg="#965EEB" color="#FFFFFF">
                    наставник
                  </StatusTag>
                </Group>
                <Text size="sm" c="ink.5">
                  обычно отвечает в течение часа · этап {t.stageCode} «{t.stageTitle}»
                </Text>
              </div>
            </Group>
            <StatusTag
              bg={onlyUpdates ? '#DFCDF3' : '#FFFFFF'}
              border="1px solid #D6D6D6"
              size="md"
              style={{ cursor: 'pointer' }}
            >
              <label style={{ cursor: 'pointer' }}>
                <input
                  type="checkbox"
                  checked={onlyUpdates}
                  onChange={(e) => setOnlyUpdates(e.currentTarget.checked)}
                  style={{ marginRight: 6, accentColor: '#965EEB' }}
                />
                Только изменения от наставника
              </label>
            </StatusTag>
          </Group>
        </Paper>

        <Group
          gap="sm"
          wrap="nowrap"
          p="sm"
          style={{ background: '#FFFFFF', border: '1px dashed #D6D6D6' }}
        >
          <Mascot size={28} />
          <Text size="sm">
            Сначала спроси{' '}
            <Link to="/chat" style={{ color: '#141414', fontWeight: 500 }}>
              AI-buddy
            </Link>
            : он отвечает сразу. Если он не знает ответ, сам передаст вопрос наставнику, и переписка
            появится здесь.
          </Text>
        </Group>

        <Stack gap={14} role="log" aria-label="Переписка с наставником">
          {messages.length === 0 && (
            <Text c="ink.5" ta="center">
              Пока ничего нет
            </Text>
          )}
          {messages.map((m) => (
            <ThreadMessageView
              key={m.id}
              message={m}
              viewer="intern"
              ackBusy={ack.isPending && ack.variables === m.id}
              onAck={(id) =>
                ack.mutate(id, { onSuccess: () => toast('Отмечено, наставник видит ответ') })
              }
            />
          ))}
          <div ref={endRef} />
        </Stack>

        {send.isError && (
          <Alert color="red" variant="light" role="alert">
            {errorText(send.error)}
          </Alert>
        )}
        <MessageComposer
          value={draft}
          onChange={setDraft}
          onSend={sendDraft}
          busy={send.isPending}
          label="Сообщение наставнику"
          placeholder={`Напиши ${t.mentor.name}…`}
          quick={QUICK}
          accent="#965EEB"
        />
      </Stack>
    </Page>
  )
}
