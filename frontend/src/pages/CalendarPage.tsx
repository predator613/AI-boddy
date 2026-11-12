import {
  ActionIcon,
  Alert,
  Anchor,
  Button,
  Checkbox,
  Group,
  Modal,
  NativeSelect,
  Paper,
  Stack,
  Text,
  TextInput,
  Textarea,
  UnstyledButton,
} from '@mantine/core'
import { IconChevronLeft, IconChevronRight, IconPlus } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import {
  useAddEntry,
  useAnswerEvent,
  useCalendar,
  useDeleteEntry,
  useNotes,
  useSaveNote,
} from '../hooks/useCalendar'
import { useMyPath } from '../hooks/usePath'
import { errorText } from '../lib/errors'
import { addDays, mondayOf } from '../lib/mentorUi'
import { msk, mskIso, ymd } from '../lib/msk'
import { EntryInput, type TCalendarEvent, type TEventKind } from '../schemas/calendar'
import { toast } from '../store/toastStore'

const HOURS = [10, 11, 12, 13, 14, 15, 16, 17]
const ROW = 56
const DAYS = ['пн', 'вт', 'ср', 'чт', 'пт']
const DAYS_FULL = ['понедельник', 'вторник', 'среда', 'четверг', 'пятница']

const KIND: Record<TEventKind, { label: string; bg: string; own?: boolean }> = {
  mentor: { label: 'с наставником', bg: '#DFCDF3' },
  team: { label: 'команда', bg: '#C9FFBF' },
  lead: { label: 'руководитель', bg: '#F7C1C5' },
  intro: { label: 'знакомство', bg: '#BADAFB' },
  focus: { label: 'фокус-время', bg: '#FFFFFF', own: true },
  study: { label: 'учеба', bg: '#FFFFFF', own: true },
  personal: { label: 'личное', bg: '#FFFFFF', own: true },
}

const STATUS: Record<TCalendarEvent['status'], string> = {
  pending: 'ждет ответа',
  accepted: 'ты придешь',
  move_requested: 'запрошен перенос',
  own: 'своя запись',
}

const REASONS = ['Пересекается с другой встречей', 'Не успею подготовиться', 'Удобнее другое время']

const hm = (d: Date) =>
  d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit', timeZone: 'Europe/Moscow' })
/** Раскладка дня: пересекающиеся события делят ширину колонки поровну. */
function layoutDay(list: TCalendarEvent[]) {
  const sorted = [...list].sort((a, b) => a.startsAt.localeCompare(b.startsAt))
  const out: { event: TCalendarEvent; col: number; cols: number }[] = []
  let group: typeof out = []
  let groupEnd = 0
  const flush = () => {
    const cols = Math.max(1, ...group.map((g) => g.col + 1))
    group.forEach((g) => out.push({ ...g, cols }))
    group = []
  }
  for (const e of sorted) {
    const start = new Date(e.startsAt).getTime()
    const end = start + e.durationMin * 60_000
    if (group.length && start >= groupEnd) flush()
    const busy = new Set(group.filter((g) => endOf(g.event).getTime() > start).map((g) => g.col))
    let col = 0
    while (busy.has(col)) col++
    group.push({ event: e, col, cols: 1 })
    groupEnd = Math.max(groupEnd, end)
  }
  if (group.length) flush()
  return out
}

const endOf = (e: TCalendarEvent) =>
  new Date(new Date(e.startsAt).getTime() + e.durationMin * 60_000)

interface CalendarPageProps {
  now?: Date
}

