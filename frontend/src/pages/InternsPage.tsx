import { Paper, SimpleGrid, Stack, Text } from '@mantine/core'
import { Link } from '@tanstack/react-router'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import { useInterns } from '../hooks/usePath'
import { useAuthStore } from '../store/authStore'
import { MentorStatsStrip } from '../components/mentor/MentorStatsStrip'

// Главная наставника и руководителя: стажеры и их прогресс.
export function InternsPage() {
  const user = useAuthStore((s) => s.user)
  const interns = useInterns()
  return (
    <Page maxWidth={1240}>
      <Stack gap="lg" style={{ flex: 1, minWidth: 0 }}>
        <PageHeader
          overline={`${user?.role === 'lead' ? 'Руководитель' : 'Наставник'} · ${user?.teamName ?? ''}`}
          title={
            <>
              Мои <Marker color="#965EEB">стажеры</Marker>
            </>
          }
        />
        <MentorStatsStrip />
        {interns.isPending && <LoadingBlocks label="Загрузка стажеров" />}
        {interns.isError && <ErrorBlock error={interns.error} onRetry={() => interns.refetch()} />}
        {interns.data && interns.data.length === 0 && (
          <Paper radius="lg" p="lg">
            <Text>Пока нет стажеров. Когда в команду придет стажер, он появится здесь.</Text>
          </Paper>
        )}
        {interns.data && interns.data.length > 0 && (
          <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="lg">
            {interns.data.map((i) => (
              <Paper
                key={i.id}
                renderRoot={(p) => (
                  <Link {...p} to="/interns/$internId" params={{ internId: i.id }} />
                )}
                radius="lg"
                p="lg"
                className="ab-lift"
                style={{
                  textDecoration: 'none',
                  color: 'inherit',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 12,
                }}
                aria-label={`${i.name}: ${i.progress.percent}%`}
              >
                <Text fz={24} fw={500}>
                  {i.name}
                </Text>
                <Text size="sm" c="ink.5">
                  {i.teamName} · неделя {i.week}
                  {i.mentor ? ` · наставник ${i.mentor.name}` : ''}
                </Text>
                <div style={{ height: 8, background: '#F4F4F4' }}>
                  <div
                    style={{ width: `${i.progress.percent}%`, height: 8, background: '#04E061' }}
                  />
                </div>
                <Text size="sm">
                  {i.progress.percent}% · {i.progress.done} из {i.progress.total} задач
                </Text>
                {i.currentStage && (
                  <StatusTag bg="#BADAFB" style={{ alignSelf: 'flex-start' }}>
                    Этап {i.currentStageCode} · {i.currentStage}
                  </StatusTag>
                )}
              </Paper>
            ))}
          </SimpleGrid>
        )}
      </Stack>
    </Page>
  )
}
