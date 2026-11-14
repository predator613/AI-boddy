import {
  Button,
  Group,
  NativeSelect,
  Paper,
  Stack,
  Text,
  Title,
  UnstyledButton,
} from '@mantine/core'
import { IconCheck, IconChevronDown } from '@tabler/icons-react'
import { useState } from 'react'
import { resourceTag, taskBadge, taskDot } from '../../lib/pathUi'
import type { TStageView, TTaskView } from '../../schemas/path'
import { StatusTag } from '../StatusTag'
import { shortDate } from '../../lib/dates'

export type TaskStatus = TTaskView['status']

const STATUS_OPTIONS: { value: TaskStatus; label: string }[] = [
  { value: 'todo', label: 'впереди' },
  { value: 'in_progress', label: 'в работе' },
  { value: 'waiting', label: 'на согласовании' },
  { value: 'done', label: 'готово' },
]

interface StageDetailProps {
  stage: TStageView
  /** Стажер: «Уже сделано». */
  onComplete?: (taskId: string) => void
  /** Стажер: «Сделать с AI-buddy». */
  onAsk?: (task: TTaskView) => void
  /** Наставник: смена статуса задачи. */
  onSetStatus?: (taskId: string, status: TaskStatus) => void
  busyTaskId?: string | null
}

// Раскрытый этап: задачи слева, дополнения (доступы, люди, материалы) справа.
export function StageDetail({
  stage,
  onComplete,
  onAsk,
  onSetStatus,
  busyTaskId,
}: StageDetailProps) {
  // Раскрытая задача: по клику видно, что сделать, кто поможет и когда задача готова.
  const [openId, setOpenId] = useState<string | null>(null)
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 24, alignItems: 'flex-start' }}>
      <Paper className="ab-rise" radius="lg" p="lg" style={{ flex: '999 1 380px', minWidth: 0 }}>
        <Group justify="space-between" align="baseline" mb="xs">
          <Title order={2}>Задачи этапа</Title>
          <Text size="sm" c="ink.5">
            {stage.doneCount} из {stage.totalCount} задач
          </Text>
        </Group>
        <ul
          style={{ listStyle: 'none', margin: 0, padding: 0 }}
          aria-label={`Задачи этапа ${stage.title}`}
        >
          {stage.tasks.map((t) => {
            const badge = taskBadge(t)
            const dot = taskDot(t.status)
            const done = t.status === 'done'
            const open = openId === t.id
            return (
              <li
                key={t.id}
                style={{
                  padding: '4px 12px 10px',
                  margin: '0 -12px',
                  borderTop: '1px solid #D6D6D6',
                  background: t.isNext ? '#F4F4F4' : 'transparent',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'flex-start', gap: 14, paddingTop: 10 }}>
                  <UnstyledButton
                    onClick={() => setOpenId(open ? null : t.id)}
                    aria-expanded={open}
                    aria-label={`${t.title}: подробнее`}
                    style={{
                      flex: 1,
                      minWidth: 0,
                      display: 'flex',
                      alignItems: 'flex-start',
                      gap: 14,
                    }}
                  >
                    <span
                      style={{
                        width: 24,
                        height: 24,
                        borderRadius: 12,
                        flex: 'none',
                        marginTop: 1,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        background: dot.bg,
                        border: dot.border,
                        boxSizing: 'border-box',
                      }}
                    >
                      {done && <IconCheck size={13} stroke={3} aria-hidden="true" />}
                    </span>
                    <Stack gap={4} style={{ flex: 1, minWidth: 0 }}>
                      <Text fw={500} size="15px" c={done ? 'ink.5' : 'ink.6'}>
                        {t.title}
                      </Text>
                      {t.note && (
                        <Text size="sm" c="ink.5">
                          {t.note}
                        </Text>
                      )}
                    </Stack>
                    <IconChevronDown
                      size={18}
                      aria-hidden="true"
                      style={{
                        flex: 'none',
                        marginTop: 3,
                        color: '#5C5C5C',
                        transform: open ? 'rotate(180deg)' : 'none',
                        transition: 'transform .2s ease',
                      }}
                    />
                  </UnstyledButton>
                  {onSetStatus ? (
                    <NativeSelect
                      aria-label={`Статус задачи «${t.title}»`}
                      size="xs"
                      w={160}
                      data={STATUS_OPTIONS}
                      value={t.status}
                      disabled={busyTaskId === t.id}
                      onChange={(e) => onSetStatus(t.id, e.currentTarget.value as TaskStatus)}
                    />
                  ) : (
                    <StatusTag bg={badge.bg} border={badge.border} color={badge.color}>
                      {badge.label}
                    </StatusTag>
                  )}
                </div>
                {open && <TaskDetailsPanel task={t} />}
                {t.isNext && (onAsk || onComplete) && (
                  <Group gap={8} mt={10} ml={38}>
                    {onAsk && t.action && (
                      <Button size="sm" color="ink" onClick={() => onAsk(t)}>
                        {t.action.kind === 'access_request'
                          ? 'Подать заявку'
                          : 'Сделать с AI-buddy'}
                      </Button>
                    )}
                    {onComplete && (
                      <Button
                        size="sm"
                        variant="outline"
                        color="ink"
                        loading={busyTaskId === t.id}
                        onClick={() => onComplete(t.id)}
                      >
                        Уже сделано
                      </Button>
                    )}
                  </Group>
                )}
              </li>
            )
          })}
        </ul>
      </Paper>

      <Stack className="ab-rise-2" gap="md" style={{ flex: '1 1 260px', minWidth: 0 }}>
        {stage.resources.length > 0 && (
          <Paper radius="lg" p="md" component="aside" aria-label="Дополнительно к этапу">
            <Title order={3} mb={8}>
              {resourcesTitle(stage)}
            </Title>
            {stage.resources.map((r) => {
              const tag = resourceTag(r.tagKind)
              return (
                <Group
                  key={r.id}
                  gap={10}
                  wrap="nowrap"
                  py={10}
                  style={{ borderTop: '1px solid #D6D6D6' }}
                >
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <Text size="sm">{r.name}</Text>
                    {r.note && (
                      <Text size="xs" c="ink.5">
                        {r.note}
                      </Text>
                    )}
                  </div>
                  {r.tag && (
                    <StatusTag bg={tag.bg} border={tag.border} color={tag.color}>
                      {r.tag}
                    </StatusTag>
                  )}
                </Group>
              )
            })}
          </Paper>
        )}
      </Stack>
    </div>
  )
}

