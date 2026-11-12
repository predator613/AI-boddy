import { Button, Checkbox, Popover, Text, UnstyledButton } from '@mantine/core'
import { IconBell, IconChevronDown } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { Mascot } from '../components/Mascot'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { useInternGoals } from '../hooks/useGoals'
import { useThreads } from '../hooks/useMentor'
import { useInternPath, useInterns, useSetInternTaskStatus } from '../hooks/usePath'
import { useChecklist, useFeed, useMarkFeedRead, useToggleChecklist } from '../hooks/useStaff'
import type { TChecklistItem } from '../schemas/staff'
import type { TGoalsView } from '../schemas/goals'
import type { TInternSummary, TPathView, TTaskView } from '../schemas/path'
import type { TThreadSummary } from '../schemas/mentor'
import { useAuthStore } from '../store/authStore'
import css from './MentorHomePage.module.css'
import { DAY, DOT, buildTodos, glass, type Todo } from '../lib/mentorHome'

// Главная наставника: стажеры – стеклянные полосы. На полосе метки «что от тебя нужно»,
// по клику полоса раскрывается: дела с одной кнопкой, задачи этапа, цели.

const GLASS: [string, string][] = [
  ['#BADAFB', '#8EC7F7'],
  ['#DFCDF3', '#C3A6EC'],
  ['#C9FFBF', '#9DF5A8'],
  ['#F7C1C5', '#F29AA2'],
]

export function MentorHomePage() {
  const user = useAuthStore((s) => s.user)
  const interns = useInterns()
  const threads = useThreads()
  const [open, setOpen] = useState<string | null>(null)
  const openId = open ?? interns.data?.[0]?.id ?? null
  return (
    <Page>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 28, flex: 1, minWidth: 0 }}>
        <PageHeader
          overline={`Наставник · ${user?.teamName ?? ''}`}
          title={
            <>
              Привет, <Marker color="#965EEB">{user?.name ?? 'наставник'}!</Marker>
            </>
          }
          right={<FeedBell />}
        />
        <section
          aria-labelledby="my-interns"
          style={{ display: 'flex', flexDirection: 'column', gap: 14 }}
        >
          <Text
            component="h2"
            id="my-interns"
            fz={22}
            fw={500}
            m={0}
            style={{ letterSpacing: '-0.02em' }}
          >
            Мои стажеры
          </Text>
          {interns.isPending && <LoadingBlocks label="Загрузка стажеров" />}
          {interns.isError && (
            <ErrorBlock error={interns.error} onRetry={() => interns.refetch()} />
          )}
          {interns.data?.length === 0 && (
            <Text>Пока нет стажеров. Когда в команду придет стажер, он появится здесь.</Text>
          )}
          {interns.data?.map((i, k) => (
            <InternRow
              key={i.id}
              intern={i}
              colors={GLASS[k % GLASS.length]}
              // пока переписку отдают моки, id стажера там свой – подстраховываемся именем
              thread={threads.data?.find((t) => t.internId === i.id || t.internName === i.name)}
              open={openId === i.id}
              onToggle={() => setOpen(openId === i.id ? '' : i.id)}
            />
          ))}
        </section>
      </div>
    </Page>
  )
}

const BAR = { done: '#04E061', current: '#00AAFF', todo: 'rgba(255,255,255,.75)' } as const

interface RowProps {
  intern: TInternSummary
  colors: [string, string]
  thread: TThreadSummary | undefined
  open: boolean
  onToggle: () => void
}