// Календарь стажера: встречи, которые назначил наставник или команда, свои записи и сроки задач.
export function CalendarPage({ now: nowProp }: CalendarPageProps) {
  const [now] = useState(() => nowProp ?? new Date())
  const [monday, setMonday] = useState(() => mondayOf(now))
  const from = monday.toISOString()
  const to = addDays(monday, 7).toISOString()
  const cal = useCalendar(from, to)
  const path = useMyPath()
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [adding, setAdding] = useState(false)
  const events = cal.data?.events ?? []
  const selected =
    events.find((e) => e.id === selectedId) ?? events.find((e) => e.status === 'pending') ?? null
  const isThisWeek = mondayOf(now).getTime() === monday.getTime()

  const deadlines = (path.data?.stages ?? [])
    .flatMap((s) => s.tasks.map((t) => ({ ...t, stageCode: s.code })))
    .filter((t) => t.dueDate && t.status !== 'done')
    .sort((a, b) => a.dueDate!.localeCompare(b.dueDate!))
  const weekDeadlines = (k: number) => {
    const day = ymd(addDays(monday, k))
    return deadlines.filter((t) => t.dueDate!.slice(0, 10) === day)
  }

  const nowMsk = msk(now)
  const nowTop = (nowMsk.hour + nowMsk.minute / 60 - HOURS[0]!) * ROW

  return (
    <Page maxWidth={1400}>
      <Stack gap="lg" style={{ flex: 1, minWidth: 0 }}>
        <PageHeader
          overline="Календарь"
          title={
            <>
              Мои <Marker color="#DFCDF3">встречи</Marker>
            </>
          }
          right={
            <Group gap={8}>
              <ActionIcon
                variant="white"
                color="ink"
                size={44}
                radius="xl"
                aria-label="Прошлая неделя"
                onClick={() => setMonday(addDays(monday, -7))}
              >
                <IconChevronLeft size={18} />
              </ActionIcon>
              <Button
                variant="white"
                color="ink"
                radius="xl"
                h={44}
                onClick={() => setMonday(mondayOf(now))}
                disabled={isThisWeek}
              >
                Эта неделя
              </Button>
              <ActionIcon
                variant="white"
                color="ink"
                size={44}
                radius="xl"
                aria-label="Следующая неделя"
                onClick={() => setMonday(addDays(monday, 7))}
              >
                <IconChevronRight size={18} />
              </ActionIcon>
              <Button
                color="ink"
                radius="xl"
                h={44}
                leftSection={<IconPlus size={16} />}
                onClick={() => setAdding(true)}
              >
                Добавить запись
              </Button>
            </Group>
          }
        />
        {cal.isPending && <LoadingBlocks label="Загрузка календаря" />}
        {cal.isError && <ErrorBlock error={cal.error} onRetry={() => cal.refetch()} />}
        {cal.data && (
          <Group align="flex-start" gap="lg" wrap="wrap">
            <Paper
              radius={20}
              p="md"
              style={{ flex: '999 1 640px', minWidth: 0, overflowX: 'auto' }}
            >
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: '52px repeat(5, minmax(120px, 1fr))',
                  minWidth: 680,
                }}
              >
                <div />
                {DAYS.map((d, i) => {
                  const day = addDays(monday, i)
                  const today = ymd(day) === nowMsk.ymd
                  return (
                    <div
                      key={d}
                      style={{
                        padding: '4px 8px 10px',
                        borderLeft: '1px solid #F4F4F4',
                        display: 'flex',
                        flexDirection: 'column',
                        gap: 6,
                      }}
                    >
                      <span style={{ display: 'flex', alignItems: 'baseline', gap: 6 }}>
                        <Text size="xs" c="ink.5" tt="uppercase">
                          {d}
                        </Text>
                        <span
                          style={{
                            fontSize: 22,
                            fontWeight: 500,
                            lineHeight: 1,
                            background: today ? '#04E061' : 'transparent',
                            padding: '2px 6px',
                            borderRadius: 6,
                          }}
                        >
                          {day.getDate()}
                        </span>
                      </span>
                      <span
                        style={{ minHeight: 24, display: 'flex', flexDirection: 'column', gap: 4 }}
                      >
                        {weekDeadlines(i).map((t) => (
                          <span
                            key={t.id}
                            title={t.title}
                            style={{
                              fontSize: 11,
                              fontWeight: 500,
                              background: '#FFF3C4',
                              borderLeft: '3px solid #141414',
                              padding: '3px 6px',
                              whiteSpace: 'nowrap',
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                            }}
                          >
                            срок: {t.title}
                          </span>
                        ))}
                      </span>
                    </div>
                  )
                })}
                <div>
                  {HOURS.map((h) => (
                    <div
                      key={h}
                      style={{
                        height: ROW,
                        fontSize: 11,
                        color: '#5C5C5C',
                        textAlign: 'right',
                        paddingRight: 10,
                        boxSizing: 'border-box',
                        transform: 'translateY(-6px)',
                      }}
                    >
                      {h}:00
                    </div>
                  ))}
                </div>
                {DAYS.map((d, i) => {
                  const day = addDays(monday, i)
                  const today = ymd(day) === nowMsk.ymd
                  return (
                    <div
                      key={d}
                      style={{
                        position: 'relative',
                        height: ROW * HOURS.length,
                        borderLeft: '1px solid #F4F4F4',
                        background: today ? '#FBFFFA' : undefined,
                        backgroundImage: 'linear-gradient(#F0F0F0 1px, transparent 1px)',
                        backgroundSize: `100% ${ROW}px`,
                      }}
                    >
                      {today && nowTop > 0 && nowTop < ROW * HOURS.length && (
                        <div
                          aria-hidden="true"
                          style={{
                            position: 'absolute',
                            left: 0,
                            right: 0,
                            top: nowTop,
                            height: 2,
                            background: '#FF4053',
                            zIndex: 2,
                          }}
                        />
                      )}
                      {layoutDay(events.filter((e) => msk(e.startsAt).ymd === ymd(day))).map(
                        ({ event: e, col, cols }) => {
                          const m = msk(e.startsAt)
                          const s = new Date(e.startsAt)
                          const top = (m.hour + m.minute / 60 - HOURS[0]!) * ROW
                          const past = endOf(e) <= now
                          const k = KIND[e.kind]
                          const badge =
                            e.status === 'pending' && !past
                              ? 'ответь'
                              : e.status === 'move_requested'
                                ? 'перенос?'
                                : null
                          // короткая встреча (меньше часа) – одна строка, иначе текст вылезает за слот
                          const compact = e.durationMin < 60
                          return (
                            <UnstyledButton
                              key={e.id}
                              className="ab-pop"
                              onClick={() => setSelectedId(e.id)}
                              aria-pressed={selected?.id === e.id}
                              aria-label={`${e.title}, ${DAYS[i]} ${hm(s)}`}
                              title={`${e.title}, ${hm(s)}–${hm(endOf(e))}`}
                              style={{
                                position: 'absolute',
                                // пересекающиеся встречи встают рядом, а не друг на друга
                                left: `calc(${(col / cols) * 100}% + 4px)`,
                                width: `calc(${100 / cols}% - 8px)`,
                                top: Math.max(top, 0) + 2,
                                height: Math.max((e.durationMin / 60) * ROW - 4, 24),
                                background: k.bg,
                                border: k.own ? '1px dashed #141414' : 0,
                                boxSizing: 'border-box',
                                borderRadius: compact ? 8 : 10,
                                outline: selected?.id === e.id ? '2px solid #141414' : 'none',
                                outlineOffset: -2,
                                padding: compact ? '0 8px' : '6px 9px',
                                overflow: 'hidden',
                                opacity: past ? 0.55 : 1,
                                display: 'flex',
                                flexDirection: compact ? 'row' : 'column',
                                alignItems: compact ? 'center' : 'stretch',
                                gap: compact ? 6 : 2,
                              }}
                            >
                              <Text
                                size="xs"
                                fw={600}
                                lh={1.2}
                                style={
                                  compact
                                    ? {
                                        flex: 1,
                                        minWidth: 0,
                                        whiteSpace: 'nowrap',
                                        overflow: 'hidden',
                                        textOverflow: 'ellipsis',
                                      }
                                    : {
                                        display: '-webkit-box',
                                        WebkitLineClamp: e.durationMin >= 90 ? 3 : 2,
                                        WebkitBoxOrient: 'vertical',
                                        overflow: 'hidden',
                                      }
                                }
                              >
                                {e.title}
                              </Text>
                              <Text size="11px" style={{ flex: 'none' }}>
                                {compact ? hm(s) : `${hm(s)}–${hm(endOf(e))}`}
                              </Text>
                              {badge && e.durationMin >= 90 && (
                                <span
                                  style={{
                                    fontSize: 10,
                                    fontWeight: 600,
                                    textTransform: 'uppercase',
                                    background: '#FFFFFF',
                                    padding: '1px 5px',
                                    alignSelf: 'flex-start',
                                  }}
                                >
                                  {badge}
                                </span>
                              )}
                              {badge && e.durationMin < 90 && (
                                <span
                                  aria-hidden="true"
                                  style={{
                                    position: 'absolute',
                                    top: compact ? '50%' : 7,
                                    right: 7,
                                    marginTop: compact ? -4 : 0,
                                    width: 8,
                                    height: 8,
                                    borderRadius: 4,
                                    background: e.status === 'pending' ? '#965EEB' : '#FF4053',
                                  }}
                                />
                              )}
                            </UnstyledButton>
                          )
                        },
                      )}
                    </div>
                  )
                })}
              </div>
              <Group gap={16} mt={14}>
                {(['mentor', 'team', 'lead', 'intro'] as const).map((k) => (
                  <Group key={k} gap={6}>
                    <span
                      style={{ width: 12, height: 12, borderRadius: 3, background: KIND[k].bg }}
                    />
                    <Text size="xs" c="ink.5">
                      {KIND[k].label}
                    </Text>
                  </Group>
                ))}
                <Group gap={6}>
                  <span
                    style={{
                      width: 12,
                      height: 12,
                      borderRadius: 3,
                      border: '1px dashed #141414',
                      boxSizing: 'border-box',
                    }}
                  />
                  <Text size="xs" c="ink.5">
                    мои записи
                  </Text>
                </Group>
                <Group gap={6}>
                  <span
                    style={{
                      width: 12,
                      height: 12,
                      borderLeft: '3px solid #141414',
                      background: '#FFF3C4',
                    }}
                  />
                  <Text size="xs" c="ink.5">
                    сроки задач
                  </Text>
                </Group>
              </Group>
            </Paper>

            <Stack gap="md" style={{ flex: '1 1 300px', maxWidth: 360, minWidth: 0 }}>
              {selected ? (
                <EventDetails
                  key={selected.id}
                  event={selected}
                  now={now}
                  onDeleted={() => setSelectedId(null)}
                  onPin={() => setSelectedId(selected.id)}
                />
              ) : (
                <Paper radius={20} p="lg">
                  <Text size="sm" c="ink.5">
                    На этой неделе встреч нет. Наставник добавит их в своем кабинете, а тебе придет
                    уведомление. Свое время можно занять кнопкой «Добавить запись».
                  </Text>
                </Paper>
              )}
              <Paper radius={20} p="lg">
                <Text component="h2" fz={16} fw={600} m={0} mb={8}>
                  Сроки задач
                </Text>
                {deadlines.length === 0 && (
                  <Text size="sm" c="ink.5">
                    Сроков впереди нет
                  </Text>
                )}
                {deadlines.slice(0, 4).map((t) => {
                  const d = new Date(t.dueDate!)
                  return (
                    <Group
                      key={t.id}
                      gap={12}
                      wrap="nowrap"
                      py={10}
                      style={{ borderTop: '1px solid #EDEDED' }}
                    >
                      <span style={{ width: 40, textAlign: 'center', lineHeight: 1, flex: 'none' }}>
                        <Text size="11px" c="ink.5" tt="uppercase">
                          {d.toLocaleDateString('ru-RU', { weekday: 'short' })}
                        </Text>
                        <Text fz={20} fw={500}>
                          {d.getDate()}
                        </Text>
                      </span>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <Text size="sm" fw={500}>
                          {t.title}
                        </Text>
                        <Text size="xs" c="ink.5">
                          этап {t.stageCode}
                        </Text>
                      </div>
                    </Group>
                  )
                })}
              </Paper>
            </Stack>
          </Group>
        )}
        <Text size="xs" c="ink.5">
          Встречи назначает наставник в своем кабинете, изменения приходят сюда и в чат с
          наставником. Данные демо-версии синтетические.
        </Text>
      </Stack>
      {adding && (
        <EntryModal
          monday={monday}
          now={now}
          onClose={() => setAdding(false)}
          onCreated={(id) => {
            setAdding(false)
            setSelectedId(id)
          }}
        />
      )}
    </Page>
  )
}

