import { Button, Paper, Text, TextInput } from '@mantine/core'
import { useState } from 'react'
import { Mascot } from '../components/Mascot'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import { useApprovals, useDecide } from '../hooks/useStaff'
import { errorText } from '../lib/errors'
import type { TApproval } from '../schemas/staff'
import { toast } from '../store/toastStore'

// Согласования руководителя: заявки стажеров, где он согласующий. Отказ – только с причиной.
const STATE: Record<TApproval['status'], { text: string; bg: string }> = {
  waiting: { text: 'ждет тебя', bg: '#BADAFB' },
  approved: { text: 'согласовано', bg: '#C9FFBF' },
  rejected: { text: 'отклонено', bg: '#F7C1C5' },
}

export function ApprovalsPage() {
  const list = useApprovals()
  const waiting = list.data?.filter((a) => a.status === 'waiting') ?? []
  return (
    <Page maxWidth={980}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 24, flex: 1, minWidth: 0 }}>
        <PageHeader
          overline="Согласования"
          title={
            <>
              Ждут <Marker color="#00AAFF">твоего ответа</Marker>
            </>
          }
        />
        {list.isPending && <LoadingBlocks label="Загрузка заявок" />}
        {list.isError && <ErrorBlock error={list.error} onRetry={() => list.refetch()} />}
        {list.data && waiting.length === 0 && (
          <Paper radius="lg" p="lg" style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
            <Mascot mood="happy" size={48} />
            <Text>Все заявки разобраны. Новые придут сюда и в уведомления.</Text>
          </Paper>
        )}
        {list.data?.map((a) => (
          <ApprovalCard key={a.id} a={a} />
        ))}
      </div>
    </Page>
  )
}

function ApprovalCard({ a }: { a: TApproval }) {
  const decide = useDecide()
  const [reason, setReason] = useState('')
  const [hint, setHint] = useState<string | null>(null)
  const st = STATE[a.status]
  const date = new Date(a.createdAt).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })
  return (
    <Paper
      radius="lg"
      p="lg"
      component="section"
      aria-label={`Заявка №${a.number}: ${a.title}`}
      style={{ display: 'flex', flexDirection: 'column', gap: 14 }}
    >
      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: 12,
        }}
      >
        <div>
          <Text fz={17} fw={500}>
            {a.title}
          </Text>
          <Text fz={12} c="ink.5">
            {a.internName} · заявка №{a.number} · {date}
          </Text>
        </div>
        <StatusTag bg={st.bg}>{st.text}</StatusTag>
      </div>
      <dl
        style={{
          margin: 0,
          display: 'grid',
          gridTemplateColumns: '140px 1fr',
          gap: '6px 12px',
          fontSize: 14,
        }}
      >
        {a.fields.map((f) => (
          <div key={f.label} style={{ display: 'contents' }}>
            <dt style={{ color: '#5C5C5C' }}>{f.label}</dt>
            <dd style={{ margin: 0 }}>{f.value}</dd>
          </div>
        ))}
        {a.rejectReason && (
          <>
            <dt style={{ color: '#5C5C5C' }}>Причина отказа</dt>
            <dd style={{ margin: 0 }}>{a.rejectReason}</dd>
          </>
        )}
      </dl>
      {a.status === 'waiting' && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'flex-start' }}>
          <Button
            color="green.5"
            c="ink.6"
            loading={decide.isPending && decide.variables?.decision === 'approve'}
            onClick={() =>
              decide.mutate(
                { id: a.id, decision: 'approve' },
                {
                  onSuccess: () =>
                    toast(`Заявка №${a.number} согласована. ${a.internName} получит доступ`),
                },
              )
            }
          >
            Согласовать
          </Button>
          <TextInput
            aria-label="Причина отказа"
            placeholder="Причина отказа – ее увидит стажер"
            value={reason}
            onChange={(e) => {
              setReason(e.currentTarget.value)
              setHint(null)
            }}
            error={hint}
            style={{ flex: '1 1 220px' }}
          />
          <Button
            color="red.2"
            c="ink.6"
            loading={decide.isPending && decide.variables?.decision === 'reject'}
            onClick={() => {
              if (!reason.trim()) return setHint('Напиши причину: ее увидит стажер')
              decide.mutate(
                { id: a.id, decision: 'reject', reason },
                {
                  onSuccess: () =>
                    toast(`Заявка №${a.number} отклонена, ${a.internName} увидит причину`),
                },
              )
            }}
          >
            Отклонить
          </Button>
        </div>
      )}
      {decide.isError && (
        <Text c="red.7" fz={13}>
          {errorText(decide.error)}
        </Text>
      )}
    </Paper>
  )
}
