import { Button, Group, Paper, Stack, Text, Title, UnstyledButton } from '@mantine/core'
import { IconCheck } from '@tabler/icons-react'
import { useState } from 'react'
import { useAgreeGoals, useInternGoals, useMyGoals } from '../../hooks/useGoals'
import { agreedLabel, type TGoal, type TGoalStep, type TGoalsView } from '../../schemas/goals'
import { ErrorBlock } from '../QueryState'
import { StatusTag } from '../StatusTag'

const GREEN = '#04E061'
const BLUE = '#00AAFF'

interface GoalsCardProps {
  /** Не задан – цели текущего стажера («Мой путь»), задан – карточка стажера у наставника. */
  internId?: string
}

// «Мои цели на стажировку»: цели с критерием успеха, согласованные с руководителем.
// Прогресс не кликается: он складывается из шагов – задач плана и подтверждений наставника.
export function GoalsCard({ internId }: GoalsCardProps) {
  return internId ? <InternGoals internId={internId} /> : <MyGoals />
}

function MyGoals() {
  const query = useMyGoals()
  if (query.isPending) return null
  if (query.isError) return <ErrorBlock error={query.error} onRetry={() => query.refetch()} />
  return <GoalsView view={query.data} />
}

function InternGoals({ internId }: { internId: string }) {
  const query = useInternGoals(internId)
  if (query.isPending) return null
  if (query.isError) return <ErrorBlock error={query.error} onRetry={() => query.refetch()} />
  return <GoalsView view={query.data} internId={internId} />
}

const SOURCE: Record<TGoalStep['source'], string> = {
  task: 'этап',
  mentor: 'наставник',
  lead: 'руководитель',
}

function stepSource(s: TGoalStep) {
  return s.source === 'task' && s.stageCode ? `этап ${s.stageCode}` : SOURCE[s.source]
}

function GoalRow({ goal: g, showUpdate }: { goal: TGoal; showUpdate: boolean }) {
  const done = g.steps.filter((s) => s.status === 'done').length
  const pct = g.progressPct
  // Шаги спрятаны, пока стажер не раскроет цель: на экране остаются название, критерий и полоска.
  const [open, setOpen] = useState(false)
  return (
    <div
      aria-label={`${g.title}: ${pct}%`}
      style={{
        background: '#F4F4F4',
        borderRadius: 12,
        padding: '12px 16px',
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
      }}
    >
      <UnstyledButton
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        aria-label={`${g.title}: показать шаги`}
        disabled={g.steps.length === 0}
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          gap: 16,
        }}
      >
        <div>
          <Text fz={15} fw={500} lh={1.25}>
            {g.title}
          </Text>
          <Text size="xs" c="ink.5" mt={3}>
            критерий: {g.criterion}
            {g.competency ? ` · ${g.competency}` : ''}
          </Text>
        </div>
        <div style={{ textAlign: 'right', flex: 'none' }}>
          <Text fz={16} fw={600}>
            {pct}%
          </Text>
          {g.steps.length > 0 && (
            <Text size="11px" c="ink.5">
              {done} из {g.steps.length} шагов {open ? '▴' : '▾'}
            </Text>
          )}
        </div>
      </UnstyledButton>
      {g.steps.length > 0 ? (
        <>
          <div style={{ display: 'flex', gap: 4 }} aria-hidden="true">
            {g.steps.map((s) => (
              <span
                key={s.id}
                style={{
                  flex: 1,
                  height: 6,
                  borderRadius: 3,
                  background:
                    s.status === 'done' ? GREEN : s.status === 'in_progress' ? BLUE : '#FFFFFF',
                }}
              />
            ))}
          </div>
          {open && (
            <ul
              className="ab-pop"
              aria-label={`Шаги цели «${g.title}»`}
              style={{
                listStyle: 'none',
                margin: 0,
                padding: 0,
                display: 'flex',
                flexWrap: 'wrap',
                gap: 6,
              }}
            >
              {g.steps.map((s) => (
                <li
                  key={s.id}
                  title={
                    s.source === 'task'
                      ? 'Закроется вместе с задачей плана'
                      : `Подтверждает ${SOURCE[s.source]}`
                  }
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    fontSize: 12,
                    background: '#FFFFFF',
                    borderRadius: 14,
                    padding: '4px 10px 4px 6px',
                    color: s.status === 'todo' ? '#5C5C5C' : '#141414',
                  }}
                >
                  <span
                    style={{
                      width: 14,
                      height: 14,
                      borderRadius: 7,
                      flex: 'none',
                      boxSizing: 'border-box',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: s.status === 'done' ? GREEN : '#FFFFFF',
                      border:
                        s.status === 'in_progress'
                          ? `2px solid ${BLUE}`
                          : s.status === 'done'
                            ? 0
                            : '1px solid #D6D6D6',
                    }}
                  >
                    {s.status === 'done' && <IconCheck size={9} stroke={4} color="#141414" />}
                  </span>
                  {s.title}
                  <span style={{ color: '#5C5C5C' }}>· {stepSource(s)}</span>
                </li>
              ))}
            </ul>
          )}
        </>
      ) : (
        <span style={{ height: 6, borderRadius: 3, background: '#FFFFFF', overflow: 'hidden' }}>
          <span
            style={{
              display: 'block',
              height: 6,
              width: `${pct}%`,
              background: pct === 100 ? GREEN : BLUE,
            }}
          />
        </span>
      )}
      {showUpdate && g.lastUpdate && (
        <Text size="xs" c="ink.5">
          {g.lastUpdate.author.name}: {g.lastUpdate.note || `${g.lastUpdate.progressPct}%`}
        </Text>
      )}
    </div>
  )
}

function GoalsView({ view, internId }: { view: TGoalsView; internId?: string }) {
  const agree = useAgreeGoals(view.internId)
  const own = !internId

  return (
    <Paper
      radius={20}
      p="lg"
      component="section"
      aria-label={own ? 'Мои цели на стажировку' : 'Цели стажировки'}
    >
      <Stack gap={12}>
        <Group justify="space-between" gap={8} align="center">
          <Title order={2}>{own ? 'Мои цели на стажировку' : 'Цели стажировки'}</Title>
          <StatusTag bg={view.agreed ? '#C9FFBF' : '#F7C1C5'}>{agreedLabel(view)}</StatusTag>
        </Group>

        {view.goals.length === 0 ? (
          <Text c="ink.5">
            {own
              ? 'Цели появятся после встречи с руководителем. AI-buddy поможет внести их в трекер прогресса.'
              : 'Цели еще не внесены.'}
          </Text>
        ) : (
          view.goals.map((g) => <GoalRow key={g.id} goal={g} showUpdate={!own} />)
        )}

        <Text size="xs" c="ink.5" lh={1.45}>
          Нажми на цель, чтобы увидеть шаги. Прогресс считается по ним сам.
        </Text>
        {!own && view.canAgree && !view.agreed && (
          <Button
            color="ink"
            loading={agree.isPending}
            onClick={() => agree.mutate()}
            style={{ alignSelf: 'flex-start' }}
          >
            Согласовать цели
          </Button>
        )}
      </Stack>
    </Paper>
  )
}
