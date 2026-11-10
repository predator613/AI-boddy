import { Button, Paper, SimpleGrid, Stack, Text, Title, UnstyledButton } from '@mantine/core'
import { useAgreeGoals, useGoalProgress, useInternGoals, useMyGoals } from '../../hooks/useGoals'
import { agreedLabel, nextPct, type TGoalsView } from '../../schemas/goals'
import { ErrorBlock } from '../QueryState'
import { StatusTag } from '../StatusTag'

const GREEN = '#04E061'
const BLUE = '#00AAFF'

interface GoalsCardProps {
  /** Не задан – цели текущего стажера («Мой путь»), задан – карточка стажера у наставника. */
  internId?: string
}

// «Мои цели на стажировку»: цели с критерием успеха, согласованные с руководителем.
// Стажер нажимает на цель, чтобы отметить прогресс (+25%), наставник видит обновления.
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

function GoalsView({ view, internId }: { view: TGoalsView; internId?: string }) {
  const progress = useGoalProgress(internId)
  const agree = useAgreeGoals(view.internId)
  const own = !internId

  return (
    <Paper radius="lg" p="lg" style={{ gridColumn: '1 / -1' }}>
      <Stack gap={14}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            gap: 8,
            flexWrap: 'wrap',
            alignItems: 'baseline',
          }}
        >
          <Title order={2}>{own ? 'Мои цели на стажировку' : 'Цели стажировки'}</Title>
          <StatusTag bg={view.agreed ? '#C9FFBF' : '#F7C1C5'}>{agreedLabel(view)}</StatusTag>
        </div>

        {view.goals.length === 0 ? (
          <Text c="ink.5">
            {own
              ? 'Цели появятся после встречи с руководителем. AI-buddy поможет внести их в трекер прогресса.'
              : 'Цели еще не внесены.'}
          </Text>
        ) : (
          <SimpleGrid cols={{ base: 1, sm: 2, md: 3 }} spacing={12}>
            {view.goals.map((g) => {
              const pct = g.progressPct
              const body = (
                <>
                  <Text fz={15} fw={500} lh={1.25}>
                    {g.title}
                  </Text>
                  <Text size="xs" c="ink.5">
                    критерий: {g.criterion}
                  </Text>
                  <span style={{ display: 'flex', alignItems: 'center', gap: 8, width: '100%' }}>
                    <span style={{ flex: 1, height: 6, background: '#FFFFFF' }}>
                      <span
                        style={{
                          display: 'block',
                          height: 6,
                          width: `${pct}%`,
                          background: pct === 100 ? GREEN : BLUE,
                          transition: 'width .3s ease',
                        }}
                      />
                    </span>
                    <Text size="xs" fw={600}>
                      {pct}%
                    </Text>
                  </span>
                  {!own && g.lastUpdate && (
                    <Text size="xs" c="ink.5">
                      {g.lastUpdate.author.name}:{' '}
                      {g.lastUpdate.note || `${g.lastUpdate.progressPct}%`}
                    </Text>
                  )}
                </>
              )
              const style = {
                background: '#F4F4F4',
                padding: 14,
                display: 'flex',
                flexDirection: 'column' as const,
                gap: 6,
              }
              return own ? (
                <UnstyledButton
                  key={g.id}
                  style={style}
                  aria-label={`${g.title}: ${pct}%. Нажми, чтобы отметить прогресс`}
                  onClick={() => progress.mutate({ goalId: g.id, progressPct: nextPct(pct) })}
                >
                  {body}
                </UnstyledButton>
              ) : (
                <div key={g.id} style={style}>
                  {body}
                </div>
              )
            })}
          </SimpleGrid>
        )}

        {own ? (
          <Text size="xs" c="ink.5">
            Нажми на цель, чтобы отметить прогресс. По пятницам AI-buddy спросит, как дела, а
            наставник увидит обновления.
          </Text>
        ) : (
          view.canAgree &&
          !view.agreed && (
            <Button
              color="ink"
              loading={agree.isPending}
              onClick={() => agree.mutate()}
              style={{ alignSelf: 'flex-start' }}
            >
              Согласовать цели
            </Button>
          )
        )}
      </Stack>
    </Paper>
  )
}
