import { Anchor, Button, Group, Paper, Stack, Text, UnstyledButton } from '@mantine/core'
import { IconCalendar, IconFileText, IconKey, IconPlus } from '@tabler/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { getRequests, withdrawRequest } from '../api/assistant'
import { errorText } from '../lib/errors'
import { toast } from '../store/toastStore'
import { PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import { JIRA, jiraKey, requestRef } from '../lib/jira'
import type { TRequestItem, TRequestStatus } from '../schemas/assistant'

const REQUEST_STATUS: Record<TRequestStatus, { label: string; bg: string }> = {
  waiting: { label: 'на согласовании', bg: '#BADAFB' },
  approved: { label: 'одобрена', bg: '#C9FFBF' },
  done: { label: 'выполнена', bg: '#C9FFBF' },
  rejected: { label: 'отклонена', bg: '#F7C1C5' },
  cancelled: { label: 'отозвана', bg: '#EDEDED' },
}

type Filter = 'all' | 'waiting' | 'approved' | 'rejected'
const FILTERS: { id: Filter; label: string }[] = [
  { id: 'all', label: 'Все' },
  { id: 'waiting', label: 'На согласовании' },
  { id: 'approved', label: 'Одобрены' },
  { id: 'rejected', label: 'Отклонены и отозваны' },
]

const matches = (f: Filter, r: TRequestItem) =>
  f === 'all' ||
  r.status === f ||
  (f === 'approved' && r.status === 'done') ||
  (f === 'rejected' && r.status === 'cancelled')
const VIA = { buddy: 'через AI-buddy', auto: 'автоматически', manual: 'вручную' } as const

export function RequestsPage({
  selectedId,
  onSelect,
}: {
  selectedId?: string
  onSelect: (id: string) => void
}) {
  const [filter, setFilter] = useState<Filter>('all')
  // Пока есть заявки на согласовании, статусы обновляются сами раз в 10 секунд.
  const query = useQuery({
    queryKey: ['requests'],
    queryFn: getRequests,
    refetchInterval: (q) => (q.state.data?.some((r) => r.status === 'waiting') ? 10_000 : false),
  })

  return (
    <Page maxWidth={1400}>
      <Stack gap="lg" style={{ flex: '999 1 520px', minWidth: 0 }}>
        <PageHeader
          overline="Мои заявки"
          title="Заявки"
          right={
            <Button
              renderRoot={(p) => <Link {...p} to="/chat" search={{ q: 'Хочу оформить заявку' }} />}
              color="ink"
              leftSection={<IconPlus size={18} />}
            >
              Новая заявка через AI-buddy
            </Button>
          }
        />
        {query.isPending && <LoadingBlocks label="Загрузка заявок" />}
        {query.isError && <ErrorBlock error={query.error} onRetry={() => query.refetch()} />}
        {query.data && (
          <>
            <Group gap={8} role="tablist" aria-label="Фильтр заявок">
              {FILTERS.map((f) => {
                const on = f.id === filter
                const n = query.data.filter((r) => matches(f.id, r)).length
                return (
                  <UnstyledButton
                    key={f.id}
                    role="tab"
                    aria-selected={on}
                    onClick={() => setFilter(f.id)}
                    style={{
                      height: 36,
                      padding: '0 14px',
                      borderRadius: 18,
                      border: on ? 0 : '1px solid #D6D6D6',
                      background: on ? '#141414' : '#FFFFFF',
                      color: on ? '#FFFFFF' : '#141414',
                      fontSize: 14,
                    }}
                  >
                    {f.label} <span style={{ opacity: 0.7 }}>{n}</span>
                  </UnstyledButton>
                )
              })}
            </Group>
            <RequestList
              items={query.data.filter((r) => matches(filter, r))}
              selectedId={selectedId ?? query.data[0]?.id}
              onSelect={onSelect}
            />
          </>
        )}
        <Text size="xs" c="ink.5">
          Доступы и заявки в отделы уходят в {JIRA} (в демо это имитация), отгулы – руководителю.
        </Text>
      </Stack>
      {query.data && query.data.length > 0 && (
        <RequestDetail item={query.data.find((r) => r.id === selectedId) ?? query.data[0]} />
      )}
    </Page>
  )
}

function RequestList({
  items,
  selectedId,
  onSelect,
}: {
  items: TRequestItem[]
  selectedId?: string
  onSelect: (id: string) => void
}) {
  if (items.length === 0) {
    return (
      <Paper radius="lg" p="lg">
        <Text c="ink.5">Здесь пока пусто.</Text>
      </Paper>
    )
  }
  return (
    <Paper
      radius="lg"
      py={8}
      component="ul"
      style={{ listStyle: 'none', margin: 0, paddingLeft: 0, paddingRight: 0 }}
      aria-label="Список заявок"
    >
      {items.map((r, i) => {
        const s = REQUEST_STATUS[r.status]
        const Icon =
          r.kind === 'access' ? IconKey : r.kind === 'leave' ? IconCalendar : IconFileText
        const on = r.id === selectedId
        return (
          <li key={r.id}>
            <UnstyledButton
              onClick={() => onSelect(r.id)}
              aria-current={on ? 'true' : undefined}
              style={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                gap: 16,
                padding: '16px 24px',
                borderTop: i === 0 ? 0 : '1px solid #D6D6D6',
                background: on ? '#F4F4F4' : 'transparent',
              }}
            >
              <span
                style={{
                  width: 40,
                  height: 40,
                  flex: 'none',
                  background: s.bg,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <Icon size={20} stroke={1.8} aria-hidden="true" />
              </span>
              <span style={{ flex: 1, minWidth: 0 }}>
                <Text fw={500} size="15px">
                  {r.title}
                </Text>
                <Text size="sm" c="ink.5">
                  {requestRef(r)} · {formatDate(r.createdAt)} · {VIA[r.via]}
                </Text>
              </span>
              <StatusTag bg={s.bg}>{s.label}</StatusTag>
            </UnstyledButton>
          </li>
        )
      })}
    </Paper>
  )
}

function RequestDetail({ item }: { item: TRequestItem }) {
  const s = REQUEST_STATUS[item.status]
  const key = jiraKey(item)
  return (
    <Paper
      component="aside"
      aria-label="Детали заявки"
      radius="lg"
      p="lg"
      style={{
        flex: '1 1 320px',
        maxWidth: 420,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        gap: 18,
      }}
    >
      <Group justify="space-between">
        <Text size="sm" c="ink.5">
          Заявка {requestRef(item)}
        </Text>
        <StatusTag bg={s.bg} size="md">
          {s.label}
        </StatusTag>
      </Group>
      <Text fz={26} fw={500} lh={1.1} style={{ letterSpacing: '-0.02em' }}>
        {item.title}
      </Text>
      {key && (
        <Group gap={8}>
          <StatusTag bg="#E7F1FB">в Jira</StatusTag>
          {item.external?.url ? (
            <Anchor href={item.external.url} target="_blank" rel="noreferrer" size="sm" c="ink.9">
              Открыть {key} в {JIRA}
            </Anchor>
          ) : (
            <Text size="sm" c="ink.5">
              {JIRA} · {key}
            </Text>
          )}
        </Group>
      )}
      <dl
        style={{
          display: 'grid',
          gridTemplateColumns: '120px 1fr',
          gap: '8px 12px',
          margin: 0,
          fontSize: 14,
        }}
      >
        {item.fields.map((f) => (
          <div key={f.label} style={{ display: 'contents' }}>
            <dt style={{ color: '#5C5C5C' }}>{f.label}</dt>
            <dd style={{ margin: 0 }}>{f.value}</dd>
          </div>
        ))}
      </dl>
      <div>
        <Text size="11px" tt="uppercase" c="ink.5" mb={10} style={{ letterSpacing: '0.06em' }}>
          История
        </Text>
        <ol style={{ listStyle: 'none', margin: 0, padding: 0 }}>
          {item.history.map((h, i) => {
            const last = i === item.history.length - 1
            const dot =
              h.state === 'done'
                ? { background: '#04E061' }
                : h.state === 'rejected'
                  ? { background: '#FF4053' }
                  : { background: '#FFFFFF', border: '3px solid #00AAFF' }
            return (
              <li key={i} style={{ display: 'flex', gap: 12 }}>
                <span
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    width: 14,
                  }}
                >
                  <span
                    style={{
                      width: 12,
                      height: 12,
                      borderRadius: 6,
                      marginTop: 3,
                      boxSizing: 'border-box',
                      ...dot,
                    }}
                  />
                  <span
                    style={{
                      flex: 1,
                      width: 2,
                      minHeight: 20,
                      background: last ? 'transparent' : '#D6D6D6',
                    }}
                  />
                </span>
                <span style={{ paddingBottom: 14 }}>
                  <Text size="sm" fw={h.state === 'now' ? 600 : 400}>
                    {h.title}
                  </Text>
                  <Text size="xs" c="ink.5">
                    {h.at}
                  </Text>
                </span>
              </li>
            )
          })}
        </ol>
      </div>
      {item.rejectReason && (
        <div style={{ background: '#F7C1C5', padding: '12px 14px', fontSize: 13, lineHeight: 1.4 }}>
          {item.rejectReason}
        </div>
      )}
      <Button
        renderRoot={(p) => (
          <Link {...p} to="/chat" search={{ q: `Расскажи про заявку №${item.number}` }} />
        )}
        variant="outline"
        color="ink"
        style={{ alignSelf: 'flex-start' }}
      >
        Обсудить в чате
      </Button>
      {item.status === 'waiting' && <WithdrawControl item={item} />}
    </Paper>
  )
}

