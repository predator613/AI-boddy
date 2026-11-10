import { Alert, Button, Skeleton, Stack } from '@mantine/core'
import { errorText } from '../lib/errors'

export function LoadingBlocks({ label = 'Загрузка' }: { label?: string }) {
  return (
    <Stack gap="lg" aria-label={label} aria-busy="true" style={{ flex: 1 }}>
      <Skeleton height={56} width="50%" />
      <Skeleton height={260} radius="lg" />
      <Skeleton height={156} radius="lg" />
    </Stack>
  )
}

export function ErrorBlock({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  return (
    <Alert color="red" variant="light" role="alert" title="Что-то пошло не так" style={{ flex: 1 }}>
      {errorText(error)}
      {onRetry && (
        <div>
          <Button mt="sm" size="xs" variant="outline" color="ink" onClick={onRetry}>
            Повторить
          </Button>
        </div>
      )}
    </Alert>
  )
}