function InternRow({ intern: i, colors, thread, open, onToggle }: RowProps) {
  const path = useInternPath(i.id)
  const goals = useInternGoals(i.id)
  const checklist = useChecklist(i.id)
  const todos = buildTodos(thread, path.data, goals.data)
  const nowItems = (checklist.data?.items ?? []).filter((c) => !c.done && c.fromWeek <= i.week + 1)
  const chips = [
    ...todos.map((t) => ({ text: t.chip, dot: DOT[t.kind] })),
    ...(nowItems.length ? [{ text: `чек-лист: ${nowItems.length}`, dot: '#141414' }] : []),
  ]
  const summary = chips.length ? chips.map((c) => c.text).join(', ') : 'все спокойно'
  return (
    <div className="ab-rise" style={{ display: 'flex', flexDirection: 'column' }}>
      <UnstyledButton
        className={css.row}
        onClick={onToggle}
        aria-expanded={open}
        aria-label={`${i.name}: ${summary}`}
        style={{ background: glass(colors), outline: open ? '2px solid #141414' : 'none' }}
      >
        <span className={css.avatar}>{i.name.slice(0, 1)}</span>
        <span className={css.who}>
          <span className={css.name}>{i.name}</span>
          <span style={{ fontSize: 12 }}>неделя {i.week}</span>
        </span>
        <span className={css.progress}>
          <span style={{ display: 'flex', alignItems: 'baseline', gap: 10 }}>
            <span className={css.pct}>{i.progress.percent}%</span>
            {i.currentStage && (
              <span style={{ fontSize: 12 }}>
                этап {i.currentStageCode} · {i.currentStage}
              </span>
            )}
          </span>
          <span style={{ display: 'flex', gap: 3 }} aria-hidden="true">
            {(path.data?.stages ?? []).map((s) => (
              <span
                key={s.id}
                style={{ flex: 1, height: 6, borderRadius: 3, background: BAR[s.status] }}
              />
            ))}
          </span>
        </span>
        <span className={css.chips} aria-hidden="true">
          {(chips.length ? chips : [{ text: 'все спокойно', dot: '#04E061' }]).map((c) => (
            <span key={c.text} className={css.chip}>
              <span className={css.dot} style={{ background: c.dot }} />
              {c.text}
            </span>
          ))}
        </span>
        <IconChevronDown
          size={22}
          aria-hidden="true"
          style={{
            flex: 'none',
            transform: open ? 'rotate(180deg)' : 'none',
            transition: 'transform .25s ease',
          }}
        />
      </UnstyledButton>
      {open && (
        <InternPanel
          intern={i}
          todos={todos}
          path={path.data}
          goals={goals.data}
          checklist={checklist.data?.items ?? []}
          chatId={thread?.internId ?? i.id}
        />
      )}
    </div>
  )
}

const STATUS: Record<
  TTaskView['status'],
  { label: string; bg: string; next: TTaskView['status'] }
> = {
  todo: { label: 'впереди', bg: '#F4F4F4', next: 'in_progress' },
  in_progress: { label: 'в работе', bg: '#BADAFB', next: 'done' },
  waiting: { label: 'на согласовании', bg: '#BADAFB', next: 'done' },
  done: { label: 'готово', bg: '#C9FFBF', next: 'todo' },
}

interface PanelProps {
  intern: TInternSummary
  todos: Todo[]
  path: TPathView | undefined
  goals: TGoalsView | undefined
  checklist: TChecklistItem[]
  /** id переписки со стажером в «Сообщениях» */
  chatId: string
}

