import { Button, Group, Paper, Stack, Text } from '@mantine/core'
import { dueText } from '../../lib/pathUi'
import type { TPathView } from '../../schemas/path'
import { StatusTag } from '../StatusTag'

interface NextStepCardProps {
  step: NonNullable<TPathView['nextStep']>
  onAsk: () => void
  onOpenStage: () => void
}

// Активная задача = следующий шаг стажера (требование кейса: показывать следующий шаг).
export function NextStepCard({ step, onAsk, onOpenStage }: NextStepCardProps) {
  const due = dueText(step.dueInDays)
  const overdue = (step.dueInDays ?? 0) < 0
  return (
    <Paper radius="lg" p="lg" style={{ outline: '2px solid #141414', outlineOffset: -2 }}>
      <Stack gap="md">
        <Group justify="space-between" gap={8}>
          <StatusTag
            bg="#04E061"
            style={{
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
              fontSize: 11,
              fontWeight: 600,
            }}
          >
            Следующий шаг
          </StatusTag>
          <Text size="xs" c={overdue ? 'red.8' : 'ink.5'}>
            этап {step.stageCode}
            {due ? ` · ${due}` : ''}
          </Text>
        </Group>
        <Text fz={24} fw={500} lh={1.1} style={{ letterSpacing: '-0.02em' }}>
          {step.title}
        </Text>
        <Group gap={8} wrap="nowrap" aria-label="Шаги задачи" fz="xs">
          <Dot state="done">1</Dot>
          <Text size="xs">Узнать порядок</Text>
          <span style={{ flex: 1, height: 2, background: '#00AAFF' }} />
          <Dot state={step.status === 'waiting' ? 'done' : 'now'}>2</Dot>
          <Text size="xs" fw={600}>
            {step.action ? 'Заявка' : 'Сделать'}
          </Text>
          <span
            style={{
              flex: 1,
              height: 2,
              background: step.status === 'waiting' ? '#00AAFF' : '#D6D6D6',
            }}
          />
          <Dot state={step.status === 'waiting' ? 'now' : 'todo'}>3</Dot>
          <Text size="xs" c="ink.5">
            Готово
          </Text>
        </Group>
        <Group gap={8}>
          {step.action && (
            <Button color="ink" onClick={onAsk}>
              Сделать с AI-buddy
            </Button>
          )}
          <Button variant="outline" color="ink" onClick={onOpenStage}>
            Открыть этап
          </Button>
        </Group>
      </Stack>
    </Paper>
  )
}

function Dot({ state, children }: { state: 'done' | 'now' | 'todo'; children: string }) {
  const style =
    state === 'done'
      ? { background: '#04E061', border: 0 }
      : state === 'now'
        ? { background: '#00AAFF', border: 0 }
        : { background: 'transparent', border: '1px solid #D6D6D6', color: '#5C5C5C' }
  return (
    <span
      style={{
        width: 22,
        height: 22,
        borderRadius: 11,
        flex: 'none',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontWeight: 600,
        fontSize: 12,
        boxSizing: 'border-box',
        ...style,
      }}
    >
      {state === 'done' ? '✓' : children}
    </span>
  )
}
