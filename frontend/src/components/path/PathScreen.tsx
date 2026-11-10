import { Button, Group, Paper, SimpleGrid, Stack, Text, Title, UnstyledButton } from '@mantine/core'
import type { ReactNode } from 'react'
import type { TPathView, TTaskView } from '../../schemas/path'
import { StatusTag } from '../StatusTag'
import { MeetingCard } from './MeetingCard'
import { NextStepCard } from './NextStepCard'
import { ProgressRing } from './ProgressRing'
import { StageCard } from './StageCard'
import { StageDetail, type TaskStatus } from './StageDetail'
import { WeekTrack } from './WeekTrack'

interface PathScreenProps {
  path: TPathView
  openCode: string | null
  onOpen: (code: string | null) => void
  header: ReactNode
  /** Режим стажера. */
  onComplete?: (taskId: string) => void
  onAsk?: (task: Pick<TTaskView, 'id' | 'title' | 'action'>) => void
  /** Режим наставника. */
  onSetStatus?: (taskId: string, status: TaskStatus) => void
  busyTaskId?: string | null
  now?: Date
}

// «Мой путь»: в обзоре кольцо прогресса крупно, при открытии этапа оно сжимается в строку,
// а главным становятся задачи этапа.
export function PathScreen({
  path,
  openCode,
  onOpen,
  header,
  onComplete,
  onAsk,
  onSetStatus,
  busyTaskId,
  now,
}: PathScreenProps) {
  const stage = openCode ? (path.stages.find((s) => s.code === openCode) ?? null) : null
  const expanded = Boolean(stage)
  const mentorMode = Boolean(onSetStatus)

  return (
    <Stack gap="lg" style={{ flex: '999 1 560px', minWidth: 0 }}>
      {header}

      <Paper
        radius="lg"
        style={{
          padding: expanded ? '16px 24px' : 32,
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'center',
          gap: expanded ? 20 : 40,
          transition: 'padding .45s cubic-bezier(.2,.8,.2,1), gap .45s cubic-bezier(.2,.8,.2,1)',
        }}
      >
        <ProgressRing path={path} size={expanded ? 96 : 260} compact={expanded} />
        {stage ? (
          <Group
            className="ab-pop"
            justify="space-between"
            gap="sm"
            style={{ flex: '1 1 320px', minWidth: 0 }}
          >
            <div>
              <Text size="11px" tt="uppercase" c="ink.5" style={{ letterSpacing: '0.06em' }}>
                Этап {stage.code} · {stage.doneCount} из {stage.totalCount} задач
              </Text>
              <Text fz={24} fw={500} style={{ letterSpacing: '-0.02em' }}>
                {stage.title}
              </Text>
            </div>
            <Group gap={6} role="tablist" aria-label="Этапы">
              {path.stages.map((s) => {
                const on = s.code === stage.code
                return (
                  <UnstyledButton
                    key={s.code}
                    role="tab"
                    aria-selected={on}
                    aria-label={`Этап ${s.code}: ${s.title}`}
                    onClick={() => onOpen(s.code)}
                    style={{
                      width: 40,
                      height: 40,
                      borderRadius: 8,
                      border: on ? 0 : '1px solid #D6D6D6',
                      background: on ? '#141414' : '#FFFFFF',
                      color: on ? '#FFFFFF' : '#141414',
                      fontWeight: 600,
                      fontSize: 14,
                      textAlign: 'center',
                    }}
                  >
                    {s.code}
                  </UnstyledButton>
                )
              })}
              <Button variant="outline" color="ink" ml={6} onClick={() => onOpen(null)}>
                Все этапы
              </Button>
            </Group>
          </Group>
        ) : (
          <Stack className="ab-pop" gap={20} style={{ flex: '1 1 320px', minWidth: 0 }}>
            <div>
              <Title order={2} fz={22}>
                Прогресс адаптации
              </Title>
              <Text size="sm" c="ink.5" mt={6}>
                Каждое деление кольца – одна задача.
              </Text>
            </div>
            <Group gap={12} grow>
              <Counter value={path.progress.done} label="выполнено" bg="#C9FFBF" />
              <Counter value={path.progress.inProgress} label="в работе" bg="#BADAFB" />
              <Counter value={path.progress.todo} label="впереди" bg="#F4F4F4" />
            </Group>
            <WeekTrack week={path.week} />
          </Stack>
        )}
      </Paper>

      {stage ? (
        <StageDetail
          stage={stage}
          onComplete={onComplete}
          onAsk={onAsk}
          onSetStatus={onSetStatus}
          busyTaskId={busyTaskId}
        />
      ) : (
        <>
          <Stack className="ab-rise" gap="sm">
            <Group justify="space-between" align="baseline">
              <Title order={2}>Этапы адаптации</Title>
              <Text size="sm" c="ink.5">
                Нажми на этап, чтобы увидеть задачи
              </Text>
            </Group>
            <SimpleGrid cols={{ base: 1, xs: 2, md: 4 }} spacing="md">
              {path.stages.map((s) => (
                <StageCard key={s.id} stage={s} onOpen={onOpen} />
              ))}
            </SimpleGrid>
          </Stack>
          <SimpleGrid className="ab-rise-2" cols={{ base: 1, md: 2 }} spacing="lg">
            {path.nextStep ? (
              <NextStepCard
                step={path.nextStep}
                onAsk={() =>
                  onAsk?.({
                    id: path.nextStep!.taskId,
                    title: path.nextStep!.title,
                    action: path.nextStep!.action,
                  })
                }
                onOpenStage={() => onOpen(path.nextStep!.stageCode)}
              />
            ) : (
              <Paper radius="lg" p="lg">
                <StatusTag bg="#04E061">все готово</StatusTag>
                <Text fz={24} fw={500} mt="md">
                  Все задачи адаптации выполнены
                </Text>
              </Paper>
            )}
            <MeetingCard mentor={path.intern.mentor} meeting={path.nextMeeting} now={now} />
          </SimpleGrid>
        </>
      )}
      {mentorMode && (
        <Text size="xs" c="ink.5">
          Ты смотришь путь стажера как наставник: статусы задач можно менять.
        </Text>
      )}
      <Text size="xs" c="ink.5">
        Демо-версия: данные синтетические, инструкции не являются регламентами Авито.
      </Text>
    </Stack>
  )
}

function Counter({ value, label, bg }: { value: number; label: string; bg: string }) {
  return (
    <div style={{ background: bg, padding: '12px 14px' }}>
      <Text fz={28} fw={500} lh={1} style={{ letterSpacing: '-0.03em' }}>
        {value}
      </Text>
      <Text size="xs" mt={4}>
        {label}
      </Text>
    </div>
  )
}
