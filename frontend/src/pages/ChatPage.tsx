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
import { IconArrowLeft, IconArrowRight } from '@tabler/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState, type FormEvent, type KeyboardEvent } from 'react'
import {
  cancelAction,
  confirmAction,
  getChatHistory,
  getRequests,
  resetChat,
  sendMessage,
} from '../api/assistant'
import { Mascot } from '../components/Mascot'
import { ChatMessageView, ThinkingBubble } from '../components/chat/ChatMessageView'
import { ChatHistory } from '../components/chat/ChatHistory'
import { chatKey, dialogsKey } from '../lib/chatKeys'
import { getDialogMessages } from '../api/chatDialogs'
import { errorText } from '../lib/errors'
import { pathKeys, useMyPath } from '../hooks/usePath'
import { chatHints } from '../lib/buddyHints'
import { ApiError } from '../api/client'
import { StatusTag } from '../components/StatusTag'
import { toast } from '../store/toastStore'
import type { TChatMessage, TChatReply } from '../schemas/assistant'
import type { TPathView } from '../schemas/path'
import { useAuthStore } from '../store/authStore'
import classes from './ChatPage.module.css'

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

// Переписка живет в backend (GET /api/chat/messages) и в кеше запросов: после перехода на другую
// страницу или перезагрузки чат восстанавливается вместе с состоянием карточек.

interface ChatPageProps {
  initialQuestion?: string
  taskId?: string
}

