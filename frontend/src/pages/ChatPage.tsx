import {
  ActionIcon,
  Alert,
  Button,
  Group,
  Stack,
  Text,
  Textarea,
  UnstyledButton,
} from '@mantine/core'
import { IconArrowRight, IconPlus } from '@tabler/icons-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState, type FormEvent, type KeyboardEvent } from 'react'
import { cancelAction, confirmAction, sendMessage } from '../api/assistant'
import { Mascot } from '../components/Mascot'
import { ChatMessageView, ThinkingBubble } from '../components/chat/ChatMessageView'
import { errorText } from '../lib/errors'
import { pathKeys } from '../hooks/usePath'
import type { TChatMessage } from '../schemas/assistant'
import type { TPathView } from '../schemas/path'
import { useAuthStore } from '../store/authStore'
import classes from './ChatPage.module.css'

const SUGGESTIONS = [
  'Какой мой следующий шаг?',
  'Как оформить отгул?',
  'Кто мой наставник?',
  'Как устроено код-ревью?',
]

let localId = 0
const userMessage = (text: string): TChatMessage => ({
  id: `local-${++localId}`,
  role: 'user',
  kind: 'text',
  text,
  source: null,
  confirm: null,
  result: null,
  mood: 'calm',
})

interface ChatPageProps {
  initialQuestion?: string
  taskId?: string
}