function EventDetails({
  event: e,
  now,
  onDeleted,
  onPin,
}: {
  event: TCalendarEvent
  now: Date
  onDeleted: () => void
  /** Оставить карточку открытой после ответа (иначе выбор перейдет к следующей встрече). */
  onPin: () => void
}) {
  const answer = useAnswerEvent()
  const remove = useDeleteEntry()
  const [moving, setMoving] = useState(false)
  const s = new Date(e.startsAt)
  const past = endOf(e) <= now
  const k = KIND[e.kind]
  const day = DAYS_FULL[msk(e.startsAt).weekday] ?? ''
  const whoTo = e.kind === 'mentor' ? 'наставнику' : 'организатору'
  return (
    <Paper radius={20} p="lg" className="ab-pop">
      <Stack gap="sm">
        <StatusTag bg={k.own ? '#F4F4F4' : k.bg} style={{ alignSelf: 'flex-start' }}>
          {k.label}
        </StatusTag>
        <Text component="h2" fz={24} fw={500} lh={1.15} m={0}>
          {e.title}
        </Text>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: '80px 1fr',
            gap: '8px 12px',
            fontSize: 14,
          }}
        >
          <Text size="sm" c="ink.5">
            Когда
          </Text>
          <Text size="sm">
            {day},{' '}
            {s.toLocaleDateString('ru-RU', {
              day: 'numeric',
              month: 'long',
              timeZone: 'Europe/Moscow',
            })}
            , {hm(s)}–{hm(endOf(e))}
          </Text>
          <Text size="sm" c="ink.5">
            С кем
          </Text>
          <Text size="sm">{e.withName}</Text>
          <Text size="sm" c="ink.5">
            Где
          </Text>
          <Text size="sm">{e.place}</Text>
          <Text size="sm" c="ink.5">
            Статус
          </Text>
          <Text size="sm" fw={500}>
            {past
              ? 'прошла'
              : e.status === 'own'
                ? e.shared
                  ? 'видит наставник'
                  : 'только у тебя'
                : STATUS[e.status]}
          </Text>
        </div>
        {e.note && (
          <div style={{ background: '#F4F4F4', borderRadius: 12, padding: '12px 14px' }}>
            <Text size="sm" lh={1.4}>
              {e.note}
            </Text>
          </div>
        )}
        {answer.isError && (
          <Alert color="red" variant="light" role="alert">
            {errorText(answer.error)}
          </Alert>
        )}
        {e.status === 'pending' && !past && !moving && (
          <Group gap={8}>
            <Button
              color="ink"
              loading={answer.isPending}
              onClick={() => {
                onPin()
                answer.mutate(
                  { id: e.id, answer: 'accept' },
                  { onSuccess: () => toast('Готово! Отметила, что ты придешь. Напомню за час') },
                )
              }}
            >
              Приду
            </Button>
            <Button
              variant="outline"
              color="ink"
              onClick={() => {
                onPin()
                setMoving(true)
              }}
            >
              Попросить перенести
            </Button>
          </Group>
        )}
        {moving && (
          <Stack gap={8} className="ab-pop">
            <Text size="sm" c="ink.5">
              Почему перенести? AI-buddy напишет {whoTo} за тебя
            </Text>
            <Group gap={6}>
              {REASONS.map((r) => (
                <Button
                  key={r}
                  size="xs"
                  radius="xl"
                  variant="default"
                  loading={answer.isPending && answer.variables?.reason === r}
                  onClick={() =>
                    answer.mutate(
                      { id: e.id, answer: 'move', reason: r },
                      {
                        onSuccess: () => {
                          setMoving(false)
                          toast(
                            `AI-buddy написал ${whoTo}: «${r}». Ответ придет в чат с наставником`,
                          )
                        },
                      },
                    )
                  }
                >
                  {r}
                </Button>
              ))}
            </Group>
          </Stack>
        )}
        {e.status === 'own' ? (
          <Button
            color="red.2"
            c="ink.9"
            style={{ alignSelf: 'flex-start' }}
            loading={remove.isPending}
            onClick={() =>
              remove.mutate(e.id, {
                onSuccess: () => {
                  toast('Запись удалена')
                  onDeleted()
                },
              })
            }
          >
            Удалить запись
          </Button>
        ) : (
          <>
            <MeetingNotes eventId={e.id} />
            <Anchor renderRoot={(p) => <Link {...p} to="/mentor" />} size="sm" fw={500} c="ink.9">
              Написать наставнику →
            </Anchor>
          </>
        )}
      </Stack>
    </Paper>
  )
}