function InternPanel({ intern: i, todos, path, goals, checklist, chatId }: PanelProps) {
  const setStatus = useSetInternTaskStatus(i.id)
  const toggle = useToggleChecklist(i.id)
  // Только что отмеченный пункт еще секунду виден с галочкой, потом уходит из списка.
  const [justDone, setJustDone] = useState<string[]>([])
  const mark = (c: TChecklistItem, done: boolean) => {
    if (done) {
      setJustDone((l) => [...l, c.id])
      setTimeout(() => setJustDone((l) => l.filter((x) => x !== c.id)), 1200)
    }
    toggle.mutate({ itemId: c.id, done })
  }
  // Сейчас – до трех пунктов до следующей недели стажера; остальное – в меню «Чек-лист».
  const now = checklist
    .filter((c) => (!c.done || justDone.includes(c.id)) && c.fromWeek <= i.week + 1)
    .slice(0, 3)
  const doneCount = checklist.filter((c) => c.done).length
  const check = (c: TChecklistItem, compact = false) => (
    <Checkbox
      key={c.id}
      color="green.6"
      size={compact ? 'xs' : 'sm'}
      checked={c.done}
      onChange={(e) => mark(c, e.currentTarget.checked)}
      label={
        compact ? (
          <span style={{ fontSize: 13, color: c.done ? '#5C5C5C' : '#141414' }}>{c.title}</span>
        ) : (
          <span style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <span
              style={{
                fontSize: 14,
                color: c.done ? '#5C5C5C' : '#141414',
                textDecoration: c.done ? 'line-through' : 'none',
              }}
            >
              {c.title}
            </span>
            <span style={{ fontSize: 12, color: '#5C5C5C' }}>{c.hint}</span>
          </span>
        )
      }
    />
  )
  const stage = path?.stages.find((s) => s.status === 'current') ?? null
  return (
    <div className={`ab-pop ${css.panel}`} role="region" aria-label={`${i.name}: подробности`}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        <div className={css.label}>Что сделать сейчас</div>
        {todos.slice(0, 3).map((t) => (
          <div key={t.title} className={css.todoRow}>
            <span className={css.dot} style={{ background: DOT[t.kind], marginTop: 6 }} />
            <span
              style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 2 }}
            >
              <Text fw={500} fz={14}>
                {t.title}
              </Text>
              <Text fz={12} c="ink.5" lineClamp={1}>
                {t.text}
              </Text>
            </span>
            {t.cta === 'chat' && (
              <Button
                renderRoot={(p) => <Link {...p} to="/inbox" search={{ intern: chatId }} />}
                size="xs"
                color="ink"
              >
                Ответить
              </Button>
            )}
            {t.cta === 'path' && (
              <Button
                renderRoot={(p) => (
                  <Link {...p} to="/interns/$internId" params={{ internId: i.id }} />
                )}
                size="xs"
                variant="outline"
                color="ink"
              >
                Открыть
              </Button>
            )}
          </div>
        ))}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {now.map((c) => check(c))}
          {now.length === 0 && todos.length === 0 && (
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <Mascot mood="happy" size={36} />
              <Text fz={14}>На этой неделе все сделано.</Text>
            </div>
          )}
        </div>
        <Popover position="bottom-start" width={320} shadow="md" radius={12}>
          <Popover.Target>
            <UnstyledButton fz={13} c="ink.5" style={{ alignSelf: 'flex-start' }}>
              Чек-лист наставника · {doneCount} из {checklist.length}
            </UnstyledButton>
          </Popover.Target>
          <Popover.Dropdown p={14}>
            <Text fz={12} c="ink.5" mb={8}>
              Сними галочку, чтобы вернуть пункт
            </Text>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {checklist.map((c) => check(c, true))}
            </div>
          </Popover.Dropdown>
        </Popover>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div>
          <div className={css.label}>
            Сейчас{stage ? ` · этап ${stage.code} ${stage.title}` : ''}
          </div>
          {stage?.tasks.map((t) => {
            const s = STATUS[t.status]
            return (
              <div key={t.id} className={css.task}>
                <Text fz={14} style={{ flex: 1 }} c={t.status === 'done' ? 'ink.5' : 'ink.6'}>
                  {t.title}
                </Text>
                <UnstyledButton
                  className={css.status}
                  style={{ background: s.bg }}
                  disabled={setStatus.isPending}
                  aria-label={`Статус «${t.title}»: ${s.label}. Нажми, чтобы сменить`}
                  onClick={() => setStatus.mutate({ taskId: t.id, status: s.next })}
                >
                  {s.label}
                </UnstyledButton>
              </div>
            )
          })}
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
            <span className={css.label}>Цели</span>
            <Text fz={12} c="ink.5">
              {goals?.agreed ? 'согласованы' : 'еще не согласованы'}
            </Text>
          </div>
          {goals?.goals.map((g) => (
            <div key={g.id} className={css.goal}>
              <span className={css.ellipsis}>{g.title}</span>
              <span className={css.track}>
                <span
                  style={{
                    width: `${g.progressPct}%`,
                    background: g.progressPct === 100 ? '#04E061' : '#00AAFF',
                  }}
                />
              </span>
              <span style={{ textAlign: 'right', fontWeight: 600 }}>{g.progressPct}%</span>
            </div>
          ))}
          {goals && goals.goals.length === 0 && (
            <Text fz={13} c="ink.5">
              Появятся после встречи с руководителем.
            </Text>
          )}
        </div>
        <div style={{ display: 'flex', gap: 8, marginTop: 'auto' }}>
          <Button
            renderRoot={(p) => <Link {...p} to="/inbox" search={{ intern: chatId }} />}
            variant="outline"
            color="ink"
            style={{ flex: 1 }}
          >
            Написать
          </Button>
          <Button
            renderRoot={(p) => <Link {...p} to="/interns/$internId" params={{ internId: i.id }} />}
            color="ink"
            style={{ flex: 1 }}
          >
            Весь путь
          </Button>
        </div>
      </div>
    </div>
  )
}