// Чат с помощником. Запрос к LLM не блокирует интерфейс: виден статус «думает», ошибки можно повторить.
export function ChatPage({ initialQuestion, taskId }: ChatPageProps) {
  const user = useAuthStore((s) => s.user)
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [text, setText] = useState('')
  const [lastFailed, setLastFailed] = useState<{
    text: string
    action: TPathView['stages'][number]['tasks'][number]['action']
  } | null>(null)
  const endRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const path = useMyPath({ enabled: user?.role === 'intern' })
  // Подсказки под полем ввода – по ситуации стажера: заявка на согласовании, следующий шаг.
  const requests = useQuery({
    queryKey: ['requests'],
    queryFn: getRequests,
    enabled: user?.role === 'intern',
  })
  const hints = chatHints(path.data, requests.data)
  // Тот же этап, что «СЕЙЧАС» на «Моем пути», а не этап следующей задачи: иначе экраны расходятся.
  const stage =
    path.data?.stages.find((s) => s.status === 'current') ??
    path.data?.stages.find((s) => s.code === path.data?.nextStep?.stageCode)
  const autoSent = useRef(false)

  const history = useQuery({ queryKey: chatKey, queryFn: getChatHistory })
  // открытый прошлый диалог (только чтение); null – текущий
  const [openId, setOpenId] = useState<string | null>(null)
  const past = useQuery({
    queryKey: ['chat', 'dialog', openId],
    queryFn: () => getDialogMessages(openId!),
    enabled: Boolean(openId),
  })
  const messages = history.data?.messages ?? []
  const setMessages = (update: (prev: TChatMessage[]) => TChatMessage[]) =>
    queryClient.setQueryData<TChatReply>(chatKey, (old) => ({
      messages: update(old?.messages ?? []),
    }))

  const send = useMutation({
    mutationFn: sendMessage,
    onSuccess: (reply) => {
      setMessages((prev) => [...prev, ...reply.messages])
      queryClient.invalidateQueries({ queryKey: dialogsKey })
      // backend мог отменить прошлую карточку подтверждения – берем свежую историю
      queryClient.invalidateQueries({ queryKey: chatKey })
    },
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
    onError: (e) => {
      // Карточку уже отменил backend (стажер поправил заявку или начал другую) – обновляем чат.
      if (e instanceof ApiError && e.status === 404) {
        toast('Эта карточка уже неактуальна')
        queryClient.invalidateQueries({ queryKey: chatKey })
      }
    },
  })

  const reset = useMutation({
    mutationFn: resetChat,
    onSuccess: (reply) => {
      queryClient.setQueryData<TChatReply>(chatKey, reply)
      queryClient.invalidateQueries({ queryKey: dialogsKey })
      setLastFailed(null)
      setOpenId(null)
    },
  })

  // Пока заявка из чата ждет решения, переписка обновляется сама: ответ отдела или
  // согласующего приходит в чат сообщением, а карточка меняет статус.
  const waiting = messages.some((m) => m.result?.status === 'waiting')
  const busy = send.isPending || act.isPending
  useEffect(() => {
    if (!waiting || busy) return
    const timer = setInterval(() => queryClient.invalidateQueries({ queryKey: chatKey }), 15_000)
    return () => clearInterval(timer)
  }, [waiting, busy, queryClient])

  const ask = (
    question: string,
    action: TPathView['stages'][number]['tasks'][number]['action'] = null,
    fromTask: string | null = null,
  ) => {
    const q = question.trim()
    if (!q || send.isPending) return
    void queryClient.cancelQueries({ queryKey: chatKey })
    setLastFailed(null)
    setMessages((prev) => [...prev, userMessage(q)])
    send.mutate({ text: q, action, taskId: fromTask })
  }

  // Пришли из «Моего пути» или панели помощника – сразу отправляем вопрос (и действие задачи).
  // Срабатывает один раз на новый вопрос, поэтому ask и queryClient в зависимости не нужны.
  // Вопрос отправляется после загрузки переписки, чтобы встать в конец, а не перед историей.
  useEffect(() => {
    if (autoSent.current || !initialQuestion || history.isPending) return
    autoSent.current = true
    const path = queryClient.getQueryData<TPathView>(pathKeys.my)
    const task = taskId
      ? path?.stages.flatMap((s) => s.tasks).find((t) => t.id === taskId)
      : undefined
    ask(initialQuestion, task?.action ?? null, task?.id ?? null)
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [initialQuestion, taskId, history.isPending])

  useEffect(() => {
    endRef.current?.scrollIntoView?.({ behavior: 'smooth', block: 'end' })
  }, [messages.length, send.isPending, openId, past.data])

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

  const shown = openId ? (past.data?.messages ?? []) : messages

  return (
    <div className={classes.shell}>
      <ChatHistory
        openId={openId}
        onOpen={setOpenId}
        onNew={() => reset.mutate()}
        newBusy={reset.isPending}
        newDisabled={messages.length === 0 || busy}
      />
      <div className={classes.chat}>
        <header className={classes.header}>
          <Mascot size={40} />
          <div style={{ flex: 1, minWidth: 0 }}>
            <Text fw={600}>AI-buddy</Text>
            <Text size="xs" c="ink.5">
              знает твой этап, команду и права доступа
            </Text>
          </div>
          {stage && !openId && (
            <StatusTag bg="#BADAFB">
              Этап {stage.code} · {stage.title}
            </StatusTag>
          )}
          {openId && (
            <Button
              variant="outline"
              color="ink"
              leftSection={<IconArrowLeft size={16} />}
              onClick={() => setOpenId(null)}
            >
              К текущему диалогу
            </Button>
          )}
        </header>

        <div className={classes.scroll}>
          <Stack gap={20} className={classes.column} aria-live="polite">
            {history.isPending && (
              <Text c="ink.5" ta="center" py={48}>
                Загружаем переписку…
              </Text>
            )}
            {history.isError && (
              <Alert color="yellow" variant="light">
                Не удалось загрузить прошлую переписку. Новые сообщения работают как обычно.
              </Alert>
            )}
            {openId && past.isPending && (
              <Text c="ink.5" ta="center" py={48}>
                Загружаем диалог…
              </Text>
            )}
            {openId && past.isError && (
              <Alert color="red" variant="light" role="alert">
                {errorText(past.error)}
              </Alert>
            )}
            {!openId && messages.length === 0 && !send.isPending && !history.isPending && (
              <Stack align="center" gap="sm" py={48} className="ab-pop">
                <Mascot size={140} glow float />
                <Text fz={26} fw={500} ta="center" style={{ letterSpacing: '-0.02em' }}>
                  Привет{user ? `, ${user.name}` : ''}! С чего начнем?
                </Text>
                <Text c="ink.5" ta="center" maw={460}>
                  Спроси о процессах и инструментах или попроси оформить заявку. Я покажу, откуда
                  взял ответ.
                </Text>
              </Stack>
            )}
            {shown.map((m) => (
              <ChatMessageView
                key={m.id}
                message={m}
                busy={act.isPending && act.variables?.actionId === m.confirm?.actionId}
                onConfirm={(actionId) => act.mutate({ actionId, ok: true })}
                onCancel={(actionId) => act.mutate({ actionId, ok: false })}
                onEdit={() => {
                  // правку backend понимает текстом: «сделай только чтение», «перенеси на 25 ноября»
                  setText('Измени в заявке: ')
                  requestAnimationFrame(() => inputRef.current?.focus())
                }}
                onAskMentor={(q) => navigate({ to: '/mentor', search: { q } })}
              />
            ))}
            {!openId && send.isPending && <ThinkingBubble />}
            {lastFailed && !send.isPending && (
              <Alert
                color="red"
                variant="light"
                role="alert"
                icon={<Mascot size={24} mood="sad" />}
              >
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
            {reset.isError && (
              <Alert color="red" variant="light" role="alert">
                {errorText(reset.error)}
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

        {openId ? (
          <div className={classes.pastBanner}>
            <Text size="sm" c="ink.5">
              Это прошлый диалог, он только для чтения.
            </Text>
            <Button color="ink" onClick={() => setOpenId(null)}>
              Вернуться к текущему
            </Button>
          </div>
        ) : (
          <div className={classes.composer}>
            <Stack gap={10} className={classes.column}>
              <Group gap={8}>
                {hints.map((s) => (
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
                  ref={inputRef}
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
                AI-buddy может ошибаться. Важное уточняй у наставника. Данные демо-версии
                синтетические.
              </Text>
            </Stack>
          </div>
        )}
      </div>
    </div>
  )
}
