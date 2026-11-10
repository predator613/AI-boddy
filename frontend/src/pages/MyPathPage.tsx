import { useNavigate } from '@tanstack/react-router'
import { BuddyPanel } from '../components/BuddyPanel'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { StatusTag } from '../components/StatusTag'
import { GoalsCard } from '../components/path/GoalsCard'
import { PathScreen } from '../components/path/PathScreen'
import { useCompleteTask, useMyPath } from '../hooks/usePath'
import { Group } from '@mantine/core'

interface MyPathPageProps {
  stage: string | null
  onStage: (code: string | null) => void
  now?: Date
}

// Главная стажера: прогресс, этапы, следующий шаг и панель помощника.
export function MyPathPage({ stage, onStage, now }: MyPathPageProps) {
  const navigate = useNavigate()
  const pathQuery = useMyPath()
  const complete = useCompleteTask()

  if (pathQuery.isPending)
    return (
      <Page>
        <LoadingBlocks label="Загрузка пути" />
      </Page>
    )
  if (pathQuery.isError)
    return (
      <Page>
        <ErrorBlock error={pathQuery.error} onRetry={() => pathQuery.refetch()} />
      </Page>
    )

  const path = pathQuery.data
  const open = stage ? path.stages.find((s) => s.code === stage) : null
  const buddy = open ? { text: open.tip, suggestions: open.suggestions } : path.buddy

  return (
    <Page>
      <PathScreen
        path={path}
        openCode={stage}
        onOpen={onStage}
        now={now}
        goals={<GoalsCard />}
        busyTaskId={complete.isPending ? complete.variables : null}
        onComplete={(taskId) => complete.mutate(taskId)}
        onAsk={(task) =>
          navigate({ to: '/chat', search: { q: `Помоги: ${task.title}`, task: task.id } })
        }
        header={
          <PageHeader
            overline={`Мой путь · ${path.intern.teamName}${path.intern.mentor ? ` · наставник ${path.intern.mentor.name}` : ''}`}
            title={
              <>
                Привет, <Marker>{path.intern.name}!</Marker>
              </>
            }
            right={
              <Group gap={8}>
                <StatusTag bg="#FFFFFF" border="1px solid #D6D6D6" size="md">
                  Неделя {path.week.current} из {path.week.total}
                </StatusTag>
                <StatusTag bg="#FFFFFF" border="1px solid #D6D6D6" size="md">
                  Стажер · {path.intern.teamName}
                </StatusTag>
              </Group>
            }
          />
        }
      />
      <BuddyPanel text={buddy.text} suggestions={buddy.suggestions} />
    </Page>
  )
}