const ACTOR_BG = {
  intern: '#BADAFB',
  mentor: '#DFCDF3',
  lead: '#BADAFB',
  buddy: '#C9FFBF',
  other: '#F4F4F4',
}

function timeLabel(iso: string, now = new Date()) {
  const d = new Date(iso)
  const hm = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  if (d.toDateString() === now.toDateString()) return `сегодня, ${hm}`
  const y = new Date(now.getTime() - DAY)
  if (d.toDateString() === y.toDateString()) return `вчера, ${hm}`
  return d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })
}

/** Колокольчик: кто что поменял – стажеры, руководитель, AI-buddy. */
function FeedBell() {
  const feed = useFeed()
  const read = useMarkFeedRead()
  const unread = feed.data?.unread ?? 0
  return (
    <Popover position="bottom-end" width={380} shadow="md" radius={16}>
      <Popover.Target>
        <UnstyledButton className={css.bell} aria-label={`Уведомления, новых: ${unread}`}>
          <IconBell size={24} stroke={1.8} aria-hidden="true" />
          {unread > 0 && <span className={css.badge}>{unread}</span>}
        </UnstyledButton>
      </Popover.Target>
      <Popover.Dropdown p={16}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: 6,
          }}
        >
          <Text fw={600}>Что изменилось</Text>
          {unread > 0 && (
            <UnstyledButton fz={12} c="ink.5" onClick={() => read.mutate()}>
              Прочитать все
            </UnstyledButton>
          )}
        </div>
        {feed.isError && <ErrorBlock error={feed.error} onRetry={() => feed.refetch()} />}
        {feed.data?.items.length === 0 && <Text fz={14}>Пока ничего нового.</Text>}
        <ul style={{ listStyle: 'none', margin: 0, padding: 0 }}>
          {feed.data?.items.map((f) => (
            <li key={f.id} className={css.feedItem}>
              <span className={css.actor} style={{ background: ACTOR_BG[f.actor.kind] }}>
                {f.actor.kind === 'buddy' ? 'AI' : f.actor.name.slice(0, 1)}
              </span>
              <span style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Text fz={14} fw={f.read ? 400 : 500} style={{ lineHeight: 1.35 }}>
                  {f.text}
                </Text>
                <Text fz={12} c="ink.5">
                  {timeLabel(f.at)}
                </Text>
              </span>
              {!f.read && (
                <span
                  className={css.dot}
                  style={{ background: '#965EEB', marginTop: 6 }}
                  aria-label="новое"
                />
              )}
            </li>
          ))}
        </ul>
      </Popover.Dropdown>
    </Popover>
  )
}
