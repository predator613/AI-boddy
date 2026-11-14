import { pathHints } from '../lib/buddyHints'
import { useNavigate } from '@tanstack/react-router'
import { BuddyPanel } from '../components/BuddyPanel'
import { Marker, PageHeader } from '../components/PageHeader'
import { Page } from '../components/Page'
import { ErrorBlock, LoadingBlocks } from '../components/QueryState'
import { WeekMeta } from '../components/path/WeekMeta'
import { GoalsCard } from '../components/path/GoalsCard'
import { PathScreen } from '../components/path/PathScreen'
import { useCompleteTask, useMyPath } from '../hooks/usePath'

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
  // Подсказки строятся из состояния: незакрытые задачи этапа, а не только текст из плана.
  const buddy = {
    text: open ? open.tip : path.buddy.text,
    suggestions: pathHints(path, open?.code),
  }

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
          navigate({
            to: '/chat',
            search: {
              q:
                typeof task.action?.payload.prompt === 'string'
                  ? task.action.payload.prompt
                  : `Помоги: ${task.title}`,
              task: task.id,
            },
          })
        }
        header={
          <PageHeader
            overline={`Мой путь · ${path.intern.teamName}`}
            title={
              <>
                Привет, <Marker>{path.intern.name}!</Marker>
              </>
            }
            right={<WeekMeta path={path} />}
          />
        }
      />
      <BuddyPanel text={buddy.text} suggestions={buddy.suggestions} />
    </Page>
  )
}