// Чат с помощником. Запрос к LLM не блокирует интерфейс: виден статус «думает», ошибки можно повторить.
export function ChatPage({ initialQuestion, taskId }: ChatPageProps) {
  const user = useAuthStore((s) => s.user)
  const queryClient = useQueryClient()
  const [messages, setMessages] = useState<TChatMessage[]>([])
  const [text, setText] = useState('')
  const [lastFailed, setLastFailed] = useState<{
    text: string
    action: TPathView['stages'][number]['tasks'][number]['action']
  } | null>(null)
  const endRef = useRef<HTMLDivElement>(null)
  const autoSent = useRef(false)

  const send = useMutation({
    mutationFn: sendMessage,
    onSuccess: (reply) => setMessages((prev) => [...prev, ...reply.messages]),
    onError: (_e, vars) => setLastFailed({ text: vars.text, action: vars.action ?? null }),
  })

  const markConfirm = (actionId: string, state: 'confirmed' | 'cancelled') =>
    setMessages((prev) =>
      prev.map((m) =>
        m.confirm?.actionId === actionId ? { ...m, confirm: { ...m.confirm, state } } : m,
      ),
    )

  const act = useMutation({
    mutationFn: ({ actionId, ok }: { actionId: string; ok: boolean }) =>
      ok ? confirmAction(actionId) : cancelAction(actionId),
    onSuccess: (reply, { actionId, ok }) => {
      markConfirm(actionId, ok ? 'confirmed' : 'cancelled')
      setMessages((prev) => [...prev, ...reply.messages])
      if (ok) {
        queryClient.invalidateQueries({ queryKey: ['requests'] })
        queryClient.invalidateQueries({ queryKey: pathKeys.my })
      }
    },
  })

  const ask = (
    question: string,
    action: TPathView['stages'][number]['tasks'][number]['action'] = null,
    fromTask: string | null = null,
  ) => {
    const q = question.trim()
    if (!q || send.isPending) return
    setLastFailed(null)
    setMessages((prev) => [...prev, userMessage(q)])
    send.mutate({ text: q, action, taskId: fromTask })
  }

  // Пришли из «Моего пути» или панели помощника – сразу отправляем вопрос (и действие задачи).
  // Срабатывает один раз на новый вопрос, поэтому ask и queryClient в зависимости не нужны.
  useEffect(() => {
    if (autoSent.current || !initialQuestion) return
    autoSent.current = true
    const path = queryClient.getQueryData<TPathView>(pathKeys.my)
    const task = taskId
      ? path?.stages.flatMap((s) => s.tasks).find((t) => t.id === taskId)
      : undefined
    ask(initialQuestion, task?.action ?? null, task?.id ?? null)
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [initialQuestion, taskId])

  useEffect(() => {
    endRef.current?.scrollIntoView?.({ behavior: 'smooth', block: 'end' })
  }, [messages.length, send.isPending])

  const submit = (e: FormEvent) => {
    e.preventDefault()
    ask(text)
    setText('')
  }
  const onKey = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      ask(text)
      setText('')
    }
  }

  return (
    <div className={classes.chat}>
      <header className={classes.header}>
        <Mascot size={40} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <Text fw={600}>AI-buddy</Text>
          <Text size="xs" c="ink.5">
            знает твой этап, команду и права доступа
          </Text>
        </div>
        <Button
          variant="outline"
          color="ink"
          leftSection={<IconPlus size={16} />}
          onClick={() => {
            setMessages([])
            setLastFailed(null)
          }}
          disabled={messages.length === 0}
        >
          Новый диалог
        </Button>
      </header>

      <div className={classes.scroll}>
        <Stack gap={20} className={classes.column} aria-live="polite">
          {messages.length === 0 && !send.isPending && (
            <Stack align="center" gap="sm" py={48} className="ab-pop">
              <Mascot size={140} glow float />
              <Text fz={26} fw={500} ta="center" style={{ letterSpacing: '-0.02em' }}>
                Привет{user ? `, ${user.name}` : ''}! С чего начнем?
              </Text>
              <Text c="ink.5" ta="center" maw={460}>
                Спроси о процессах и инструментах или попроси оформить заявку. Я покажу, откуда взял
                ответ.
              </Text>
            </Stack>
          )}
          {messages.map((m) => (
            <ChatMessageView
              key={m.id}
              message={m}
              busy={act.isPending && act.variables?.actionId === m.confirm?.actionId}
              onConfirm={(actionId) => act.mutate({ actionId, ok: true })}
              onCancel={(actionId) => act.mutate({ actionId, ok: false })}
            />
          ))}
          {send.isPending && <ThinkingBubble />}
          {lastFailed && !send.isPending && (
            <Alert color="red" variant="light" role="alert" icon={<Mascot size={24} mood="sad" />}>
              {errorText(send.error)}{' '}
              <Button
                size="xs"
                variant="outline"
                color="ink"
                ml="xs"
                onClick={() => {
                  setMessages((prev) => prev.slice(0, -1))
                  ask(lastFailed.text, lastFailed.action)
                }}
              >
                Повторить
              </Button>
            </Alert>
          )}
          {act.isError && (
            <Alert color="red" variant="light" role="alert">
              {errorText(act.error)}
            </Alert>
          )}
          <div ref={endRef} />
        </Stack>
      </div>

      <div className={classes.composer}>
        <Stack gap={10} className={classes.column}>
          <Group gap={8}>
            {SUGGESTIONS.map((s) => (
              <UnstyledButton
                key={s}
                onClick={() => ask(s)}
                disabled={send.isPending}
                className={classes.chip}
              >
                {s}
              </UnstyledButton>
            ))}
          </Group>
          <form onSubmit={submit} className={classes.form}>
            <Textarea
              aria-label="Сообщение"
              placeholder="Спроси о процессах, доступах или попроси оформить заявку…"
              autosize
              minRows={1}
              maxRows={5}
              variant="unstyled"
              size="md"
              value={text}
              onChange={(e) => setText(e.currentTarget.value)}
              onKeyDown={onKey}
              style={{ flex: 1 }}
            />
            <ActionIcon
              type="submit"
              size={40}
              radius="md"
              aria-label="Отправить"
              disabled={!text.trim() || send.isPending}
            >
              <IconArrowRight size={20} />
            </ActionIcon>
          </form>
          <Text size="xs" c="ink.5" ta="center">
            AI-buddy может ошибаться. Важное уточняй у наставника. Данные демо-версии синтетические.
          </Text>
        </Stack>
      </div>
    </div>
  )
}
