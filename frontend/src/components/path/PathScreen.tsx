import { Button, Group, Paper, Stack, Text } from '@mantine/core'
import type { ReactNode } from 'react'
import { dueText } from '../../lib/pathUi'
import type { TPathView, TTaskView } from '../../schemas/path'
import { StatusTag } from '../StatusTag'
import { ProgressTicks } from './ProgressTicks'
import { StageCarousel } from './StageCarousel'
import { StageDetail, type TaskStatus } from './StageDetail'
import classes from './PathScreen.module.css'

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
  /** Карточка целей стажировки (GoalsCard): видна, пока не открыт этап. */
  goals?: ReactNode
}

// «Мой путь»: слева прогресс и следующий шаг, справа этапы-карточки, под ними цели
// или задачи выбранного этапа.
export function PathScreen({
  path,
  openCode,
  onOpen,
  header,
  onComplete,
  onAsk,
  onSetStatus,
  busyTaskId,
  goals,
}: PathScreenProps) {
  const stage = openCode ? (path.stages.find((s) => s.code === openCode) ?? null) : null
  const current = path.stages.find((s) => s.status === 'current') ?? path.stages[0]
  const mentorMode = Boolean(onSetStatus)
  const step = path.nextStep
  const due = step ? dueText(step.dueInDays) : null

  return (
    <Stack gap="lg" style={{ flex: '999 1 600px', minWidth: 0 }}>
      {header}

      <div className={classes.grid}>
        <Paper
          radius={20}
          p="lg"
          pt={28}
          className={`${classes.progress} ab-rise`}
          aria-label="Прогресс адаптации"
          component="section"
        >
          <Text
            component="h2"
            fz={22}
            fw={500}
            lh={1.25}
            m={0}
            style={{ letterSpacing: '-0.02em' }}
          >
            {mentorMode
              ? `Пройдено ${path.progress.percent}% пути адаптации`
              : `Позади ${path.progress.percent}% пути адаптации!`}
          </Text>
          <div className={classes.ring}>
            <ProgressTicks path={path} />
          </div>
          <div className={classes.legend}>
            <span>
              <i style={{ background: '#04E061' }} />
              готово {path.progress.done}
            </span>
            <span>
              <i style={{ background: '#00AAFF' }} />в работе {path.progress.inProgress}
            </span>
            <span>
              <i style={{ background: '#E4E4E4' }} />
              впереди {path.progress.todo}
            </span>
          </div>
          <div style={{ height: 1, background: '#EDEDED' }} />
          {step ? (
            <>
              <Stack gap={10} style={{ flex: 1 }}>
                <Group gap={10}>
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
                  <Text size="xs" c={(step.dueInDays ?? 0) < 0 ? 'red.8' : 'ink.5'}>
                    этап {step.stageCode}
                    {due ? ` · ${due}` : ''}
                  </Text>
                </Group>
                <Text fz={26} fw={500} lh={1.12} style={{ letterSpacing: '-0.02em' }}>
                  {step.title}
                </Text>
                {step.action && !mentorMode && (
                  <Text size="sm" c="ink.5" lh={1.4}>
                    AI-buddy подготовит все за тебя, останется только подтвердить.
                  </Text>
                )}
              </Stack>
              <Stack gap={8}>
                {step.action && onAsk && (
                  <Button
                    color="ink"
                    size="md"
                    radius="md"
                    onClick={() =>
                      onAsk({ id: step.taskId, title: step.title, action: step.action })
                    }
                  >
                    Сделать с AI-buddy
                  </Button>
                )}
                <Button
                  variant="outline"
                  color="ink"
                  size="md"
                  radius="md"
                  onClick={() => onOpen(step.stageCode)}
                >
                  Открыть этап
                </Button>
              </Stack>
            </>
          ) : (
            <Stack gap={8}>
              <StatusTag bg="#04E061" style={{ alignSelf: 'flex-start' }}>
                все готово
              </StatusTag>
              <Text fz={22} fw={500}>
                Все задачи адаптации выполнены
              </Text>
            </Stack>
          )}
        </Paper>

        {path.stages.length > 0 && (
          <StageCarousel
            stages={path.stages}
            selectedCode={stage?.code ?? current?.code ?? ''}
            onOpen={onOpen}
          />
        )}

        {stage ? (
          <Stack gap="sm" className="ab-pop" style={{ minWidth: 0 }}>
            <Group justify="space-between" gap={8}>
              <Text component="h2" fz={20} fw={500} m={0}>
                Этап {stage.code} · {stage.title}
              </Text>
              <Button variant="outline" color="ink" size="sm" onClick={() => onOpen(null)}>
                К целям
              </Button>
            </Group>
            <StageDetail
              stage={stage}
              onComplete={onComplete}
              onAsk={onAsk}
              onSetStatus={onSetStatus}
              busyTaskId={busyTaskId}
            />
          </Stack>
        ) : (
          <div className="ab-rise-2" style={{ minWidth: 0 }}>
            {goals}
          </div>
        )}
      </div>

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
