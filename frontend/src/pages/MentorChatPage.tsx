import { Alert, Group, Stack, Switch, Text } from '@mantine/core'
import { Link } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { Mascot } from '../components/Mascot'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import { MessageComposer } from '../components/mentor/MessageComposer'
import { MaterialsFold } from '../components/mentor/MaterialsFold'
import { ThreadMessageView } from '../components/mentor/ThreadMessageView'
import { useAckUpdate, useMyMentorThread, useSendToMentor } from '../hooks/useMentor'
import { errorText } from '../lib/errors'
import { foldMaterials } from '../lib/mentorUi'
import { toast } from '../store/toastStore'
import classes from './ChatPage.module.css'

const QUICK = ['Можно перенести 1:1?', 'Посмотришь мое ревью?', 'Спасибо!']

// Чат стажера с наставником: ответы на вопросы, которые AI-buddy передал наставнику,
// и карточки изменений (план, материалы, встречи). Шапка и поле ввода закреплены,
// прокручивается только лента – как в мессенджере.
export function MentorChatPage() {
  const thread = useMyMentorThread()
  const send = useSendToMentor()
  const ack = useAckUpdate()
  const [draft, setDraft] = useState('')
  const [onlyUpdates, setOnlyUpdates] = useState(false)
  const [allMaterials, setAllMaterials] = useState(false)
  const endRef = useRef<HTMLDivElement>(null)
  const count = thread.data?.messages.length ?? 0

  useEffect(() => {
    endRef.current?.scrollIntoView?.({ block: 'end' })
  }, [count, onlyUpdates])

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
  const fold = foldMaterials(t.messages)
  const recent = allMaterials ? t.messages : fold.visible
  const messages = onlyUpdates ? recent.filter((m) => m.kind === 'update') : recent
  const sendDraft = () =>
    send.mutate(draft.trim(), {
      onSuccess: () => {
        setDraft('')
        toast(`${t.mentor.name} получит уведомление о новом сообщении`)
      },
    })

  return (
    <div className={classes.chat}>
      <header className={classes.header}>
        <span
          aria-hidden="true"
          style={{
            width: 40,
            height: 40,
            borderRadius: 20,
            background: '#DFCDF3',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontWeight: 600,
            flex: 'none',
          }}
        >
          {t.mentor.name.charAt(0)}
        </span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <Group gap={8}>
            <Text component="h1" fz={16} fw={600} m={0}>
              {t.mentor.name}
            </Text>
            <StatusTag bg="#965EEB" color="#FFFFFF">
              наставник
            </StatusTag>
          </Group>
          <Text size="xs" c="ink.5" truncate>
            обычно отвечает в течение часа · этап {t.stageCode} «{t.stageTitle}»
          </Text>
        </div>
        <Switch
          color="grape.6"
          size="sm"
          label="Только изменения"
          checked={onlyUpdates}
          onChange={(e) => setOnlyUpdates(e.currentTarget.checked)}
          aria-label="Только изменения от наставника"
        />
      </header>

      <div className={classes.scroll}>
        <Stack gap={14} className={classes.column}>
          <Group
            gap="sm"
            wrap="nowrap"
            p="sm"
            style={{ background: '#FFFFFF', border: '1px dashed #D6D6D6', borderRadius: 12 }}
          >
            <Mascot size={28} />
            <Text size="sm">
              Сначала спроси{' '}
              <Link to="/chat" style={{ color: '#141414', fontWeight: 500 }}>
                AI-buddy
              </Link>
              : он отвечает сразу. Если он не знает ответ, сам передаст вопрос наставнику, и
              переписка появится здесь.
            </Text>
          </Group>
          <Stack gap={14} role="log" aria-label="Переписка с наставником">
            <MaterialsFold
              hidden={fold.hidden}
              open={allMaterials}
              onToggle={() => setAllMaterials((v) => !v)}
            />
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
          </Stack>
          <div ref={endRef} />
        </Stack>
      </div>

      <div className={classes.composer}>
        <Stack gap={10} className={classes.column}>
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
      </div>
    </div>
  )
}
