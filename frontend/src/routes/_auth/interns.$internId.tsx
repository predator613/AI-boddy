import { Anchor } from '@mantine/core'
import { Link, createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { Marker, PageHeader } from '../../components/PageHeader'
import { Page } from '../../components/Page'
import { ErrorBlock, LoadingBlocks } from '../../components/QueryState'
import { PathScreen } from '../../components/path/PathScreen'
import { useInternPath, useSetInternTaskStatus } from '../../hooks/usePath'

export const Route = createFileRoute('/_auth/interns/$internId')({
  component: InternPathPage,
})

// Путь стажера глазами наставника: можно менять статусы задач.
function InternPathPage() {
  const { internId } = Route.useParams()
  const [stage, setStage] = useState<string | null>(null)
  const pathQuery = useInternPath(internId)
  const setStatus = useSetInternTaskStatus(internId)

  if (pathQuery.isPending)
    return (
      <Page>
        <LoadingBlocks label="Загрузка пути стажера" />
      </Page>
    )
  if (pathQuery.isError)
    return (
      <Page>
        <ErrorBlock error={pathQuery.error} onRetry={() => pathQuery.refetch()} />
      </Page>
    )
  const path = pathQuery.data
  return (
    <Page maxWidth={1240}>
      <PathScreen
        path={path}
        openCode={stage}
        onOpen={setStage}
        busyTaskId={setStatus.isPending ? setStatus.variables?.taskId : null}
        onSetStatus={(taskId, status) => setStatus.mutate({ taskId, status })}
        header={
          <PageHeader
            overline={`Путь стажера · ${path.intern.teamName}`}
            title={<Marker color="#DFCDF3">{path.intern.name}</Marker>}
            right={
              <Anchor component={Link} to="/" c="ink.6">
                ← Все стажеры
              </Anchor>
            }
          />
        }
      />
    </Page>
  )
}