/** Конспект встречи: свернут, пока не нужен. Виден только стажеру. */
function MeetingNotes({ eventId }: { eventId: string }) {
  const notes = useNotes()
  const save = useSaveNote()
  const saved = notes.data?.find((n) => n.eventId === eventId)?.text ?? ''
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState<string | null>(null)
  const text = draft ?? saved
  return (
    <div style={{ borderTop: '1px solid #EDEDED', paddingTop: 10 }}>
      <UnstyledButton
        fz="sm"
        fw={500}
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        style={{ display: 'flex', alignItems: 'center', gap: 6 }}
      >
        Конспект встречи
        <Text span fz="xs" c="ink.5">
          {saved ? '· есть заметки' : '· только для тебя'}
        </Text>
      </UnstyledButton>
      {open && (
        <Stack gap={8} mt={8} className="ab-pop">
          <Textarea
            aria-label="Конспект встречи"
            placeholder="Что узнала, о чем договорились, обратная связь"
            autosize
            minRows={3}
            maxRows={8}
            value={text}
            onChange={(e) => setDraft(e.currentTarget.value)}
          />
          <Button
            size="xs"
            color="ink"
            style={{ alignSelf: 'flex-start' }}
            disabled={draft === null || draft === saved}
            loading={save.isPending}
            onClick={() =>
              save.mutate(
                { eventId, text },
                { onSuccess: () => (setDraft(null), toast('Конспект сохранен')) },
              )
            }
          >
            Сохранить
          </Button>
        </Stack>
      )}
    </div>
  )
}

