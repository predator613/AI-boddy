import {
  ActionIcon,
  Alert,
  Button,
  Checkbox,
  Group,
  Modal,
  NativeSelect,
  Paper,
  Stack,
  Text,
  TextInput,
  UnstyledButton,
} from '@mantine/core'
import { IconChevronLeft, IconChevronRight, IconPlus } from '@tabler/icons-react'
import { useState } from 'react'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import {
  useCancelMeeting,
  useCreateMentorMeeting,
  useMeetings,
  useMoveMeeting,
  useThreads,
} from '../hooks/useMentor'
import { errorText } from '../lib/errors'
import { addDays, mondayOf } from '../lib/mentorUi'
import type { TMentorMeeting } from '../schemas/mentor'
import { toast } from '../store/toastStore'

const HOURS = [10, 11, 12, 13, 14, 15, 16, 17]
const ROW = 52
const DAYS = ['пн', 'вт', 'ср', 'чт', 'пт']
const KIND = {
  one_on_one: { label: '1:1', bg: '#DFCDF3' },
  review: { label: 'разбор', bg: '#BADAFB' },
  team: { label: 'команда', bg: '#C9FFBF' },
} as const

const hm = (d: Date) => d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
const dm = (d: Date) => d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' })

// Расписание наставника: встречи со стажерами на неделю. Стажер получает уведомление о каждом изменении.
export function SchedulePage() {
  const [monday, setMonday] = useState(() => mondayOf(new Date()))
  const from = monday.toISOString()
  const to = addDays(monday, 7).toISOString()
  const meetings = useMeetings(from, to)
  const move = useMoveMeeting()
  const cancel = useCancelMeeting()
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const selected = meetings.data?.find((m) => m.id === selectedId) ?? null
  const [today] = useState(() => new Date().toDateString())

  const shift = (m: TMentorMeeting) => {
    const next = new Date(m.startsAt)
    next.setHours(Math.min(next.getHours() + 1, 17))
    move.mutate(
      { id: m.id, startsAt: next.toISOString() },
      {
        onSuccess: () =>
          toast(`${m.withName}: встреча перенесена на ${hm(next)}, уведомление отправлено`),
      },
    )
  }

  return (
    <Page maxWidth={1400}>
      <Stack gap="lg" style={{ flex: 1, minWidth: 0 }}>
        <PageHeader
          overline="Расписание"
          title={
            <>
              Встречи <Marker color="#DFCDF3">со стажерами</Marker>
            </>
          }
          right={
            <Group gap={8}>
              <ActionIcon
                variant="default"
                size={44}
                aria-label="Прошлая неделя"
                onClick={() => setMonday(addDays(monday, -7))}
              >
                <IconChevronLeft size={18} />
              </ActionIcon>
              <ActionIcon
                variant="default"
                size={44}
                aria-label="Следующая неделя"
                onClick={() => setMonday(addDays(monday, 7))}
              >
                <IconChevronRight size={18} />
              </ActionIcon>
              <Button
                color="ink"
                h={44}
                leftSection={<IconPlus size={18} />}
                onClick={() => setCreating(true)}
              >
                Встреча
              </Button>
            </Group>
          }
        />
        {meetings.isPending && <LoadingBlocks label="Загрузка расписания" />}
        {meetings.isError && (
          <ErrorBlock error={meetings.error} onRetry={() => meetings.refetch()} />
        )}
        {meetings.data && (
          <Group align="flex-start" gap="lg" wrap="wrap">
            <Paper
              radius="lg"
              p="md"
              style={{ flex: '999 1 560px', minWidth: 0, overflowX: 'auto' }}
            >
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: `48px repeat(5, minmax(110px, 1fr))`,
                  minWidth: 620,
                }}
              >
                <div />
                {DAYS.map((d, i) => {
                  const day = addDays(monday, i)
                  const isToday = day.toDateString() === today
                  return (
                    <div
                      key={d}
                      style={{
                        padding: '6px 8px 12px',
                        fontSize: 13,
                        borderLeft: '1px solid #F4F4F4',
                      }}
                    >
                      <span style={{ color: '#5C5C5C' }}>{d}</span>{' '}
                      <b
                        style={{
                          fontWeight: isToday ? 600 : 400,
                          background: isToday ? '#04E061' : 'transparent',
                          padding: '0 4px',
                        }}
                      >
                        {day.getDate()}
                      </b>
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
                        paddingRight: 8,
                      }}
                    >
                      {h}:00
                    </div>
                  ))}
                </div>
                {DAYS.map((d, i) => {
                  const day = addDays(monday, i).toDateString()
                  return (
                    <div
                      key={d}
                      style={{
                        position: 'relative',
                        height: ROW * HOURS.length,
                        borderLeft: '1px solid #F4F4F4',
                        backgroundImage: 'linear-gradient(#F4F4F4 1px, transparent 1px)',
                        backgroundSize: `100% ${ROW}px`,
                      }}
                    >
                      {meetings.data
                        .filter((m) => new Date(m.startsAt).toDateString() === day)
                        .map((m) => {
                          const s = new Date(m.startsAt)
                          const top = (s.getHours() + s.getMinutes() / 60 - HOURS[0]!) * ROW
                          const end = new Date(s.getTime() + m.durationMin * 60_000)
                          return (
                            <UnstyledButton
                              key={m.id}
                              className="ab-pop"
                              onClick={() => setSelectedId(m.id)}
                              aria-pressed={m.id === selectedId}
                              aria-label={`${m.title}, ${DAYS[i]} ${hm(s)}`}
                              style={{
                                position: 'absolute',
                                left: 4,
                                right: 4,
                                top: Math.max(top, 0) + 2,
                                height: (m.durationMin / 60) * ROW - 4,
                                background: KIND[m.kind].bg,
                                outline: m.id === selectedId ? '2px solid #141414' : 'none',
                                outlineOffset: -2,
                                padding: '6px 8px',
                                overflow: 'hidden',
                              }}
                            >
                              <Text size="xs" fw={600} lh={1.2}>
                                {m.title}
                              </Text>
                              <Text size="11px">
                                {hm(s)}–{hm(end)}
                              </Text>
                            </UnstyledButton>
                          )
                        })}
                    </div>
                  )
                })}
              </div>
            </Paper>
            <Paper
              radius="lg"
              p="lg"
              style={{ flex: '1 1 280px', maxWidth: 380 }}
              aria-label="Встреча"
            >
              {selected ? (
                <Stack gap="sm" className="ab-pop">
                  <StatusTag bg={KIND[selected.kind].bg} style={{ alignSelf: 'flex-start' }}>
                    {KIND[selected.kind].label}
                  </StatusTag>
                  <Text component="h2" fz={22} fw={500} m={0}>
                    {selected.title}
                  </Text>
                  <Text size="sm">
                    {dm(new Date(selected.startsAt))}, {hm(new Date(selected.startsAt))} ·{' '}
                    {selected.durationMin} мин
                  </Text>
                  <Text size="sm">С кем: {selected.withName}</Text>
                  <Text size="xs" c="ink.5">
                    Напоминание придет стажеру за час до встречи
                  </Text>
                  <Group gap={8}>
                    <Button
                      variant="outline"
                      color="ink"
                      loading={move.isPending}
                      onClick={() => shift(selected)}
                    >
                      Перенести на час
                    </Button>
                    <Button
                      color="red.2"
                      c="ink.9"
                      loading={cancel.isPending}
                      onClick={() =>
                        cancel.mutate(selected.id, {
                          onSuccess: () => {
                            toast(
                              `Встреча «${selected.title}» отменена, ${selected.withName} получит уведомление`,
                            )
                            setSelectedId(null)
                          },
                        })
                      }
                    >
                      Отменить
                    </Button>
                  </Group>
                </Stack>
              ) : (
                <Text size="sm" c="ink.5">
                  Выбери встречу в календаре, чтобы перенести или отменить. Стажер получит
                  уведомление.
                </Text>
              )}
            </Paper>
          </Group>
        )}
      </Stack>
      {creating && (
        <MeetingModal
          monday={monday}
          onClose={() => setCreating(false)}
          onCreated={(id) => {
            setCreating(false)
            setSelectedId(id)
          }}
        />
      )}
    </Page>
  )
}

