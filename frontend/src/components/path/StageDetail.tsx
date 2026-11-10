import { Button, Group, NativeSelect, Paper, Stack, Text, Title } from '@mantine/core'
import { IconCheck } from '@tabler/icons-react'
import { resourceTag, taskBadge, taskDot } from '../../lib/pathUi'
import type { TStageView, TTaskView } from '../../schemas/path'
import { StatusTag } from '../StatusTag'

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
            return (
              <li
                key={t.id}
                style={{
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: 14,
                  padding: '14px 12px',
                  margin: '0 -12px',
                  borderTop: '1px solid #D6D6D6',
                  background: t.isNext ? '#F4F4F4' : 'transparent',
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
                  {t.isNext && (onAsk || onComplete) && (
                    <Group gap={8} mt={6}>
                      {onAsk && t.action && (
                        <Button size="sm" color="ink" onClick={() => onAsk(t)}>
                          Сделать с AI-buddy
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
                </Stack>
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
        {stage.tip && (
          <div style={{ background: stage.color, padding: '16px 18px' }}>
            <div className="ab-hand" style={{ fontSize: 22, lineHeight: 1 }}>
              Совет
            </div>
            <Text size="sm" mt={6} style={{ lineHeight: 1.4 }}>
              {stage.tip}
            </Text>
          </div>
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