function resourcesTitle(stage: TStageView) {
  const kinds = new Set(stage.resources.map((r) => r.kind))
  if (kinds.size === 1 && kinds.has('access')) return 'Статус доступов'
  if (kinds.has('person')) return 'Твоя команда'
  if (kinds.has('tool')) return 'Инструменты'
  if (kinds.has('article')) return 'Материалы'
  return 'Полезное'
}

function TaskDetailsPanel({ task }: { task: TTaskView }) {
  const d = task.details
  const rows: [string, string][] = [['Срок', shortDate(task.dueDate)]]
  if (d) {
    rows.push(['Кто поможет', d.helper], ['Готово, когда', d.doneWhen])
  }
  return (
    <div
      className="ab-pop"
      style={{ margin: '10px 0 4px 38px', display: 'flex', flexDirection: 'column', gap: 12 }}
    >
      <Text size="sm" style={{ lineHeight: 1.5, maxWidth: 640 }}>
        {d?.about || task.note || 'Подробностей пока нет – спроси наставника или AI-buddy.'}
      </Text>
      <dl
        style={{
          margin: 0,
          display: 'grid',
          gridTemplateColumns: '130px 1fr',
          gap: '6px 12px',
          fontSize: 13,
          lineHeight: 1.4,
        }}
      >
        {rows.map(([k, v]) => (
          <div key={k} style={{ display: 'contents' }}>
            <dt style={{ color: '#5C5C5C' }}>{k}</dt>
            <dd style={{ margin: 0 }}>{v}</dd>
          </div>
        ))}
      </dl>
    </div>
  )
}