function MeetingModal({
  monday,
  onClose,
  onCreated,
}: {
  monday: Date
  onClose: () => void
  onCreated: (id: string) => void
}) {
  const threads = useThreads()
  const create = useCreateMentorMeeting()
  const interns = threads.data ?? []
  const [internId, setInternId] = useState('')
  const [title, setTitle] = useState('')
  const [day, setDay] = useState('2')
  const [hour, setHour] = useState('14')
  const [notify, setNotify] = useState(true)
  const [titleError, setTitleError] = useState<string | null>(null)
  const who = internId || interns[0]?.internId || 'all'
  const name = interns.find((i) => i.internId === who)?.internName

  const submit = () => {
    const finalTitle =
      title.trim() || (who === 'all' ? 'Встреча со стажерами' : `1:1 · ${name ?? 'стажер'}`)
    if (finalTitle.length > 120) return setTitleError('Не длиннее 120 символов')
    const startsAt = addDays(monday, Number(day))
    startsAt.setHours(Number(hour), 0, 0, 0)
    create.mutate(
      {
        internId: who,
        title: finalTitle,
        startsAt: startsAt.toISOString(),
        durationMin: 60,
        notify,
      },
      {
        onSuccess: (m) => {
          toast(
            notify
              ? `Встреча назначена. ${who === 'all' ? 'Стажеры получат' : `${m.withName} получит`} приглашение`
              : 'Встреча назначена без уведомления',
          )
          onCreated(m.id)
        },
      },
    )
  }

  return (
    <Modal opened onClose={onClose} title="Назначить встречу" centered radius="lg">
      <Stack gap="sm">
        <NativeSelect
          label="Стажер"
          value={who}
          onChange={(e) => setInternId(e.currentTarget.value)}
          data={[
            ...interns.map((i) => ({ value: i.internId, label: i.internName })),
            { value: 'all', label: 'все стажеры' },
          ]}
        />
        <TextInput
          label="Тема"
          placeholder={who === 'all' ? 'Встреча со стажерами' : `1:1 · ${name ?? 'стажер'}`}
          value={title}
          onChange={(e) => (setTitle(e.currentTarget.value), setTitleError(null))}
          error={titleError}
        />
        <Group grow>
          <NativeSelect
            label="День"
            value={day}
            onChange={(e) => setDay(e.currentTarget.value)}
            data={DAYS.map((d, i) => ({
              value: String(i),
              label: `${d} ${addDays(monday, i).getDate()}`,
            }))}
          />
          <NativeSelect
            label="Время"
            value={hour}
            onChange={(e) => setHour(e.currentTarget.value)}
            data={HOURS.map((h) => ({ value: String(h), label: `${h}:00` }))}
          />
        </Group>
        <Checkbox
          color="green.6"
          label="Отправить приглашение и напомнить за час"
          checked={notify}
          onChange={(e) => setNotify(e.currentTarget.checked)}
        />
        {create.isError && (
          <Alert color="red" variant="light" role="alert">
            {errorText(create.error)}
          </Alert>
        )}
        <Group justify="flex-end">
          <Button variant="default" onClick={onClose}>
            Отмена
          </Button>
          <Button color="ink" loading={create.isPending} onClick={submit}>
            Назначить
          </Button>
        </Group>
      </Stack>
    </Modal>
  )
}