// Отзыв заявки: сначала спрашиваем, точно ли, – действие нельзя отменить.
function WithdrawControl({ item }: { item: TRequestItem }) {
  const [asking, setAsking] = useState(false)
  const qc = useQueryClient()
  const withdraw = useMutation({
    mutationFn: () => withdrawRequest(item.id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['requests'] })
      qc.invalidateQueries({ queryKey: ['path'] })
      toast(`Заявка №${item.number} отозвана`)
      setAsking(false)
    },
  })
  if (!asking) {
    return (
      <UnstyledButton
        onClick={() => setAsking(true)}
        style={{
          alignSelf: 'flex-start',
          fontSize: 13,
          color: '#5C5C5C',
          textDecoration: 'underline',
        }}
      >
        Отозвать заявку
      </UnstyledButton>
    )
  }
  return (
    <Stack
      gap={10}
      role="group"
      aria-label="Отзыв заявки"
      style={{ background: '#F4F4F4', padding: '12px 14px' }}
    >
      <Text size="sm">
        Отозвать заявку? Она закроется и в {JIRA}, задача на «Моем пути» снова будет «в работе».
        Заявку можно будет оформить заново.
      </Text>
      {withdraw.isError && (
        <Text size="sm" c="red.8" role="alert">
          {errorText(withdraw.error)}
        </Text>
      )}
      <Group gap={8}>
        <Button
          size="xs"
          color="red"
          loading={withdraw.isPending}
          onClick={() => withdraw.mutate()}
        >
          Отозвать
        </Button>
        <Button size="xs" variant="default" onClick={() => setAsking(false)}>
          Оставить
        </Button>
      </Group>
    </Stack>
  )
}

function formatDate(iso: string) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
    timeZone: 'Europe/Moscow',
  })
}