const TYPES = [
  { value: 'focus', label: 'Фокус-время' },
  { value: 'study', label: 'Учеба' },
  { value: 'personal', label: 'Личное' },
] as const

function EntryModal({
  monday,
  now,
  onClose,
  onCreated,
}: {
  monday: Date
  now: Date
  onClose: () => void
  onCreated: (id: string) => void
}) {
  const add = useAddEntry()
  const todayIdx = msk(now).weekday
  const [title, setTitle] = useState('')
  const [kind, setKind] = useState<(typeof TYPES)[number]['value']>('focus')
  const [day, setDay] = useState(
    String(mondayOf(now).getTime() === monday.getTime() && todayIdx < 5 ? todayIdx : 0),
  )
  const [time, setTime] = useState('16')
  const [dur, setDur] = useState('60')
  const [share, setShare] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const submit = () => {
    const input = {
      title,
      kind,
      startsAt: mskIso(addDays(monday, Number(day)), Number(time)),
      durationMin: Number(dur),
      shareWithMentor: share,
    }
    const parsed = EntryInput.safeParse(input)
    if (!parsed.success) return setError(parsed.error.issues[0]?.message ?? 'Проверь поля')
    add.mutate(input, {
      onSuccess: (ev) => {
        toast(
          share
            ? 'Запись добавлена. Наставник увидит, что это время занято'
            : 'Запись добавлена в твой календарь',
        )
        onCreated(ev.id)
      },
    })
  }

  const times = []
  for (let h = 10; h < 18; h += 0.5)
    times.push({ value: String(h), label: `${Math.floor(h)}:${h % 1 ? '30' : '00'}` })

  return (
    <Modal opened onClose={onClose} title="Новая запись" centered radius="lg">
      <Stack gap="sm">
        <TextInput
          label="Что планируешь"
          placeholder="Например, «Разобрать памятку по ревью»"
          value={title}
          onChange={(e) => (setTitle(e.currentTarget.value), setError(null))}
          data-autofocus
        />
        <div>
          <Text size="sm" fw={500} mb={6}>
            Тип
          </Text>
          <Group gap={6}>
            {TYPES.map((t) => (
              <Button
                key={t.value}
                size="xs"
                radius="xl"
                color="ink"
                variant={kind === t.value ? 'filled' : 'default'}
                aria-pressed={kind === t.value}
                onClick={() => setKind(t.value)}
              >
                {t.label}
              </Button>
            ))}
          </Group>
        </div>
        <Group grow align="flex-start">
          <NativeSelect
            label="День"
            value={day}
            onChange={(e) => (setDay(e.currentTarget.value), setError(null))}
            data={DAYS.map((d, i) => ({
              value: String(i),
              label: `${d} ${addDays(monday, i).getDate()}`,
            }))}
          />
          <NativeSelect
            label="Начало"
            value={time}
            onChange={(e) => (setTime(e.currentTarget.value), setError(null))}
            data={times}
          />
          <NativeSelect
            label="Длительность"
            value={dur}
            onChange={(e) => (setDur(e.currentTarget.value), setError(null))}
            data={[
              { value: '30', label: '30 мин' },
              { value: '60', label: '1 час' },
              { value: '90', label: '1,5 часа' },
              { value: '120', label: '2 часа' },
            ]}
          />
        </Group>
        <Checkbox
          color="green.6"
          label="Показать наставнику в его расписании"
          checked={share}
          onChange={(e) => setShare(e.currentTarget.checked)}
        />
        {(error || add.isError) && (
          <Alert color="red" variant="light" role="alert">
            {error ?? errorText(add.error)}
          </Alert>
        )}
        <Group justify="flex-end">
          <Button variant="default" onClick={onClose}>
            Отмена
          </Button>
          <Button color="ink" loading={add.isPending} onClick={submit}>
            Добавить
          </Button>
        </Group>
      </Stack>
    </Modal>
  )
}
